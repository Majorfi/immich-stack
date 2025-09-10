package utils

import (
	"regexp"
	"testing"
)

func TestLRUCache(t *testing.T) {
	// Create a small cache for testing
	cache := NewLRUCache(3)

	// Test basic get/put operations
	t.Run("Basic operations", func(t *testing.T) {
		// Cache miss
		if _, ok := cache.Get("test1"); ok {
			t.Error("Expected cache miss")
		}

		// Store regex
		regex1 := regexp.MustCompile("test1")
		cache.Put("test1", regex1)

		// Cache hit
		if result, ok := cache.Get("test1"); !ok || result != regex1 {
			t.Error("Expected cache hit")
		}
	})

	// Test LRU eviction
	t.Run("LRU eviction", func(t *testing.T) {
		cache := NewLRUCache(2) // Small cache

		// Fill cache to capacity
		regex1 := regexp.MustCompile("pattern1")
		regex2 := regexp.MustCompile("pattern2")
		cache.Put("pattern1", regex1)
		cache.Put("pattern2", regex2)

		// Verify both are cached
		if _, ok := cache.Get("pattern1"); !ok {
			t.Error("Expected pattern1 to be cached")
		}
		if _, ok := cache.Get("pattern2"); !ok {
			t.Error("Expected pattern2 to be cached")
		}

		// Add third item, should evict least recently used
		regex3 := regexp.MustCompile("pattern3")
		cache.Put("pattern3", regex3)

		// pattern1 should be evicted (least recently used)
		if _, ok := cache.Get("pattern1"); ok {
			t.Error("Expected pattern1 to be evicted")
		}

		// pattern2 and pattern3 should still be cached
		if _, ok := cache.Get("pattern2"); !ok {
			t.Error("Expected pattern2 to still be cached")
		}
		if _, ok := cache.Get("pattern3"); !ok {
			t.Error("Expected pattern3 to be cached")
		}
	})

	// Test access ordering
	t.Run("Access updates LRU order", func(t *testing.T) {
		cache := NewLRUCache(2)

		// Add two items
		regex1 := regexp.MustCompile("old")
		regex2 := regexp.MustCompile("new")
		cache.Put("old", regex1)
		cache.Put("new", regex2)

		// Access the old item to make it recently used
		cache.Get("old")

		// Add third item, should evict "new" (now least recently used)
		regex3 := regexp.MustCompile("latest")
		cache.Put("latest", regex3)

		// "old" should still be cached (was accessed recently)
		if _, ok := cache.Get("old"); !ok {
			t.Error("Expected 'old' to still be cached after recent access")
		}

		// "new" should be evicted
		if _, ok := cache.Get("new"); ok {
			t.Error("Expected 'new' to be evicted")
		}

		// "latest" should be cached
		if _, ok := cache.Get("latest"); !ok {
			t.Error("Expected 'latest' to be cached")
		}
	})

	// Test updating existing entries
	t.Run("Update existing entry", func(t *testing.T) {
		cache := NewLRUCache(2)

		// Add initial regex
		regex1 := regexp.MustCompile("pattern")
		cache.Put("key", regex1)

		// Update with new regex
		regex2 := regexp.MustCompile("updated_pattern")
		cache.Put("key", regex2)

		// Should get updated regex
		if result, ok := cache.Get("key"); !ok || result != regex2 {
			t.Error("Expected updated regex")
		}
	})
}

func TestCompile(t *testing.T) {
	// Test the public API works with LRU cache
	pattern := "test_pattern_\\d+"

	// First call compiles and caches
	regex1, err := RegexCompile(pattern)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Second call should return cached version
	regex2, err := RegexCompile(pattern)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should be the same regex object (cached)
	if regex1 != regex2 {
		t.Error("Expected cached regex to be returned")
	}

	// Test invalid regex returns error
	if _, err = RegexCompile("[invalid"); err == nil {
		t.Error("Expected error for invalid regex")
	}
}
