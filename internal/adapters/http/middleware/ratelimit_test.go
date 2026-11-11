package middleware

import (
	nethttp "net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// fakeClock is a manually advanced, concurrency-safe clock.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func newLimitedRouter(l *rateLimiter) *gin.Engine {
	r := gin.New()
	r.POST("/login", l.handle, func(c *gin.Context) { c.Status(nethttp.StatusOK) })
	return r
}

func doPost(r *gin.Engine, ip string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(nethttp.MethodPost, "/login", nil)
	req.RemoteAddr = ip + ":40000"
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestRateLimit_OverLimit_Returns429WithRetryAfter(t *testing.T) {
	clock := newFakeClock()
	r := newLimitedRouter(newRateLimiter(3, time.Minute, clock.Now))

	for i := range 3 {
		if rec := doPost(r, "10.0.0.1"); rec.Code != nethttp.StatusOK {
			t.Fatalf("request %d: status = %d, want 200", i+1, rec.Code)
		}
	}

	rec := doPost(r, "10.0.0.1")
	assertErrorResponse(t, rec, nethttp.StatusTooManyRequests, "RATE_LIMITED", "too many requests")
	// One token is regained every 20s.
	if got := rec.Header().Get("Retry-After"); got != "20" {
		t.Fatalf("Retry-After = %q, want 20", got)
	}
}

func TestRateLimit_TwoClients_HaveIndependentLimits(t *testing.T) {
	clock := newFakeClock()
	r := newLimitedRouter(newRateLimiter(2, time.Minute, clock.Now))

	for range 2 {
		doPost(r, "10.0.0.1")
	}
	if rec := doPost(r, "10.0.0.1"); rec.Code != nethttp.StatusTooManyRequests {
		t.Fatalf("first client: status = %d, want 429", rec.Code)
	}
	for i := range 2 {
		if rec := doPost(r, "10.0.0.2"); rec.Code != nethttp.StatusOK {
			t.Fatalf("second client request %d: status = %d, want 200", i+1, rec.Code)
		}
	}
}

func TestRateLimit_AfterWindow_AllowsFullBurstAgain(t *testing.T) {
	clock := newFakeClock()
	r := newLimitedRouter(newRateLimiter(3, time.Minute, clock.Now))

	for range 3 {
		doPost(r, "10.0.0.1")
	}
	if rec := doPost(r, "10.0.0.1"); rec.Code != nethttp.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", rec.Code)
	}

	clock.Advance(time.Minute)

	for i := range 3 {
		if rec := doPost(r, "10.0.0.1"); rec.Code != nethttp.StatusOK {
			t.Fatalf("request %d after window: status = %d, want 200", i+1, rec.Code)
		}
	}
	if rec := doPost(r, "10.0.0.1"); rec.Code != nethttp.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", rec.Code)
	}
}

func TestRateLimit_PartialWait_RegainsOneRequest(t *testing.T) {
	clock := newFakeClock()
	r := newLimitedRouter(newRateLimiter(3, time.Minute, clock.Now))

	for range 3 {
		doPost(r, "10.0.0.1")
	}
	clock.Advance(15 * time.Second)
	rec := doPost(r, "10.0.0.1")
	if rec.Code != nethttp.StatusTooManyRequests || rec.Header().Get("Retry-After") != "5" {
		t.Fatalf("status = %d, Retry-After = %q, want 429 and 5", rec.Code, rec.Header().Get("Retry-After"))
	}

	clock.Advance(5 * time.Second)
	if rec := doPost(r, "10.0.0.1"); rec.Code != nethttp.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec := doPost(r, "10.0.0.1"); rec.Code != nethttp.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", rec.Code)
	}
}

func TestRateLimit_IdleClients_AreEvicted(t *testing.T) {
	clock := newFakeClock()
	l := newRateLimiter(2, time.Minute, clock.Now)
	r := newLimitedRouter(l)

	doPost(r, "10.0.0.1")
	doPost(r, "10.0.0.2")
	clock.Advance(time.Minute)
	doPost(r, "10.0.0.3")

	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.buckets) != 1 {
		t.Fatalf("buckets = %d, want 1 (idle clients evicted)", len(l.buckets))
	}
	if _, ok := l.buckets["10.0.0.3"]; !ok {
		t.Fatal("active client was evicted")
	}
}

func TestRateLimit_ConcurrentRequests_AllowExactlyLimit(t *testing.T) {
	clock := newFakeClock()
	r := newLimitedRouter(newRateLimiter(10, time.Minute, clock.Now))

	var ok, limited atomic.Int64
	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			switch doPost(r, "10.0.0.1").Code {
			case nethttp.StatusOK:
				ok.Add(1)
			case nethttp.StatusTooManyRequests:
				limited.Add(1)
			}
		}()
	}
	wg.Wait()

	if ok.Load() != 10 || limited.Load() != 40 {
		t.Fatalf("ok = %d, limited = %d, want 10 and 40", ok.Load(), limited.Load())
	}
}

func TestRateLimit_NonPositiveConfig_Panics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("RateLimit(0, ...) did not panic")
		}
	}()
	RateLimit(0, time.Minute)
}

func doPostIPv6(r *gin.Engine, ip string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(nethttp.MethodPost, "/login", nil)
	req.RemoteAddr = "[" + ip + "]:40000"
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestRateLimit_IPv6SameSlash64_SharesLimit(t *testing.T) {
	r := newLimitedRouter(newRateLimiter(2, time.Minute, newFakeClock().Now))

	doPostIPv6(r, "2001:db8:1:2::1")
	doPostIPv6(r, "2001:db8:1:2::2")
	if rec := doPostIPv6(r, "2001:db8:1:2:ffff::3"); rec.Code != nethttp.StatusTooManyRequests {
		t.Fatalf("third request from the same /64: status = %d, want 429", rec.Code)
	}
	if rec := doPostIPv6(r, "2001:db8:1:3::1"); rec.Code != nethttp.StatusOK {
		t.Fatalf("request from another /64: status = %d, want 200", rec.Code)
	}
}

func TestRateLimit_TooManyClients_RefusesNewOnesUntilIdle(t *testing.T) {
	clock := newFakeClock()
	l := newRateLimiter(5, time.Minute, clock.Now)
	l.maxKeys = 2
	r := newLimitedRouter(l)

	doPost(r, "10.0.0.1")
	doPost(r, "10.0.0.2")
	if rec := doPost(r, "10.0.0.3"); rec.Code != nethttp.StatusTooManyRequests {
		t.Fatalf("new client over the cap: status = %d, want 429", rec.Code)
	}
	if rec := doPost(r, "10.0.0.1"); rec.Code != nethttp.StatusOK {
		t.Fatalf("known client at the cap: status = %d, want 200", rec.Code)
	}

	clock.Advance(time.Minute)
	if rec := doPost(r, "10.0.0.3"); rec.Code != nethttp.StatusOK {
		t.Fatalf("new client once others are idle: status = %d, want 200", rec.Code)
	}
}

func TestClientKey_VariousAddresses_GroupsIPv6BySlash64(t *testing.T) {
	tests := map[string]string{
		"10.1.2.3":          "10.1.2.3",
		"2001:db8:1:2::abc": "2001:db8:1:2::/64",
		"::ffff:10.1.2.3":   "::ffff:10.1.2.3",
		"not-an-ip":         "not-an-ip",
	}
	for ip, want := range tests {
		if got := clientKey(ip); got != want {
			t.Errorf("clientKey(%q) = %q, want %q", ip, got, want)
		}
	}
}
