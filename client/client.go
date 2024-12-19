package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Policy struct {
	MaxHits int
	Period  time.Duration
}

func (p *Policy) CurrentHits(requestTimes []time.Time) int {
	periodStart := time.Now().Add(-p.Period)
	count := 0
	for _, t := range requestTimes {
		if t.After(periodStart) {
			count++
		}
	}
	return count
}

func (p *Policy) IsViolated(requestTimes []time.Time) bool {
	return p.CurrentHits(requestTimes) >= p.MaxHits
}

type RequestKey struct {
	Token    string
	Endpoint string
}

type AsyncHttpClient struct {
	mu                   sync.Mutex
	requestTimestamps    map[RequestKey][]time.Time
	rateLimitPolicies    map[RequestKey]map[string][]Policy
	baseURL              *url.URL
	maxRequestsPerSecond float64
	userAgent            string
	client               *http.Client
}

func NewAsyncHttpClient(baseURL *url.URL, userAgent string, maxRequestsPerSecond float64) *AsyncHttpClient {
	return &AsyncHttpClient{
		requestTimestamps:    make(map[RequestKey][]time.Time),
		rateLimitPolicies:    make(map[RequestKey]map[string][]Policy),
		baseURL:              baseURL,
		maxRequestsPerSecond: maxRequestsPerSecond,
		userAgent:            userAgent,
		client:               &http.Client{},
	}
}

type RequestArgs struct {
	Endpoint      string
	Token         string
	Method        string
	QueryParams   map[string]string
	Body          *strings.Reader
	BodyRaw       any
	Headers       map[string]string
	IgnoreBaseURL bool
}

func (c *AsyncHttpClient) SendRequest(
	ctx context.Context,
	requestArgs RequestArgs,
) (*http.Response, error) {
	err := error(nil)
	var headers map[string]string
	if requestArgs.Headers == nil {
		headers = map[string]string{}
	} else {
		headers = requestArgs.Headers
	}

	headers["User-Agent"] = c.userAgent

	token := requestArgs.Token
	if token != "" {
		headers["Authorization"] = "Bearer " + token
	} else {
		token = "IP"
	}
	key := RequestKey{Token: token, Endpoint: requestArgs.Endpoint}

	method := requestArgs.Method
	if method == "" {
		method = "GET"
	}
	if err := c.waitUntilRequestAllowed(ctx, key); err != nil {
		return nil, err
	}
	var requestUrl *url.URL
	if requestArgs.IgnoreBaseURL {
		requestUrl, err = url.Parse(requestArgs.Endpoint)
		if err != nil {
			return nil, err
		}
	} else {
		requestUrl = c.baseURL.ResolveReference(&url.URL{Path: c.baseURL.Path + "/" + requestArgs.Endpoint})
	}
	if requestArgs.QueryParams != nil {
		query := requestUrl.Query()
		for k, v := range requestArgs.QueryParams {
			query.Add(k, v)
		}
		requestUrl.RawQuery = query.Encode()
	}
	c.mu.Lock()
	c.requestTimestamps[key] = append(c.requestTimestamps[key], time.Now())
	c.mu.Unlock()

	req := &http.Request{}
	if requestArgs.Body != nil {
		req, err = http.NewRequestWithContext(ctx, method, requestUrl.String(), requestArgs.Body)
	} else {
		req, err = http.NewRequestWithContext(ctx, method, requestUrl.String(), nil)
	}

	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	c.adjustPolicies(key, resp.Header)
	return resp, nil
}

func (c *AsyncHttpClient) waitUntilRequestAllowed(ctx context.Context, key RequestKey) error {
	for {
		canMakeRequest := c.canMakeRequest(key)

		if canMakeRequest {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func (c *AsyncHttpClient) adjustPolicies(key RequestKey, headers http.Header) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.rateLimitPolicies[key], "dummy")

	now := time.Now()
	timestamps := c.requestTimestamps[key]
	newPolicies := make(map[string][]Policy)
	for _, rule := range c.getRules(headers) {
		var policies []Policy
		for policy, currentHits := range c.parsePolicies(rule, headers) {
			policies = append(policies, policy)
			startTime := now.Add(-policy.Period)
			trackedHits := 0
			for _, t := range timestamps {
				if t.After(startTime) {
					trackedHits++
				}
			}
			missingHits := currentHits - trackedHits
			for i := 0; i < missingHits; i++ {
				timestamps = append(timestamps, now)
			}
		}
		newPolicies[rule] = policies
	}
	c.rateLimitPolicies[key] = newPolicies
}

func (c *AsyncHttpClient) canMakeRequest(key RequestKey) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ipIsRateLimited() {
		return false
	}
	if _, ok := c.rateLimitPolicies[key]; !ok {
		c.rateLimitPolicies[key] = map[string][]Policy{
			"dummy": {{MaxHits: 1, Period: 9999999 * time.Second}},
		}
		return true
	}

	for _, policies := range c.rateLimitPolicies[key] {
		for _, policy := range policies {
			if policy.IsViolated(c.requestTimestamps[key]) {
				return false
			}
		}
	}
	return true
}

func (c *AsyncHttpClient) ipIsRateLimited() bool {
	start := time.Now().Add(-time.Second / time.Duration(c.maxRequestsPerSecond))
	requestsTimePeriod := 0
	for _, timestamps := range c.requestTimestamps {
		for _, t := range timestamps {
			if t.After(start) {
				requestsTimePeriod++
			}
		}
	}
	return requestsTimePeriod >= 1
}

func (c *AsyncHttpClient) getRules(headers http.Header) []string {
	rules := headers.Get("X-Rate-Limit-Rules")
	if rules == "" {
		return nil
	}
	return strings.Split(rules, ",")
}

func (c *AsyncHttpClient) parsePolicies(rule string, headers http.Header) map[Policy]int {
	limitHeader := fmt.Sprintf("X-Rate-Limit-%s", rule)
	stateHeader := fmt.Sprintf("X-Rate-Limit-%s-State", rule)
	limits := headers.Get(limitHeader)
	states := headers.Get(stateHeader)
	if limits == "" || states == "" {
		return nil
	}

	policies := make(map[Policy]int)
	limitParts := strings.Split(limits, ",")
	stateParts := strings.Split(states, ",")
	for i := range limitParts {
		limit := strings.Split(limitParts[i], ":")
		state := strings.Split(stateParts[i], ":")
		maxHits, _ := strconv.Atoi(limit[0])
		period, _ := strconv.Atoi(limit[1])
		currentHits, _ := strconv.Atoi(state[0])
		policies[Policy{MaxHits: maxHits, Period: time.Duration(period) * time.Second}] = currentHits
	}
	return policies
}
