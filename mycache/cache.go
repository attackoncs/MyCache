// 并发控制，通过互斥索实现并发缓存，并封装lru的相关函数

package mycache

import (
	"mycache/lru"
	"sync"
)

// 实例化cache,确保是可并发读写的lru
type cache struct {
	mu         sync.Mutex //并发时的互斥锁，这里还可优化成用channel提高写性能，读时在优化
	lru        *lru.Cache //lru缓存
	cacheBytes int64      //最大内存
}

// 封装lru的Add,确保并发安全
func (c *cache) add(key string, value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	//延迟初始化，创建将延迟到第一次使用时，提高性能，减少内存要求。
	if c.lru == nil {
		c.lru = lru.New(c.cacheBytes, nil)
	}
	c.lru.Add(key, value)
}

// 封装lru的Get,确保并发安全
func (c *cache) get(key string) (value ByteView, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		return
	}

	if v, ok := c.lru.Get(key); ok {
		return v.(ByteView), ok
	}

	return
}
