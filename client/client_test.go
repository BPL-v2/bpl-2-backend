package client

import (
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ========== PriorityMutex ==========

func TestPriorityMutex_LockUnlock(t *testing.T) {
	m := NewPriorityMutex()
	m.Lock()
	m.Unlock()
}

func TestPriorityMutex_PriorityLockUnlock(t *testing.T) {
	m := NewPriorityMutex()
	m.PriorityLock()
	m.PriorityUnlock()
}

func TestPriorityMutex_PrioritySkipsQueue(t *testing.T) {
	m := NewPriorityMutex()

	var order []string
	var orderMu sync.Mutex
	record := func(name string) {
		orderMu.Lock()
		order = append(order, name)
		orderMu.Unlock()
	}

	var wg sync.WaitGroup

	// Hold the mutex via Lock(). This holds both lowPriorityAccess and
	// dataMutex (nextToAccess is released during Lock).
	m.Lock()

	// LP1 and LP2 both block on lowPriorityAccess (held by us).
	wg.Go(func() {
		m.Lock()
		record("LP1")
		m.Unlock()
	})
	time.Sleep(20 * time.Millisecond)

	wg.Go(func() {
		m.Lock()
		record("LP2")
		m.Unlock()
	})
	time.Sleep(20 * time.Millisecond)

	// PRIO skips lowPriorityAccess entirely — it only needs nextToAccess
	// (free) then dataMutex (held by us). So it waits directly on dataMutex,
	// ahead of LP1/LP2 who are still stuck on lowPriorityAccess.
	wg.Go(func() {
		m.PriorityLock()
		record("PRIO")
		m.PriorityUnlock()
	})
	time.Sleep(20 * time.Millisecond)

	// Release: dataMutex.Unlock lets PRIO in immediately.
	// lowPriorityAccess.Unlock lets LP1 start its Lock chain.
	// PRIO finishes first despite starting last.
	m.Unlock()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for goroutines")
	}

	assert.Equal(t, []string{"PRIO", "LP1", "LP2"}, order)
}

// ========== Policy ==========

func TestPolicy_CurrentHits(t *testing.T) {
	p := Policy{MaxHits: 5, Period: 10 * time.Second}

	now := time.Now()
	times := []time.Time{
		now.Add(-20 * time.Second), // outside period
		now.Add(-5 * time.Second),  // inside
		now.Add(-2 * time.Second),  // inside
		now.Add(-1 * time.Second),  // inside
	}
	assert.Equal(t, 3, p.CurrentHits(times))
}

func TestPolicy_CurrentHits_Empty(t *testing.T) {
	p := Policy{MaxHits: 5, Period: 10 * time.Second}
	assert.Equal(t, 0, p.CurrentHits(nil))
}

func TestPolicy_IsViolated_True(t *testing.T) {
	p := Policy{MaxHits: 2, Period: 10 * time.Second}
	now := time.Now()
	times := []time.Time{now.Add(-1 * time.Second), now.Add(-2 * time.Second)}
	assert.True(t, p.IsViolated(times))
}

func TestPolicy_IsViolated_False(t *testing.T) {
	p := Policy{MaxHits: 5, Period: 10 * time.Second}
	now := time.Now()
	times := []time.Time{now.Add(-1 * time.Second)}
	assert.False(t, p.IsViolated(times))
}

// ========== AsyncHttpClient helpers ==========

func newTestClient() *AsyncHttpClient {
	u, _ := url.Parse("http://example.com")
	return NewAsyncHttpClient(u, "test-agent", 10.0)
}

func TestGetRules(t *testing.T) {
	c := newTestClient()
	headers := http.Header{}
	headers.Set("X-Rate-Limit-Rules", "rule1,rule2,rule3")
	assert.Equal(t, []string{"rule1", "rule2", "rule3"}, c.getRules(headers))
}

func TestGetRules_Empty(t *testing.T) {
	c := newTestClient()
	headers := http.Header{}
	assert.Nil(t, c.getRules(headers))
}

func TestParsePolicies(t *testing.T) {
	c := newTestClient()
	headers := http.Header{}
	headers.Set("X-Rate-Limit-MyRule", "30:60:120,100:120:900")
	headers.Set("X-Rate-Limit-MyRule-State", "5:60:0,20:120:0")
	result := c.parsePolicies("MyRule", headers)
	require.Len(t, result, 2)
	for policy, currentHits := range result {
		if policy.MaxHits == 30 {
			assert.Equal(t, 5, currentHits)
		} else if policy.MaxHits == 100 {
			assert.Equal(t, 20, currentHits)
		}
	}
}

func TestParsePolicies_EmptyHeaders(t *testing.T) {
	c := newTestClient()
	result := c.parsePolicies("Missing", http.Header{})
	assert.Nil(t, result)
}

func TestCanMakeRequest_FirstRequest(t *testing.T) {
	c := newTestClient()
	key := RequestKey{Token: "tok", Endpoint: "ep"}
	// First request always allowed (creates dummy policy)
	assert.True(t, c.canMakeRequest(key))
}

func TestCanMakeRequest_PolicyViolated(t *testing.T) {
	c := newTestClient()
	key := RequestKey{Token: "tok", Endpoint: "ep"}
	c.rateLimitPolicies[key] = map[string][]Policy{
		"rule": {{MaxHits: 1, Period: 10 * time.Second}},
	}
	now := time.Now()
	c.requestTimestamps[key] = []time.Time{now}
	assert.False(t, c.canMakeRequest(key))
}

func TestIpIsRateLimited_NoRequests(t *testing.T) {
	c := newTestClient()
	assert.False(t, c.ipIsRateLimited())
}

func TestIpIsRateLimited_RecentRequest(t *testing.T) {
	c := newTestClient()
	key := RequestKey{Token: "tok", Endpoint: "ep"}
	c.requestTimestamps[key] = []time.Time{time.Now()}
	assert.True(t, c.ipIsRateLimited())
}
