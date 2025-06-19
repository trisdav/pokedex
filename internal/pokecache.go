package internal
import (
	"time"
	"sync"
)

type cacheEntry struct {
	createdAt time.Time
	val []byte
}

type Cache struct {
	Map map[string]cacheEntry
	Mu sync.Mutex
}

func NewCache(interval time.Duration) Cache {
	cache := new(Cache)
	cache.Map = make(map[string]cacheEntry)
	
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			cache.reapLoop(interval)
		}
	}()
	
	return *cache
}

func (cache Cache) Add(key string, val []byte) {
	cache.Mu.Lock()
	newEntry := cacheEntry{createdAt:time.Now(),val:val}
	cache.Map[key] = newEntry
	cache.Mu.Unlock()
}

func (cache Cache) Get(key string) ([]byte, bool) {
	cache.Mu.Lock() // This seems safe? I don't think I need this.
	value, exists := cache.Map[key]
	cache.Mu.Unlock()
	return value.val, exists
}

func (cache Cache) reapLoop(interval time.Duration) {
	cache.Mu.Lock()
	for key,_ := range cache.Map {
		if cache.Map[key].createdAt.Before((time.Now().Add(-interval))) {
			delete(cache.Map,key)
		}
	}
	cache.Mu.Unlock()
}