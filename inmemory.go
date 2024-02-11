package main

import (
	"bufio"
	"context"
	"errors"
	"os"
	"strings"
	"sync"
)

type Node struct {
	Value *Task
	Next  *Node
}

type InmemQueue struct {
	head        *Node
	tail        *Node
	size        int
	in_progress int
	lock        sync.Mutex
}

func (q *InmemQueue) Add(ctx context.Context, t *Task) {
	// TODO: Deprecate Add method
	q.Put(ctx, t)
}

func (q *InmemQueue) Put(ctx context.Context, t *Task) {
	q.lock.Lock()
	defer q.lock.Unlock()

	node := &Node{Value: t}
	q.tail.Next = node
	q.tail = node
	q.size++
}

func (q *InmemQueue) Take(ctx context.Context) (*Task, error) {
	q.lock.Lock()
	defer q.lock.Unlock()

	if q.head.Next == nil {
		return nil, errors.New("empty queue")
	}

	node := q.head.Next
	q.head.Next = node.Next
	q.size--
	q.in_progress++

	return node.Value, nil
}

func (q *InmemQueue) Get(ctx context.Context) (*Task, error) {
	// TODO: Deprecate Get method
	return q.Take(ctx)
}

func (q *InmemQueue) Size(ctx context.Context) int64 {
	return int64(q.size + q.in_progress)
}

func (q *InmemQueue) TaskDone(ctx context.Context) {
	q.lock.Lock()
	defer q.lock.Unlock()

	q.in_progress--
}

func (q *InmemQueue) LoadFromFile(site *Site, Filepath string) error {
	f, err := os.Open(Filepath)

	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	ctx := context.Background()
	for scanner.Scan() {
		items := strings.Split(scanner.Text(), "|||")
		task, err := site.NewTask(items[0], site.MaxDepth-1)
		if err != nil {
			continue
		}
		q.Put(ctx, task)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func NewInmemQueue() *InmemQueue {
	q := &InmemQueue{}

	q.head = &Node{}
	q.tail = q.head

	return q
}

type InmemVisited struct {
	items map[string]bool
	lock  sync.RWMutex
}

func (v *InmemVisited) Add(ctx context.Context, uri string) {
	v.lock.Lock()
	defer v.lock.Unlock()

	v.items[uri] = true
}

func (v *InmemVisited) Exists(ctx context.Context, uri string) bool {
	v.lock.RLock()
	defer v.lock.RUnlock()

	if _, exists := v.items[uri]; exists {
		return true
	}

	return false
}

func NewInmemVisited() *InmemVisited {
	v := &InmemVisited{
		items: make(map[string]bool),
	}

	return v
}
