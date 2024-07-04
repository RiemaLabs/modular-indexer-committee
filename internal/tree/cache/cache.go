package cache

import (
	"container/list"

	verkle "github.com/ethereum/go-verkle"
)

type LRUCache struct {
	capacity int
	cache    map[[verkle.KeySize]byte]*list.Element
	queue    *list.List
}

type entry struct {
	value []byte
}

func NewLRUCache(capacity int) *LRUCache {
	return &LRUCache{
		capacity: capacity,
		cache:    make(map[[verkle.KeySize]byte]*list.Element),
		queue:    list.New(),
	}
}

func transformKeys(key []byte) [verkle.KeySize]byte {
	var transformedKey [verkle.KeySize]byte
	copy(transformedKey[:], key)
	return transformedKey
}

func (c *LRUCache) Insert(value []byte) (evictedValue []byte, evicted bool) {
	key := transformKeys(value)
	if elem, ok := c.cache[key]; ok {
		c.queue.MoveToFront(elem)
		return nil, false
	}

	newEntry := &entry{value}
	elem := c.queue.PushFront(newEntry)
	c.cache[key] = elem

	if c.queue.Len() > c.capacity {
		lastElem := c.queue.Back()
		if lastElem != nil {
			lastEntry := lastElem.Value.(*entry)
			c.queue.Remove(lastElem)
			delete(c.cache, transformKeys(lastEntry.value))
			return lastEntry.value, true
		}
	}

	return nil, false
}
