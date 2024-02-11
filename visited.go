package main

import (
	"context"
	"time"
)

type Visited interface {
	Add(ctx context.Context, uri string)
	Exists(ctx context.Context, uri string) bool
}

func NewVisited(useInternalCache bool) Visited {
	if useInternalCache {
		return NewInmemVisited()
	}

	return NewRedisCache(
		RedisConnectionParams{
			Addr:     *RedisAddr,
			Base:     *RedisBase,
			Password: *RedisPass,
			Domain:   *Domain,
			Timeout:  2000 * time.Millisecond,
		},
	)
}
