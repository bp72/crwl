package main

import (
	"bufio"
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

func (q *InmemQueue) Add(t *Task) {
	// TODO: Deprecate Add method
	q.Put(t)
}

func (q *InmemQueue) Put(t *Task) {
	q.lock.Lock()
	defer q.lock.Unlock()

	node := &Node{Value: t}
	q.tail.Next = node
	q.tail = node
	q.size++
}

func (q *InmemQueue) Take() (*Task, error) {
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

func (q *InmemQueue) Get() (*Task, error) {
	// TODO: Deprecate Get method
	return q.Take()
}

func (q *InmemQueue) Size() int64 {
	return int64(q.size + q.in_progress)
}

func (q *InmemQueue) TaskDone() {
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

	for scanner.Scan() {
		items := strings.Split(scanner.Text(), "|||")
		task, err := site.NewTask(items[0], site.MaxDepth-1)
		if err != nil {
			continue
		}
		q.Add(task)
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
