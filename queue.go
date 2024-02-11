package main

import (
	"context"
	"time"
)

type Queue interface {
	Size(ctx context.Context) int64
	Put(ctx context.Context, t *Task)
	Take(ctx context.Context) (*Task, error)
	TaskDone(ctx context.Context)
}

func NewQueue(useInternalQueue bool) Queue {
	if useInternalQueue {
		return NewInmemQueue()
	}

	return NewRedisQueue(
		RedisConnectionParams{
			Addr:     *RedisAddr,
			Base:     *RedisBase,
			Password: *RedisPass,
			Domain:   *Domain,
			Timeout:  2000 * time.Millisecond,
		},
	)
}
