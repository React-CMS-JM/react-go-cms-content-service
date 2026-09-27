// Package cache stores the in-memory popular category and tag lists.
package cache

import (
	"sync"
	"time"

	"react-go-cms-content-service/internal/domain/entity"
)

const (
	popularTTL     = 5 * time.Minute
	maxPopularKeys = 32
)

type entry struct {
	at         time.Time
	categories []entity.Taxonomy
	tags       []entity.Taxonomy
	kind       string
}

// Snapshot is one cached popular list.
type Snapshot struct {
	Categories []entity.Taxonomy
	Tags       []entity.Taxonomy
	Kind       string
}

// Popular is the shared category and tag popular-list cache.
// One instance is injected into the category and tag repositories so a clear drops both lists.
type Popular struct {
	mu    sync.Mutex
	items map[string]entry
}

// NewPopular builds an empty popular-list cache.
func NewPopular() *Popular {
	return &Popular{items: map[string]entry{}}
}

// Get returns a cached snapshot when the key is present and younger than the TTL.
func (c *Popular) Get(key string) (Snapshot, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item, ok := c.items[key]
	if !ok || time.Since(item.at) > popularTTL {
		return Snapshot{}, false
	}
	return Snapshot{Categories: item.categories, Tags: item.tags, Kind: item.kind}, true
}

// Put stores a snapshot and resets the map once it reaches the entry cap.
func (c *Popular) Put(key string, snap Snapshot) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.items) >= maxPopularKeys {
		c.items = map[string]entry{}
	}
	c.items[key] = entry{
		at:         time.Now(),
		categories: snap.Categories,
		tags:       snap.Tags,
		kind:       snap.Kind,
	}
}

// Clear drops every cached popular list.
func (c *Popular) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = map[string]entry{}
}
