package main

import (
	"errors"
	"sync"
)

type Node struct {
	Value *Task
	Next  *Node
}

type Queue struct {
	head        *Node
	tail        *Node
	size        int
	in_progress int
	lock        sync.Mutex
}

func (q *Queue) Add(t *Task) {
	q.lock.Lock()
	defer q.lock.Unlock()
	node := &Node{Value: t}
	if q.head == nil {
		q.head = node
		q.tail = node
	} else {
		q.tail.Next = node
		q.tail = node
	}
	q.size++
}

func (q *Queue) Get() (*Task, error) {
	q.lock.Lock()
	defer q.lock.Unlock()
	if q.head == nil {
		return nil, errors.New("Empty queue")
	}

	node := q.head
	q.head = node.Next
	q.size--
	q.in_progress++
	return node.Value, nil
}

func (q *Queue) Size() int {
	return q.size + q.in_progress
}

func (q *Queue) TaskDone() {
	q.lock.Lock()
	defer q.lock.Unlock()

	q.in_progress--
}
