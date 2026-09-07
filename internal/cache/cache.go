package cache

import (
	"hash/fnv"
	"sync"
	"time"

	"gh-pin-repo-go/internal/model"
)

const shardCount = 32

type cacheItem struct {
	data      []model.PinnedRepo
	expiresAt time.Time
}

type cacheShard struct {
	sync.RWMutex
	items map[string]cacheItem
}

// ShardedCache is a high-performance, thread-safe, sharded in-memory cache
type ShardedCache struct {
	shards []*cacheShard
	ttl    time.Duration
	stopCh chan struct{}
}

// New creates a new ShardedCache with the given default TTL
func New(defaultTTL time.Duration) *ShardedCache {
	c := &ShardedCache{
		shards: make([]*cacheShard, shardCount),
		ttl:    defaultTTL,
		stopCh: make(chan struct{}),
	}
	for i := 0; i < shardCount; i++ {
		c.shards[i] = &cacheShard{
			items: make(map[string]cacheItem),
		}
	}

	// Start background cleanup ticker
	go c.janitor(5 * time.Minute)

	return c
}

func (c *ShardedCache) getShard(key string) *cacheShard {
	h := fnv.New32a()
	h.Write([]byte(key))
	return c.shards[h.Sum32()%shardCount]
}

// Get retrieves an item from the cache
func (c *ShardedCache) Get(key string) ([]model.PinnedRepo, bool) {
	shard := c.getShard(key)
	shard.RLock()
	item, found := shard.items[key]
	shard.RUnlock()

	if !found {
		return nil, false
	}

	if time.Now().After(item.expiresAt) {
		// Clean up lazily
		shard.Lock()
		delete(shard.items, key)
		shard.Unlock()
		return nil, false
	}

	return item.data, true
}

// Set stores an item in the cache
func (c *ShardedCache) Set(key string, data []model.PinnedRepo, ttl time.Duration) {
	if ttl <= 0 {
		ttl = c.ttl
	}

	shard := c.getShard(key)
	shard.Lock()
	shard.items[key] = cacheItem{
		data:      data,
		expiresAt: time.Now().Add(ttl),
	}
	shard.Unlock()
}

// Delete removes an item from the cache
func (c *ShardedCache) Delete(key string) {
	shard := c.getShard(key)
	shard.Lock()
	delete(shard.items, key)
	shard.Unlock()
}

// Close terminates the janitor routine
func (c *ShardedCache) Close() {
	select {
	case <-c.stopCh:
	default:
		close(c.stopCh)
	}
}

func (c *ShardedCache) janitor(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			now := time.Now()
			for _, shard := range c.shards {
				shard.Lock()
				for k, v := range shard.items {
					if now.After(v.expiresAt) {
						delete(shard.items, k)
					}
				}
				shard.Unlock()
			}
		}
	}
}
