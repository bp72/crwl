package main

import (
	"context"
)

type Queue interface {
	Size(ctx context.Context) int64
	Put(ctx context.Context, t *Task)
	Take(ctx context.Context) (*Task, error)
	TaskDone()
}
