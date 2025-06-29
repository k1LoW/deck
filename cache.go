package deck

import (
	"sync"
)

var globalCache = &cache{}

type cache struct {
	m sync.Map
}

func LoadImageCache(key string) (*Image, bool) {
	if v, ok := globalCache.m.Load(key); ok {
		if i, ok := v.(*Image); ok {
			return i, true
		}
	}
	return nil, false
}

func StoreImageCache(key string, i *Image) {
	if i == nil {
		return
	}
	globalCache.m.Store(key, i)
}
