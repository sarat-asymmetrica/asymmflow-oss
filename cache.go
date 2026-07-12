package main

import (
	"sync"
	"time"
)

// CacheEntry represents a cached value with expiration
type CacheEntry struct {
	Value      any
	Expiration time.Time
}

// Cache provides in-memory caching with TTL
type Cache struct {
	mu       sync.RWMutex
	entries  map[string]*CacheEntry
	stop     chan struct{}
	stopOnce sync.Once
}

// NewCache creates a new cache instance
func NewCache() *Cache {
	c := &Cache{
		entries: make(map[string]*CacheEntry),
		stop:    make(chan struct{}),
	}

	// Start background cleanup goroutine
	go c.cleanup()

	return c
}

// Stop halts the background cleanup goroutine. Safe to call multiple times.
// The long-lived app never needs this (the cache lives for the process), but
// short-lived owners — every test's setupTestApp — must stop it or the cleanup
// goroutine (and, transitively, anything it keeps referenced) leaks for the
// rest of the run. Hundreds of these leaking across a full test suite were the
// real cross-test resource contamination behind the "Butler flake" (Wave 9.5 C4).
func (c *Cache) Stop() {
	c.stopOnce.Do(func() {
		close(c.stop)
	})
}

// Get retrieves a value from cache
func (c *Cache) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.Expiration) {
		return nil, false
	}

	return entry.Value, true
}

// Set stores a value in cache with TTL
func (c *Cache) Set(key string, value any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = &CacheEntry{
		Value:      value,
		Expiration: time.Now().Add(ttl),
	}
}

// Delete removes a value from cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, key)
}

// InvalidatePattern deletes all keys matching a pattern (simple prefix match)
func (c *Cache) InvalidatePattern(pattern string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key := range c.entries {
		if len(key) >= len(pattern) && key[:len(pattern)] == pattern {
			delete(c.entries, key)
		}
	}
}

// Clear removes all entries from cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*CacheEntry)
}

// Stats returns cache statistics
func (c *Cache) Stats() map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := len(c.entries)
	expired := 0
	now := time.Now()

	for _, entry := range c.entries {
		if now.After(entry.Expiration) {
			expired++
		}
	}

	return map[string]any{
		"total_entries":   total,
		"active_entries":  total - expired,
		"expired_entries": expired,
	}
}

// cleanup removes expired entries every minute until Stop() is called.
func (c *Cache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.removeExpired()
		case <-c.stop:
			return
		}
	}
}

// removeExpired deletes all expired entries
func (c *Cache) removeExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.Expiration) {
			delete(c.entries, key)
		}
	}
}

// --- Cache Key Constants ---

const (
	CacheKeyCustomerList   = "customers:list"
	CacheKeySupplierList   = "suppliers:list"
	CacheKeyProductCatalog = "products:catalog"
	CacheKeyRoleList       = "roles:list"
	CacheKeyDashboardStats = "dashboard:stats"
	CacheKeyFinancialDash  = "financial:dashboard"
	CacheKeyCustomerPrefix = "customer:"
	CacheKeySupplierPrefix = "supplier:"
	CacheKeyOrderPrefix    = "order:"
	CacheKeyInvoicePrefix  = "invoice:"
)

// --- Cache TTL Constants ---

const (
	CacheTTLShort  = 1 * time.Minute  // For rapidly changing data (dashboard stats)
	CacheTTLMedium = 5 * time.Minute  // For master data (customers, suppliers)
	CacheTTLLong   = 10 * time.Minute // For static data (products, roles)
)
