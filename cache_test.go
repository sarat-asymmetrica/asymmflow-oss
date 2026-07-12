package main

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// =============================================================================
// PRODUCTION CACHE SERVICE TESTS
// =============================================================================
// Complete test suite for in-memory cache implementation.
// Tests verify TTL behavior, concurrency safety, and pattern matching.
// Run with: go test -v -race
// =============================================================================

// TestCacheSetGet verifies basic cache set/get operations
func TestCacheSetGet(t *testing.T) {
	t.Run("should store and retrieve value", func(t *testing.T) {
		cache := NewCache()
		t.Cleanup(cache.Stop)
		cache.Set("key1", "value1", 5*time.Minute)

		val, found := cache.Get("key1")
		if !found {
			t.Error("Expected to find key1")
		}
		if val != "value1" {
			t.Errorf("Expected value1, got %v", val)
		}
	})

	t.Run("should return false for missing key", func(t *testing.T) {
		cache := NewCache()
		t.Cleanup(cache.Stop)
		_, found := cache.Get("nonexistent")
		if found {
			t.Error("Expected not to find nonexistent key")
		}
	})

	t.Run("should handle different value types", func(t *testing.T) {
		cache := NewCache()
		t.Cleanup(cache.Stop)

		// String
		cache.Set("string_key", "test_string", CacheTTLShort)
		val, found := cache.Get("string_key")
		if !found || val != "test_string" {
			t.Errorf("String test failed: found=%v, val=%v", found, val)
		}

		// Int
		cache.Set("int_key", 42, CacheTTLShort)
		val, found = cache.Get("int_key")
		if !found || val != 42 {
			t.Errorf("Int test failed: found=%v, val=%v", found, val)
		}

		// Struct
		type TestStruct struct {
			Name string
			Age  int
		}
		testStruct := TestStruct{Name: "Alice", Age: 30}
		cache.Set("struct_key", testStruct, CacheTTLShort)
		val, found = cache.Get("struct_key")
		if !found {
			t.Error("Struct not found")
		}
		retrieved, ok := val.(TestStruct)
		if !ok || retrieved.Name != "Alice" || retrieved.Age != 30 {
			t.Errorf("Struct test failed: %+v", val)
		}

		// Map
		testMap := map[string]int{"a": 1, "b": 2}
		cache.Set("map_key", testMap, CacheTTLShort)
		val, found = cache.Get("map_key")
		if !found {
			t.Error("Map not found")
		}

		// Slice
		testSlice := []string{"x", "y", "z"}
		cache.Set("slice_key", testSlice, CacheTTLShort)
		val, found = cache.Get("slice_key")
		if !found {
			t.Error("Slice not found")
		}
	})

	t.Run("should overwrite existing key", func(t *testing.T) {
		cache := NewCache()
		t.Cleanup(cache.Stop)
		cache.Set("key", "value1", CacheTTLShort)
		cache.Set("key", "value2", CacheTTLShort)

		val, found := cache.Get("key")
		if !found || val != "value2" {
			t.Errorf("Expected value2 after overwrite, got %v", val)
		}
	})
}

// TestCacheTTLExpiry verifies TTL-based expiration
func TestCacheTTLExpiry(t *testing.T) {
	t.Run("should expire after TTL", func(t *testing.T) {
		cache := NewCache()
		t.Cleanup(cache.Stop)
		cache.Set("key", "value", 10*time.Millisecond)

		// Verify exists immediately
		_, found := cache.Get("key")
		if !found {
			t.Error("Expected key to exist immediately after Set")
		}

		// Wait for expiry
		time.Sleep(20 * time.Millisecond)

		// Verify expired
		_, found = cache.Get("key")
		if found {
			t.Error("Expected key to be expired")
		}
	})

	t.Run("should not expire before TTL", func(t *testing.T) {
		cache := NewCache()
		t.Cleanup(cache.Stop)
		cache.Set("key", "value", 1*time.Second)

		// Wait half the TTL
		time.Sleep(500 * time.Millisecond)

		_, found := cache.Get("key")
		if !found {
			t.Error("Expected key to still exist before TTL")
		}
	})

	t.Run("should cleanup expired entries in background", func(t *testing.T) {
		// Note: Cleanup runs every 1 minute, so we manually trigger removeExpired
		cache := NewCache()
		t.Cleanup(cache.Stop)

		// Set multiple entries with very short TTL
		for i := 0; i < 10; i++ {
			cache.Set(fmt.Sprintf("key_%d", i), i, 10*time.Millisecond)
		}

		// Verify all exist
		stats := cache.Stats()
		if stats["total_entries"].(int) != 10 {
			t.Errorf("Expected 10 entries, got %v", stats["total_entries"])
		}

		// Wait for expiry
		time.Sleep(20 * time.Millisecond)

		// Manually trigger cleanup (since background cleanup is 1 minute interval)
		cache.removeExpired()

		// Verify all cleaned up
		stats = cache.Stats()
		if stats["total_entries"].(int) != 0 {
			t.Errorf("Expected 0 entries after cleanup, got %v", stats["total_entries"])
		}
	})

	t.Run("should handle long TTL values", func(t *testing.T) {
		cache := NewCache()
		t.Cleanup(cache.Stop)
		cache.Set("key", "value", 24*time.Hour)

		// Verify exists now
		val, found := cache.Get("key")
		if !found || val != "value" {
			t.Error("Expected key with long TTL to exist")
		}

		// Verify stats show as active (not expired)
		stats := cache.Stats()
		if stats["active_entries"].(int) != 1 {
			t.Errorf("Expected 1 active entry, got %v", stats["active_entries"])
		}
		if stats["expired_entries"].(int) != 0 {
			t.Errorf("Expected 0 expired entries, got %v", stats["expired_entries"])
		}
	})
}

// TestCacheInvalidatePattern verifies pattern-based invalidation
func TestCacheInvalidatePattern(t *testing.T) {
	t.Run("should delete all keys matching prefix", func(t *testing.T) {
		cache := NewCache()
		t.Cleanup(cache.Stop)
		cache.Set("customer:1", "data1", CacheTTLMedium)
		cache.Set("customer:2", "data2", CacheTTLMedium)
		cache.Set("supplier:1", "data3", CacheTTLMedium)
		cache.Set("order:1", "data4", CacheTTLMedium)

		// Invalidate customer pattern
		cache.InvalidatePattern("customer:")

		// Verify customer keys are gone
		_, found := cache.Get("customer:1")
		if found {
			t.Error("Expected customer:1 to be deleted")
		}
		_, found = cache.Get("customer:2")
		if found {
			t.Error("Expected customer:2 to be deleted")
		}

		// Verify non-matching keys still exist
		_, found = cache.Get("supplier:1")
		if !found {
			t.Error("Expected supplier:1 to still exist")
		}
		_, found = cache.Get("order:1")
		if !found {
			t.Error("Expected order:1 to still exist")
		}
	})

	t.Run("should not delete non-matching keys", func(t *testing.T) {
		cache := NewCache()
		t.Cleanup(cache.Stop)
		cache.Set("customer:123", "val1", CacheTTLMedium)
		cache.Set("supplier:456", "val2", CacheTTLMedium)
		cache.Set("order:789", "val3", CacheTTLMedium)

		cache.InvalidatePattern("supplier:")

		// Verify only supplier: keys deleted
		_, found := cache.Get("customer:123")
		if !found {
			t.Error("Expected customer:123 to remain")
		}
		_, found = cache.Get("supplier:456")
		if found {
			t.Error("Expected supplier:456 to be deleted")
		}
		_, found = cache.Get("order:789")
		if !found {
			t.Error("Expected order:789 to remain")
		}
	})

	t.Run("should handle empty pattern", func(t *testing.T) {
		cache := NewCache()
		t.Cleanup(cache.Stop)
		cache.Set("key1", "val1", CacheTTLMedium)
		cache.Set("key2", "val2", CacheTTLMedium)
		cache.Set("key3", "val3", CacheTTLMedium)

		// Empty pattern matches all keys
		cache.InvalidatePattern("")

		stats := cache.Stats()
		if stats["total_entries"].(int) != 0 {
			t.Errorf("Expected all entries deleted with empty pattern, got %v entries", stats["total_entries"])
		}
	})

	t.Run("should be case sensitive", func(t *testing.T) {
		cache := NewCache()
		t.Cleanup(cache.Stop)
		cache.Set("Customer:1", "data", CacheTTLMedium)
		cache.Set("customer:2", "data", CacheTTLMedium)

		// Lowercase pattern should not match uppercase key
		cache.InvalidatePattern("customer:")

		// Verify uppercase key still exists (case mismatch)
		_, found := cache.Get("Customer:1")
		if !found {
			t.Error("Expected Customer:1 to still exist (case sensitive)")
		}

		// Verify lowercase key was deleted
		_, found = cache.Get("customer:2")
		if found {
			t.Error("Expected customer:2 to be deleted")
		}
	})
}

// TestCacheConcurrency verifies thread-safety
func TestCacheConcurrency(t *testing.T) {
	t.Run("should handle concurrent Set operations", func(t *testing.T) {
		cache := NewCache()
		t.Cleanup(cache.Stop)
		var wg sync.WaitGroup

		// 100 goroutines setting different keys
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				cache.Set(fmt.Sprintf("key_%d", idx), idx, CacheTTLShort)
			}(i)
		}
		wg.Wait()

		// Verify all 100 keys exist
		stats := cache.Stats()
		if stats["total_entries"].(int) != 100 {
			t.Errorf("Expected 100 entries after concurrent Set, got %v", stats["total_entries"])
		}
	})

	t.Run("should handle concurrent Get operations", func(t *testing.T) {
		cache := NewCache()
		t.Cleanup(cache.Stop)
		cache.Set("shared_key", "value", CacheTTLMedium)

		var wg sync.WaitGroup
		errors := make(chan error, 100)

		// 100 goroutines reading same key
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				val, found := cache.Get("shared_key")
				if !found {
					errors <- fmt.Errorf("key not found")
					return
				}
				if val != "value" {
					errors <- fmt.Errorf("wrong value: %v", val)
					return
				}
			}()
		}
		wg.Wait()
		close(errors)

		// Check for any errors
		for err := range errors {
			t.Error(err)
		}
	})

	t.Run("should handle concurrent mixed operations", func(t *testing.T) {
		cache := NewCache()
		t.Cleanup(cache.Stop)
		var wg sync.WaitGroup

		// Mix of operations
		for i := 0; i < 50; i++ {
			// Set
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				cache.Set(fmt.Sprintf("key_%d", idx), idx, CacheTTLMedium)
			}(i)

			// Get
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				cache.Get(fmt.Sprintf("key_%d", idx))
			}(i)

			// Delete
			if i%10 == 0 {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()
					cache.Delete(fmt.Sprintf("key_%d", idx))
				}(i)
			}

			// InvalidatePattern
			if i%20 == 0 {
				wg.Add(1)
				go func() {
					defer wg.Done()
					cache.InvalidatePattern("key_1")
				}()
			}
		}

		wg.Wait()

		// Should not panic - exact count doesn't matter due to deletes
		stats := cache.Stats()
		t.Logf("Final entries after mixed operations: %v", stats["total_entries"])
	})

	t.Run("should handle concurrent cleanup and access", func(t *testing.T) {
		cache := NewCache()
		t.Cleanup(cache.Stop)
		var wg sync.WaitGroup

		// Set entries with short TTL
		for i := 0; i < 50; i++ {
			cache.Set(fmt.Sprintf("key_%d", i), i, 20*time.Millisecond)
		}

		// Start goroutines reading while entries expire
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					cache.Get(fmt.Sprintf("key_%d", idx))
					time.Sleep(2 * time.Millisecond)
				}
			}(i)
		}

		// Manually trigger cleanup during reads
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(25 * time.Millisecond)
			cache.removeExpired()
		}()

		wg.Wait()

		// Should complete without panics or race conditions
		stats := cache.Stats()
		t.Logf("Final stats after concurrent cleanup: %+v", stats)
	})

	t.Run("should handle concurrent overwrites of same key", func(t *testing.T) {
		cache := NewCache()
		t.Cleanup(cache.Stop)
		var wg sync.WaitGroup

		// 100 goroutines overwriting same key
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				cache.Set("contested_key", idx, CacheTTLMedium)
			}(i)
		}
		wg.Wait()

		// Should have exactly 1 entry (last write wins)
		stats := cache.Stats()
		if stats["total_entries"].(int) != 1 {
			t.Errorf("Expected 1 entry after concurrent overwrites, got %v", stats["total_entries"])
		}

		// Value should be some integer 0-99
		val, found := cache.Get("contested_key")
		if !found {
			t.Error("Expected contested_key to exist")
		}
		if _, ok := val.(int); !ok {
			t.Errorf("Expected int value, got %T", val)
		}
	})
}

// TestCacheStats verifies statistics reporting
func TestCacheStats(t *testing.T) {
	t.Run("should return zero stats for empty cache", func(t *testing.T) {
		cache := NewCache()
		t.Cleanup(cache.Stop)
		stats := cache.Stats()

		if stats["total_entries"].(int) != 0 {
			t.Errorf("Expected 0 total_entries, got %v", stats["total_entries"])
		}
		if stats["active_entries"].(int) != 0 {
			t.Errorf("Expected 0 active_entries, got %v", stats["active_entries"])
		}
		if stats["expired_entries"].(int) != 0 {
			t.Errorf("Expected 0 expired_entries, got %v", stats["expired_entries"])
		}
	})

	t.Run("should count active entries correctly", func(t *testing.T) {
		cache := NewCache()
		t.Cleanup(cache.Stop)
		cache.Set("key1", "val1", CacheTTLLong)
		cache.Set("key2", "val2", CacheTTLLong)
		cache.Set("key3", "val3", CacheTTLLong)

		stats := cache.Stats()

		if stats["total_entries"].(int) != 3 {
			t.Errorf("Expected 3 total_entries, got %v", stats["total_entries"])
		}
		if stats["active_entries"].(int) != 3 {
			t.Errorf("Expected 3 active_entries, got %v", stats["active_entries"])
		}
		if stats["expired_entries"].(int) != 0 {
			t.Errorf("Expected 0 expired_entries, got %v", stats["expired_entries"])
		}
	})

	t.Run("should count expired entries correctly", func(t *testing.T) {
		cache := NewCache()
		t.Cleanup(cache.Stop)
		cache.Set("key1", "val1", 5*time.Millisecond)
		cache.Set("key2", "val2", 5*time.Millisecond)
		cache.Set("key3", "val3", CacheTTLLong) // This one won't expire

		// Wait for expiry
		time.Sleep(10 * time.Millisecond)

		stats := cache.Stats()

		if stats["total_entries"].(int) != 3 {
			t.Errorf("Expected 3 total_entries (before cleanup), got %v", stats["total_entries"])
		}
		if stats["expired_entries"].(int) != 2 {
			t.Errorf("Expected 2 expired_entries, got %v", stats["expired_entries"])
		}
		if stats["active_entries"].(int) != 1 {
			t.Errorf("Expected 1 active_entry, got %v", stats["active_entries"])
		}
	})

	t.Run("should update stats after cleanup", func(t *testing.T) {
		cache := NewCache()
		t.Cleanup(cache.Stop)
		cache.Set("key1", "val1", 5*time.Millisecond)
		cache.Set("key2", "val2", 5*time.Millisecond)

		// Wait for expiry
		time.Sleep(10 * time.Millisecond)

		// Stats before cleanup
		statsBefore := cache.Stats()
		if statsBefore["expired_entries"].(int) != 2 {
			t.Errorf("Expected 2 expired before cleanup, got %v", statsBefore["expired_entries"])
		}

		// Cleanup
		cache.removeExpired()

		// Stats after cleanup
		statsAfter := cache.Stats()
		if statsAfter["total_entries"].(int) != 0 {
			t.Errorf("Expected 0 total_entries after cleanup, got %v", statsAfter["total_entries"])
		}
		if statsAfter["expired_entries"].(int) != 0 {
			t.Errorf("Expected 0 expired_entries after cleanup, got %v", statsAfter["expired_entries"])
		}
	})
}

// TestCacheClear verifies clearing all entries
func TestCacheClear(t *testing.T) {
	t.Run("should remove all entries", func(t *testing.T) {
		cache := NewCache()
		t.Cleanup(cache.Stop)
		cache.Set("key1", "val1", CacheTTLMedium)
		cache.Set("key2", "val2", CacheTTLMedium)
		cache.Set("key3", "val3", CacheTTLMedium)

		// Verify entries exist
		stats := cache.Stats()
		if stats["total_entries"].(int) != 3 {
			t.Errorf("Expected 3 entries before clear, got %v", stats["total_entries"])
		}

		// Clear
		cache.Clear()

		// Verify all removed
		stats = cache.Stats()
		if stats["total_entries"].(int) != 0 {
			t.Errorf("Expected 0 entries after clear, got %v", stats["total_entries"])
		}

		// Verify specific keys are gone
		_, found := cache.Get("key1")
		if found {
			t.Error("Expected key1 to be cleared")
		}
	})

	t.Run("should handle clearing empty cache", func(t *testing.T) {
		cache := NewCache()
		t.Cleanup(cache.Stop)

		// Clear empty cache - should not panic
		cache.Clear()

		stats := cache.Stats()
		if stats["total_entries"].(int) != 0 {
			t.Errorf("Expected 0 entries, got %v", stats["total_entries"])
		}
	})

	t.Run("should allow setting after clear", func(t *testing.T) {
		cache := NewCache()
		t.Cleanup(cache.Stop)
		cache.Set("key1", "val1", CacheTTLMedium)
		cache.Clear()

		// Set new value after clear
		cache.Set("key2", "val2", CacheTTLMedium)

		val, found := cache.Get("key2")
		if !found || val != "val2" {
			t.Error("Expected to set and get after clear")
		}
	})
}

// TestCacheDelete verifies single key deletion
func TestCacheDelete(t *testing.T) {
	t.Run("should delete existing key", func(t *testing.T) {
		cache := NewCache()
		t.Cleanup(cache.Stop)
		cache.Set("key", "value", CacheTTLMedium)

		// Verify exists
		_, found := cache.Get("key")
		if !found {
			t.Error("Expected key to exist before delete")
		}

		// Delete
		cache.Delete("key")

		// Verify deleted
		_, found = cache.Get("key")
		if found {
			t.Error("Expected key to be deleted")
		}

		// Verify stats updated
		stats := cache.Stats()
		if stats["total_entries"].(int) != 0 {
			t.Errorf("Expected 0 entries after delete, got %v", stats["total_entries"])
		}
	})

	t.Run("should handle deleting non-existent key", func(t *testing.T) {
		cache := NewCache()
		t.Cleanup(cache.Stop)

		// Delete non-existent key - should not panic
		cache.Delete("missing_key")

		stats := cache.Stats()
		if stats["total_entries"].(int) != 0 {
			t.Errorf("Expected 0 entries, got %v", stats["total_entries"])
		}
	})

	t.Run("should delete only specified key", func(t *testing.T) {
		cache := NewCache()
		t.Cleanup(cache.Stop)
		cache.Set("key1", "val1", CacheTTLMedium)
		cache.Set("key2", "val2", CacheTTLMedium)
		cache.Set("key3", "val3", CacheTTLMedium)

		cache.Delete("key2")

		// Verify key2 deleted
		_, found := cache.Get("key2")
		if found {
			t.Error("Expected key2 to be deleted")
		}

		// Verify others remain
		_, found = cache.Get("key1")
		if !found {
			t.Error("Expected key1 to remain")
		}
		_, found = cache.Get("key3")
		if !found {
			t.Error("Expected key3 to remain")
		}
	})
}

// Helper for testing concurrent operations safely
func testConcurrentOperations(t *testing.T, operations int, operation func(int)) {
	var wg sync.WaitGroup
	wg.Add(operations)

	for i := 0; i < operations; i++ {
		go func(idx int) {
			defer wg.Done()
			operation(idx)
		}(i)
	}

	wg.Wait()
}
