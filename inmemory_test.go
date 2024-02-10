package main

import (
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
