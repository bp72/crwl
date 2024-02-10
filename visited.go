package main

import (
	"context"
)

type Visited interface {
	Add(ctx context.Context, uri string)
	Exists(ctx context.Context, uri string) bool
}
