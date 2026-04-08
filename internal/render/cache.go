package render

import (
	"fmt"
	"sync"
	"time"
)

type Cache struct {
	mu    sync.RWMutex
	items map[string]Document
}

func NewCache() *Cache {
	return &Cache{
		items: make(map[string]Document),
	}
}

func (c *Cache) Get(path string, width int, modTime time.Time) (Document, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	doc, ok := c.items[cacheKey(path, width, modTime)]
	return doc, ok
}

func (c *Cache) Set(doc Document) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[cacheKey(doc.Path, doc.Width, doc.ModTime)] = doc
}

func cacheKey(path string, width int, modTime time.Time) string {
	return fmt.Sprintf("%s|%d|%d", path, width, modTime.UnixNano())
}
