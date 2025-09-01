package deck

import (
	"sync"
	"testing"
)

func clearCache() {
	globalCache = &cache{
		mu: sync.RWMutex{},
		m:  make(map[string]*internalImage),
	}
}

func TestLoadImageCache(t *testing.T) {
	clearCache()
	defer clearCache() // don't use t.Cleanup here, want to clear before next test

	buf := dummyPNG(t)
	i, err := NewImageFromCodeBlock(buf)
	if err != nil {
		t.Fatalf("TestLoadImageCache setup failed: %v", err)
	}

	const key = "test_key.png"
	i.url = key
	i.link = ""
	StoreImageCache(key, i)

	cached1, ok := LoadImageCache(key)
	if !ok {
		t.Fatalf("LoadImageCache failed to find cached image")
	}
	if !cached1.Equivalent(i) {
		t.Errorf("Loaded image is not equivalent to original")
	}
	cached1.link = "modified_link"

	cached2, ok := LoadImageCache(key)
	if !ok {
		t.Fatalf("LoadImageCache failed to find cached image on second load")
	}
	// check no side effects from modifying
	if cached2.link != "" || !cached2.Equivalent(i) {
		t.Errorf("Loaded image is not equivalent to original on second load")
	}
}
