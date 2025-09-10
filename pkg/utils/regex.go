package utils

import (
	"container/list"
	"regexp"
	"sync"
)

// lruEntry represents an entry in the LRU cache
type lruEntry struct {
	key   string
	regex *regexp.Regexp
	node  *list.Element
}

// LRUCache implements a thread-safe LRU cache for compiled regular expressions
type LRUCache struct {
	mu       sync.Mutex
	capacity int
	cache    map[string]*lruEntry
	lru      *list.List
}

/**************************************************************************************************
** NewLRUCache creates a new LRU cache for compiled regular expressions.
**
** @param capacity - Maximum number of cached patterns before evicting LRU
** @return *LRUCache - Initialized LRU cache instance
**************************************************************************************************/
func NewLRUCache(capacity int) *LRUCache {
	return &LRUCache{
		capacity: capacity,
		cache:    make(map[string]*lruEntry),
		lru:      list.New(),
	}
}

/**************************************************************************************************
** Get retrieves a compiled regex from the cache and marks it as most recently used.
**
** @param pattern - Regex pattern string key
** @return *regexp.Regexp - Compiled regex if present
** @return bool - True if found in cache
**************************************************************************************************/
func (c *LRUCache) Get(pattern string) (*regexp.Regexp, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, ok := c.cache[pattern]; ok {
		c.lru.MoveToFront(entry.node)
		return entry.regex, true
	}
	return nil, false
}

/**************************************************************************************************
** Put inserts or updates a compiled regex in the cache, evicting the LRU entry if at capacity.
**
** @param pattern - Regex pattern string key
** @param regex - Compiled regex to store
**************************************************************************************************/
func (c *LRUCache) Put(pattern string, regex *regexp.Regexp) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, ok := c.cache[pattern]; ok {
		entry.regex = regex
		c.lru.MoveToFront(entry.node)
		return
	}

	if len(c.cache) >= c.capacity {
		c.evictLRU()
	}

	node := c.lru.PushFront(pattern)
	entry := &lruEntry{
		key:   pattern,
		regex: regex,
		node:  node,
	}
	c.cache[pattern] = entry
}

/**************************************************************************************************
** evictLRU removes the least recently used cache entry if one exists.
**************************************************************************************************/
func (c *LRUCache) evictLRU() {
	if c.lru.Len() == 0 {
		return
	}

	node := c.lru.Back()
	if node != nil {
		pattern := node.Value.(string)
		delete(c.cache, pattern)
		c.lru.Remove(node)
	}
}

// Default cache instance with LRU eviction to prevent memory leaks
// in long-running applications with dynamic criteria. Defaults to 1000 entries.
var defaultCache = NewLRUCache(1000)

// NumericSuffixPattern matches numeric suffixes in filenames (e.g., "001", "123").
// Compiled once at package initialization for performance.
var NumericSuffixPattern = regexp.MustCompile(`^(\d+)$`)

/**************************************************************************************************
** RegexCompile compiles a regular expression pattern and caches the result using the default LRU cache.
** This avoids repeated compilation and helps prevent memory growth in long-running applications.
**
** @param pattern - The regex pattern to compile
** @return *regexp.Regexp - Compiled regex
** @return error - Compilation error, if any
**************************************************************************************************/
func RegexCompile(pattern string) (*regexp.Regexp, error) {
	if re, ok := defaultCache.Get(pattern); ok {
		return re, nil
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	defaultCache.Put(pattern, re)
	return re, nil
}
