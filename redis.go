package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisConnectionParams struct {
	Addr     string
	Password string
	Base     int
	Domain   string
	Timeout  time.Duration
}

type RedisQueue struct {
	Domain     string
	Key        string
	Timeout    time.Duration
	Client     *redis.Client
	lock       sync.Mutex
	InProgress int64
}

func (q *RedisQueue) Size(ctx context.Context) int64 {
	ctx, cancel := context.WithTimeout(ctx, q.Timeout)
	defer cancel()

	if val, err := q.Client.LLen(ctx, q.Key).Result(); err == nil {
		return val + q.InProgress
	}

	return q.InProgress
}

func (q *RedisQueue) Put(ctx context.Context, t *Task) {
	ctx, cancel := context.WithTimeout(ctx, q.Timeout)
	defer cancel()

	b, err := json.Marshal(t)
	if err != nil {
		fmt.Println(err)
		return
	}

	if err := q.Client.RPush(ctx, q.Key, b).Err(); err != nil {
		panic(err)
	}
}

func (q *RedisQueue) Take(ctx context.Context) (*Task, error) {
	ctx, cancel := context.WithTimeout(ctx, q.Timeout)
	defer cancel()

	q.lock.Lock()
	defer q.lock.Unlock()

	if val, err := q.Client.LPop(ctx, q.Key).Result(); err != nil {
		return nil, err
	} else {
		q.InProgress++
		var task *Task
		_ = json.Unmarshal([]byte(val), &task)
		return task, nil
	}
}

func (q *RedisQueue) TaskDone(ctx context.Context) {
	q.lock.Lock()
	q.InProgress--
	q.lock.Unlock()
}

func NewRedisQueue(p RedisConnectionParams) *RedisQueue {
	r := &RedisQueue{
		Domain:  p.Domain,
		Key:     fmt.Sprintf("queue:%s", p.Domain),
		Timeout: p.Timeout,
		Client: redis.NewClient(&redis.Options{
			Addr:         p.Addr,
			Password:     p.Password,
			DB:           p.Base,
			WriteTimeout: time.Second * 5,
			ReadTimeout:  time.Second * 5,
			DialTimeout:  time.Second * 5,
			PoolTimeout:  time.Second * 1,
			PoolSize:     1000,
		}),
	}

	return r
}

type RedisCache struct {
	Domain  string
	Key     string
	Timeout time.Duration
	TTL     time.Duration
	Client  *redis.Client
}

func (c *RedisCache) Add(ctx context.Context, uri string) {
	ctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	key := fmt.Sprintf("%s:%s", c.Key, uri)
	Log.Info("set", "key", key)
	if _, err := c.Client.Set(ctx, key, uri, c.TTL).Result(); err != nil {
		panic(err)
	}
}

func (c *RedisCache) Exists(ctx context.Context, uri string) bool {
	ctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	key := fmt.Sprintf("%s:%s", c.Key, uri)

	if _, err := c.Client.Get(ctx, key).Result(); err != nil {
		return false
	}

	return true
}

func NewRedisCache(p RedisConnectionParams) *RedisCache {
	r := &RedisCache{
		Domain:  p.Domain,
		Key:     fmt.Sprintf("cache:%s", p.Domain),
		Timeout: p.Timeout,
		Client: redis.NewClient(&redis.Options{
			Addr:         p.Addr,
			Password:     p.Password,
			DB:           p.Base,
			WriteTimeout: time.Second * 5,
			ReadTimeout:  time.Second * 5,
			DialTimeout:  time.Second * 5,
			PoolTimeout:  time.Second * 1,
			PoolSize:     1000,
		}),
		TTL: 24 * 7 * time.Hour,
	}

	return r
}
