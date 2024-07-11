package cache

import (
	"fmt"
	"testing"
)

func TestCacheInsert(t *testing.T) {
	lru := NewLRUCache(3)
	_, _ = lru.Insert([]byte("value1"))
	_, _ = lru.Insert([]byte("value2"))
	_, evicted := lru.Insert([]byte("value3"))
	if evicted {
		t.Fatalf("Expected no eviction")
	}
	fmt.Printf("Inserted: %v, Evicted: %t\n", "value3", evicted)
	evictedValue, evicted := lru.Insert([]byte("value4"))
	if !evicted {
		t.Fatalf("Expected eviction")
	}
	fmt.Printf("Evicted: %s\n", evictedValue)
}
