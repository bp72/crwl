package main

import (
	"context"
	"sync"
)

type Visited interface {
	Add(ctx context.Context, uri string)
	Exists(ctx context.Context, uri string) bool
}

type InmemVisited struct {
	items map[string]bool
	lock  sync.RWMutex
}

func (v InmemVisited) Add(uri string) {
	v.lock.Lock()
	defer v.lock.Unlock()

	v.items[uri] = true
}

func (v InmemVisited) Exists(uri string) bool {
	v.lock.RLock()
	defer v.lock.RUnlock()

	if _, exists := v.items[uri]; exists {
		return true
	}

	return false
}
