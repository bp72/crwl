package main

import (
	"context"
	"testing"
)

func createTestSite(maxDepth int) *Site {
	return &Site{
		BaseUrl:       "http://example.com",
		MaxDepth:      maxDepth,
		KeywordPrefix: "/pref",
	}
}

func createTestTask(uri string, depth int, site *Site) *Task {
	return &Task{
		Uri:   uri,
		Site:  site,
		Depth: depth,
	}
}

func TestQueuePut(t *testing.T) {
	site := createTestSite(3)
	tasks := []*Task{
		createTestTask("uri1", 0, site),
		createTestTask("uri2", 0, site),
		createTestTask("uri3", 0, site),
	}

	q := NewInmemQueue()
	task1 := tasks[0]

	for pos, task := range tasks {
		q.Put(task)
		if q.Size() != int64(pos+1) {
			t.Errorf("Test 'New Queue. Put task' failed. Invalid size. Got %d, expected %d", q.Size(), pos+1)
		}
		if q.head.Next.Value != task1 {
			t.Errorf("Test 'New Queue. Put task' failed. Invalid head item. Got %v, expected %v", q.head.Next.Value, task1)
		}
		if q.tail.Value != task {
			t.Errorf("Test 'New Queue. Put task' failed. Invalid tail item. Got %v, expected %v", q.head.Next.Value, task)
		}
	}
}

func TestQueueTake(t *testing.T) {
	site := createTestSite(3)
	tasks := []*Task{
		createTestTask("uri1", 0, site),
		createTestTask("uri2", 0, site),
		createTestTask("uri3", 0, site),
	}

	q := NewInmemQueue()

	for _, task := range tasks {
		q.Put(task)
	}

	for pos, expectedTask := range tasks {
		task, err := q.Take()
		if err != nil {
			t.Errorf("Test 'New Queue. Take task' failed. Unexpected err %v", err)
		}
		if task != expectedTask {
			t.Errorf("Test 'New Queue. Take task' failed. Invalid task. Got %v, expected %v", task, expectedTask)
		}
		if q.in_progress != pos+1 {
			t.Errorf("Test 'New Queue. Take task' failed. Invalid in progress task number. Got %d, expected %d", q.in_progress, pos+1)
		}
	}

	for pos, _ := range tasks {
		q.TaskDone()
		if q.in_progress != len(tasks)-pos-1 {
			t.Errorf("Test 'New Queue. Take task' failed. Invalid in progress task number. Got %d, expected %d", q.in_progress, len(tasks)-pos)
		}
	}
}

func TestInmemVisited(t *testing.T) {
	v := NewInmemVisited()
	ctx := context.Background()

	v.Add(ctx, "1")

	if len(v.items) != 1 {
		t.Errorf("Test 'InmemVisited. Add' failed. Invalid size. Got %d, expected %d", len(v.items), 1)
	}

	v.Add(ctx, "1")
	if len(v.items) != 1 {
		t.Errorf("Test 'InmemVisited. Add' failed. Invalid size. Got %d, expected %d", len(v.items), 1)
	}

	v.Add(ctx, "2")
	if len(v.items) != 2 {
		t.Errorf("Test 'InmemVisited. Add' failed. Invalid size. Got %d, expected %d", len(v.items), 2)
	}

	exists := v.Exists(ctx, "1")
	if exists != true {
		t.Errorf("Test 'InmemVisited.Exists' failed. Invalid result. Got %v, expected %v", exists, true)
	}

	exists = v.Exists(ctx, "2")
	if exists != true {
		t.Errorf("Test 'InmemVisited.Exists' failed. Invalid result. Got %v, expected %v", exists, true)
	}

	exists = v.Exists(ctx, "3")
	if exists != false {
		t.Errorf("Test 'InmemVisited.Exists' failed. Invalid result. Got %v, expected %v", exists, false)
	}
}
