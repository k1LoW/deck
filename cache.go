package deck

import (
	"sync"
)

var globalCache = &cache{
	mu: sync.Mutex{},
	m:  make(map[string]*Image),
}

type cache struct {
	mu sync.Mutex
	m  map[string]*Image
}

func LoadImageCache(key string) (*Image, bool) {
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()
	if v, ok := globalCache.m[key]; ok {
		return v, true
	}
	return nil, false
}

func StoreImageCache(key string, i *Image) {
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()
	if i == nil {
		return
	}
	// compact the cache
	for k, v := range globalCache.m {
		if v.Checksum() == i.Checksum() {
			if globalCache.m[k].modTime.After(i.modTime) {
				globalCache.m[key] = v
				return
			}
			globalCache.m[k] = i
		}
	}
	globalCache.m[key] = i
}
