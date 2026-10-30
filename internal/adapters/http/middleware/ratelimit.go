package middleware

import (
	"math"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/adapters/http/response"
	"github.com/traily-org/server/internal/pkg/apperror"
)

// errTooManyRequests is returned when a client exceeds its rate limit.
var errTooManyRequests = apperror.RateLimited("too many requests")

// defaultMaxClients caps the number of clients tracked at once, so that a
// flood of distinct addresses cannot grow memory without bound.
const defaultMaxClients = 100_000

// ipv6PrefixBits is the prefix IPv6 clients are grouped by: a single
// subscriber usually owns a whole /64 and could otherwise rotate addresses.
const ipv6PrefixBits = 64

// RateLimit returns a middleware that allows each client IP (as reported by
// gin's ClientIP) at most requests requests per period, using an in-memory
// token bucket: a client can burst up to requests requests, then regains one
// request every per/requests. Requests over the limit get 429 RATE_LIMITED
// with a Retry-After header in seconds.
//
// IPv6 clients are grouped by /64. State is kept in memory, so limits are per
// process. Idle clients are evicted periodically, and when too many clients
// are tracked at once new ones are refused until some go idle. RateLimit
// panics if requests or per is not positive, since that is a configuration
// error.
func RateLimit(requests int, per time.Duration) gin.HandlerFunc {
	return newRateLimiter(requests, per, time.Now).handle
}

// bucket is the token bucket of a single client.
type bucket struct {
	tokens float64
	last   time.Time
}

// rateLimiter is a concurrency-safe set of token buckets keyed by client IP.
type rateLimiter struct {
	capacity  float64
	interval  time.Duration // time to regain one token
	idleAfter time.Duration // idle time after which a bucket is full again
	maxKeys   int           // maximum number of tracked clients
	now       func() time.Time

	mu          sync.Mutex
	buckets     map[string]*bucket
	nextCleanup time.Time
}

// newRateLimiter builds a rateLimiter reading the time from now, which tests
// replace with a fake clock.
func newRateLimiter(requests int, per time.Duration, now func() time.Time) *rateLimiter {
	if requests <= 0 || per <= 0 {
		panic("middleware: RateLimit requires a positive number of requests and period")
	}
	interval := per / time.Duration(requests)
	if interval <= 0 {
		interval = 1
	}
	return &rateLimiter{
		capacity:    float64(requests),
		interval:    interval,
		idleAfter:   per,
		maxKeys:     defaultMaxClients,
		now:         now,
		buckets:     make(map[string]*bucket),
		nextCleanup: now().Add(per),
	}
}

// handle is the gin middleware.
func (l *rateLimiter) handle(c *gin.Context) {
	allowed, retryAfter := l.allow(clientKey(c.ClientIP()))
	if !allowed {
		c.Header("Retry-After", strconv.Itoa(retryAfterSeconds(retryAfter)))
		response.Error(c, errTooManyRequests)
		return
	}
	c.Next()
}

// allow consumes a token for key. When none is available it reports false and
// how long the client must wait for the next token.
func (l *rateLimiter) allow(key string) (bool, time.Duration) {
	now := l.now()

	l.mu.Lock()
	defer l.mu.Unlock()

	l.cleanup(now, false)

	b, ok := l.buckets[key]
	if !ok {
		if len(l.buckets) >= l.maxKeys {
			l.cleanup(now, true)
			if len(l.buckets) >= l.maxKeys {
				return false, l.interval
			}
		}
		b = &bucket{tokens: l.capacity, last: now}
		l.buckets[key] = b
	} else {
		l.refill(b, now)
	}

	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	missing := 1 - b.tokens
	return false, time.Duration(missing * float64(l.interval))
}

// refill adds the tokens earned since the last request, up to capacity.
func (l *rateLimiter) refill(b *bucket, now time.Time) {
	elapsed := now.Sub(b.last)
	if elapsed <= 0 {
		return
	}
	b.tokens = math.Min(l.capacity, b.tokens+float64(elapsed)/float64(l.interval))
	b.last = now
}

// cleanup evicts, at most once per period unless forced, the buckets idle
// long enough to be full again: dropping them does not change their client's
// limit. The caller must hold l.mu.
func (l *rateLimiter) cleanup(now time.Time, force bool) {
	if !force && now.Before(l.nextCleanup) {
		return
	}
	for key, b := range l.buckets {
		if now.Sub(b.last) >= l.idleAfter {
			delete(l.buckets, key)
		}
	}
	l.nextCleanup = now.Add(l.idleAfter)
}

// retryAfterSeconds rounds d up to whole seconds, with a minimum of 1, as
// expected by the Retry-After header.
func retryAfterSeconds(d time.Duration) int {
	s := int(math.Ceil(d.Seconds()))
	if s < 1 {
		return 1
	}
	return s
}

// clientKey returns the rate-limit key of a client IP: the address itself for
// IPv4, its /64 prefix for IPv6.
func clientKey(ip string) string {
	parsed := net.ParseIP(ip)
	if parsed == nil || parsed.To4() != nil {
		return ip
	}
	return parsed.Mask(net.CIDRMask(ipv6PrefixBits, 128)).String() + "/64"
}
