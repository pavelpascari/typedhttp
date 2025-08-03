package ratelimit

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Common errors
var (
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
)

// RateLimiter interface defines the contract for rate limiting implementations
type RateLimiter interface {
	Allow(key string) bool
}

// TokenBucket represents a token bucket for rate limiting
type TokenBucket struct {
	capacity     int
	tokens       int
	refillTokens int
	lastRefill   time.Time
	interval     time.Duration
	mu           sync.Mutex
}

// TokenBucketConfig holds token bucket configuration
type TokenBucketConfig struct {
	Capacity       int
	RefillInterval time.Duration
	RefillTokens   int
	BurstCapacity  int
}

// TokenBucketRateLimiter implements token bucket rate limiting
type TokenBucketRateLimiter struct {
	config  TokenBucketConfig
	buckets map[string]*TokenBucket
	mu      sync.RWMutex
}

// TokenBucketOption configures token bucket rate limiter
type TokenBucketOption func(*TokenBucketConfig)

// WithRefillTokens sets the number of tokens to refill per interval
func WithRefillTokens(tokens int) TokenBucketOption {
	return func(c *TokenBucketConfig) {
		c.RefillTokens = tokens
	}
}

// WithBurstCapacity sets the burst capacity
func WithBurstCapacity(capacity int) TokenBucketOption {
	return func(c *TokenBucketConfig) {
		c.BurstCapacity = capacity
	}
}

// NewTokenBucketRateLimiter creates a new token bucket rate limiter
func NewTokenBucketRateLimiter(capacity int, refillInterval time.Duration, opts ...TokenBucketOption) *TokenBucketRateLimiter {
	config := TokenBucketConfig{
		Capacity:       capacity,
		RefillInterval: refillInterval,
		RefillTokens:   1,
		BurstCapacity:  capacity,
	}

	for _, opt := range opts {
		opt(&config)
	}

	return &TokenBucketRateLimiter{
		config:  config,
		buckets: make(map[string]*TokenBucket),
	}
}

// GetConfig returns the rate limiter configuration
func (r *TokenBucketRateLimiter) GetConfig() TokenBucketConfig {
	return r.config
}

// Allow checks if a request should be allowed
func (r *TokenBucketRateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	bucket, exists := r.buckets[key]
	if !exists {
		bucket = &TokenBucket{
			capacity:     r.config.Capacity,
			tokens:       r.config.Capacity,
			refillTokens: r.config.RefillTokens,
			lastRefill:   time.Now(),
			interval:     r.config.RefillInterval,
		}
		r.buckets[key] = bucket
	}

	return bucket.consume()
}

// consume tries to consume a token from the bucket
func (b *TokenBucket) consume() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Refill tokens based on elapsed time
	now := time.Now()
	elapsed := now.Sub(b.lastRefill)
	if elapsed >= b.interval {
		tokensToAdd := int(elapsed/b.interval) * b.refillTokens
		b.tokens = minInt(b.capacity, b.tokens+tokensToAdd)
		b.lastRefill = now
	}

	// Try to consume a token
	if b.tokens > 0 {
		b.tokens--

		return true
	}

	return false
}

// IPBasedConfig holds IP-based rate limiter configuration
type IPBasedConfig struct {
	RequestsPerWindow int
	Window            time.Duration
	MaxIPs            int
	CleanupInterval   time.Duration
	Whitelist         []string
	Blacklist         []string
}

// IPEntry tracks requests for an IP address
type IPEntry struct {
	requests []time.Time
	mu       sync.Mutex
}

// IPBasedRateLimiter implements IP-based rate limiting
type IPBasedRateLimiter struct {
	config      IPBasedConfig
	ipEntries   map[string]*IPEntry
	whitelist   map[string]bool
	blacklist   map[string]bool
	mu          sync.RWMutex
	stopCleanup chan bool
}

// IPBasedOption configures IP-based rate limiter
type IPBasedOption func(*IPBasedConfig)

// WithMaxIPs sets the maximum number of tracked IPs
func WithMaxIPs(maxIPs int) IPBasedOption {
	return func(c *IPBasedConfig) {
		c.MaxIPs = maxIPs
	}
}

// WithCleanupInterval sets the cleanup interval for expired entries
func WithCleanupInterval(interval time.Duration) IPBasedOption {
	return func(c *IPBasedConfig) {
		c.CleanupInterval = interval
	}
}

// WithWhitelist sets the IP whitelist
func WithWhitelist(ips []string) IPBasedOption {
	return func(c *IPBasedConfig) {
		c.Whitelist = ips
	}
}

// WithBlacklist sets the IP blacklist
func WithBlacklist(ips []string) IPBasedOption {
	return func(c *IPBasedConfig) {
		c.Blacklist = ips
	}
}

// NewIPBasedRateLimiter creates a new IP-based rate limiter
func NewIPBasedRateLimiter(requestsPerWindow int, window time.Duration, opts ...IPBasedOption) *IPBasedRateLimiter {
	config := IPBasedConfig{
		RequestsPerWindow: requestsPerWindow,
		Window:            window,
		MaxIPs:            1000,
		CleanupInterval:   5 * time.Minute,
	}

	for _, opt := range opts {
		opt(&config)
	}

	limiter := &IPBasedRateLimiter{
		config:      config,
		ipEntries:   make(map[string]*IPEntry),
		whitelist:   make(map[string]bool),
		blacklist:   make(map[string]bool),
		stopCleanup: make(chan bool),
	}

	// Build whitelist and blacklist maps
	for _, ip := range config.Whitelist {
		limiter.whitelist[ip] = true
	}
	for _, ip := range config.Blacklist {
		limiter.blacklist[ip] = true
	}

	// Start cleanup goroutine
	go limiter.cleanup()

	return limiter
}

// GetConfig returns the rate limiter configuration
func (r *IPBasedRateLimiter) GetConfig() IPBasedConfig {
	return r.config
}

// Allow checks if a request should be allowed for the given key
func (r *IPBasedRateLimiter) Allow(key string) bool {
	return r.AllowIP(key)
}

// AllowIP checks if a request should be allowed for the given IP
func (r *IPBasedRateLimiter) AllowIP(ip string) bool {
	// Extract IP from address string (remove port if present)
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}

	// Check blacklist first
	if r.blacklist[ip] {
		return false
	}

	// Check whitelist
	if r.whitelist[ip] {
		return true
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	entry, exists := r.ipEntries[ip]
	if !exists {
		entry = &IPEntry{
			requests: make([]time.Time, 0),
		}
		r.ipEntries[ip] = entry
	}

	return entry.allow(r.config.RequestsPerWindow, r.config.Window)
}

// GetActiveIPCount returns the number of currently tracked IPs
func (r *IPBasedRateLimiter) GetActiveIPCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.ipEntries)
}

// allow checks if a request should be allowed for this IP entry
func (e *IPEntry) allow(limit int, window time.Duration) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-window)

	// Remove expired requests
	validRequests := make([]time.Time, 0, len(e.requests))
	for _, reqTime := range e.requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}
	e.requests = validRequests

	// Check if limit exceeded
	if len(e.requests) >= limit {
		return false
	}

	// Add current request
	e.requests = append(e.requests, now)

	return true
}

// cleanup removes expired IP entries
func (r *IPBasedRateLimiter) cleanup() {
	ticker := time.NewTicker(r.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.mu.Lock()
			now := time.Now()
			cutoff := now.Add(-r.config.Window)

			for ip, entry := range r.ipEntries {
				entry.mu.Lock()
				hasValidRequests := false
				for _, reqTime := range entry.requests {
					if reqTime.After(cutoff) {
						hasValidRequests = true

						break
					}
				}
				entry.mu.Unlock()

				if !hasValidRequests {
					delete(r.ipEntries, ip)
				}
			}
			r.mu.Unlock()

		case <-r.stopCleanup:
			return
		}
	}
}

// UserBasedConfig holds user-based rate limiter configuration
type UserBasedConfig struct {
	RequestsPerWindow int
	Window            time.Duration
	MaxUsers          int
	UserTiers         map[string]int
}

// UserBasedRateLimiter implements user-based rate limiting
type UserBasedRateLimiter struct {
	config      UserBasedConfig
	userEntries map[string]*IPEntry
	mu          sync.RWMutex
}

// UserBasedOption configures user-based rate limiter
type UserBasedOption func(*UserBasedConfig)

// WithMaxUsers sets the maximum number of tracked users
func WithMaxUsers(maxUsers int) UserBasedOption {
	return func(c *UserBasedConfig) {
		c.MaxUsers = maxUsers
	}
}

// WithUserTiers sets tier-based limits
func WithUserTiers(tiers map[string]int) UserBasedOption {
	return func(c *UserBasedConfig) {
		c.UserTiers = tiers
	}
}

// NewUserBasedRateLimiter creates a new user-based rate limiter
func NewUserBasedRateLimiter(requestsPerWindow int, window time.Duration, opts ...UserBasedOption) *UserBasedRateLimiter {
	config := UserBasedConfig{
		RequestsPerWindow: requestsPerWindow,
		Window:            window,
		MaxUsers:          10000,
		UserTiers:         make(map[string]int),
	}

	for _, opt := range opts {
		opt(&config)
	}

	return &UserBasedRateLimiter{
		config:      config,
		userEntries: make(map[string]*IPEntry),
	}
}

// GetConfig returns the rate limiter configuration
func (r *UserBasedRateLimiter) GetConfig() UserBasedConfig {
	return r.config
}

// Allow checks if a request should be allowed for the given key
func (r *UserBasedRateLimiter) Allow(key string) bool {
	return r.AllowUser(key, "")
}

// AllowUser checks if a request should be allowed for the given user
func (r *UserBasedRateLimiter) AllowUser(userID, tier string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, exists := r.userEntries[userID]
	if !exists {
		entry = &IPEntry{
			requests: make([]time.Time, 0),
		}
		r.userEntries[userID] = entry
	}

	// Determine limit based on tier
	limit := r.config.RequestsPerWindow
	if tierLimit, ok := r.config.UserTiers[tier]; ok {
		limit = tierLimit
	}

	return entry.allow(limit, r.config.Window)
}

// SlidingWindowRateLimiter implements sliding window rate limiting
type SlidingWindowRateLimiter struct {
	limit   int
	window  time.Duration
	entries map[string]*IPEntry
	mu      sync.RWMutex
}

// NewSlidingWindowRateLimiter creates a new sliding window rate limiter
func NewSlidingWindowRateLimiter(limit int, window time.Duration) *SlidingWindowRateLimiter {
	return &SlidingWindowRateLimiter{
		limit:   limit,
		window:  window,
		entries: make(map[string]*IPEntry),
	}
}

// Allow checks if a request should be allowed
func (r *SlidingWindowRateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, exists := r.entries[key]
	if !exists {
		entry = &IPEntry{
			requests: make([]time.Time, 0),
		}
		r.entries[key] = entry
	}

	return entry.allow(r.limit, r.window)
}

// Middleware provides rate limiting middleware
type Middleware struct {
	limiter        RateLimiter
	userExtractor  func(interface{}) string
	metrics        *Metrics
	metricsEnabled bool
}

// Option configures rate limit middleware
type Option func(*Middleware)

// WithUserExtractor sets a custom user extractor function
func WithUserExtractor(extractor func(interface{}) string) Option {
	return func(m *Middleware) {
		m.userExtractor = extractor
	}
}

// WithMetricsEnabled enables metrics collection
func WithMetricsEnabled(enabled bool) Option {
	return func(m *Middleware) {
		m.metricsEnabled = enabled
		if enabled {
			m.metrics = &Metrics{}
		}
	}
}

// Metrics holds rate limiting metrics
type Metrics struct {
	TotalRequests   int64
	AllowedRequests int64
	DeniedRequests  int64
	mu              sync.RWMutex
}

// NewRateLimitMiddleware creates a new rate limit middleware
func NewRateLimitMiddleware(limiter RateLimiter, opts ...Option) *Middleware {
	middleware := &Middleware{
		limiter: limiter,
	}

	for _, opt := range opts {
		opt(middleware)
	}

	return middleware
}

// HTTPMiddleware returns HTTP middleware function
func (m *Middleware) HTTPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract IP from request
			ip := r.RemoteAddr
			if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
				ip = strings.Split(forwardedFor, ",")[0]
			}
			if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
				ip = realIP
			}

			// Check rate limit
			allowed := m.limiter.Allow(ip)

			// Update metrics
			if m.metricsEnabled {
				m.metrics.mu.Lock()
				m.metrics.TotalRequests++
				if allowed {
					m.metrics.AllowedRequests++
				} else {
					m.metrics.DeniedRequests++
				}
				m.metrics.mu.Unlock()
			}

			if !allowed {
				m.writeRateLimitError(w, r)
				return
			}

			// Add rate limit headers
			m.addRateLimitHeaders(w, ip)

			next.ServeHTTP(w, r)
		})
	}
}

// Before implements TypedPreMiddleware interface
func (m *Middleware) Before(ctx context.Context, req interface{}) (context.Context, error) {
	var key string

	// Try to extract user ID first
	if m.userExtractor != nil {
		key = m.userExtractor(req)
	}

	// Fallback to IP address
	if key == "" {
		if httpReq, ok := ctx.Value("http_request").(*http.Request); ok {
			key = httpReq.RemoteAddr
		}
	}

	if key == "" {
		return ctx, errors.New("unable to extract rate limit key")
	}

	// Check rate limit
	allowed := m.limiter.Allow(key)
	if !allowed {
		return ctx, ErrRateLimitExceeded
	}

	return ctx, nil
}

// GetMetrics returns current metrics
func (m *Middleware) GetMetrics() *Metrics {
	if !m.metricsEnabled {
		return nil
	}

	m.metrics.mu.RLock()
	defer m.metrics.mu.RUnlock()

	return &Metrics{
		TotalRequests:   m.metrics.TotalRequests,
		AllowedRequests: m.metrics.AllowedRequests,
		DeniedRequests:  m.metrics.DeniedRequests,
	}
}

// writeRateLimitError writes a rate limit exceeded error response
func (m *Middleware) writeRateLimitError(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": "rate limit exceeded",
	})
}

// addRateLimitHeaders adds rate limit headers to response
func (m *Middleware) addRateLimitHeaders(w http.ResponseWriter, _ string) {
	// Default headers - in a real implementation, these would be calculated based on the limiter type.
	w.Header().Set("X-RateLimit-Limit", "100")
	w.Header().Set("X-RateLimit-Remaining", "99")
	w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10))
}

// Helper function for Go < 1.21.
func minInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}
