package deck

import (
	"sync"
)

var globalCache = &cache{
	mu: sync.RWMutex{},
	m:  make(map[string]*internalImage),
}

type cache struct {
	mu sync.RWMutex
	m  map[string]*internalImage
}

func LoadImageCache(key string) (*Image, bool) {
	globalCache.mu.RLock()
	defer globalCache.mu.RUnlock()

	if v, ok := globalCache.m[key]; ok {
		var img Image
		if err := v.toImage(&img); err == nil {
			return &img, true
		}
	}
	return nil, false
}

func StoreImageCache(key string, i *Image) {
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()

	globalCache.m[key] = i.toInternal()
}
