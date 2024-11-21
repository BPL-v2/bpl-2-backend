package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
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
	baseURL              string
	maxRequestsPerSecond float64
	retry                bool
	userAgent            string
	client               *http.Client
}

func NewAsyncHttpClient(baseURL, userAgent string, maxRequestsPerSecond float64, retry bool) *AsyncHttpClient {
	return &AsyncHttpClient{
		requestTimestamps:    make(map[RequestKey][]time.Time),
		rateLimitPolicies:    make(map[RequestKey]map[string][]Policy),
		baseURL:              baseURL,
		maxRequestsPerSecond: maxRequestsPerSecond,
		retry:                retry,
		userAgent:            userAgent,
		client:               &http.Client{},
	}
}

func (c *AsyncHttpClient) SendRequest(ctx context.Context, endpoint, token string, ignoreBaseURL bool, method string, headers map[string]string) (*http.Response, error) {
	if headers == nil {
		headers = make(map[string]string)
	}
	headers["User-Agent"] = c.userAgent
	if token != "" {
		headers["Authorization"] = "Bearer " + token
	} else {
		token = "IP"
	}
	key := RequestKey{Token: token, Endpoint: endpoint}
	if method == "" {
		method = "GET"
	}

	if err := c.waitUntilRequestAllowed(ctx, key); err != nil {
		return nil, err
	}

	url := endpoint
	if !ignoreBaseURL {
		url = fmt.Sprintf("%s/%s", c.baseURL, endpoint)
	}

	c.mu.Lock()
	c.requestTimestamps[key] = append(c.requestTimestamps[key], time.Now())
	c.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
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

	if resp.StatusCode == http.StatusTooManyRequests && c.retry {
		retryAfter, _ := strconv.Atoi(resp.Header.Get("Retry-After"))
		if retryAfter == 0 {
			retryAfter = 1
		}
		time.Sleep(time.Duration(retryAfter) * time.Second)
		return c.SendRequest(ctx, endpoint, token, ignoreBaseURL, method, headers)
	}

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

func main() {
	client := NewAsyncHttpClient("https://www.pathofexile.com/api", "user-agent", 10, true)
	token := "xxxxxxx"
	ctx := context.Background()

	responseCodes := make(chan int)
	characters := []string{
		"BaldNudist",
		"AtrocityCommitter",
		"CastOnCringePortal",
		"CastOnCringeYiks",
	}
	var wg sync.WaitGroup
	wg.Add(len(characters))

	for _, character := range characters {
		go func(char string) {
			defer wg.Done()
			res, err := client.SendRequest(ctx, fmt.Sprintf("character/%s", char), token, false, "GET", nil)
			if err != nil {
				log.Fatal(err)
			} else {
				defer res.Body.Close()
				_, err := io.ReadAll(res.Body)
				if err != nil {
					log.Fatal(err)
				} else {
					responseCodes <- res.StatusCode
				}
			}
		}(character)
	}
	go func() {
		for response := range responseCodes {
			fmt.Println("Response status:", response)
		}
	}()

	wg.Wait()

}
