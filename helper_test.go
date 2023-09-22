package main

import (
	"testing"
)

func TestTask(t *testing.T) {

	task := &Task{
		Uri: "/subdir/name-of-uri",
		Site: &Site{
			BaseUrl:       "http://domain.com",
			MaxDepth:      3,
			KeywordPrefix: "/db",
		},
		Depth: 0,
	}

	expectUrl := "http://domain.com/subdir/name-of-uri"
	gotUrl := task.GetUrl()

	if gotUrl != expectUrl {
		t.Errorf("Test Task.GetUrl failed. Got %q, expected %q", gotUrl, expectUrl)
	}

	expectedSubdirTree := "n/a/m"
	gotSubdirTree := task.GetSubTree()

	if gotSubdirTree != expectedSubdirTree {
		t.Errorf("Test Task.GetSubdirTree() failed. Got %q, expected %q", gotSubdirTree, expectedSubdirTree)
	}

	expectedFilename := "__subdir__name-of-uri"
	gotFilename := task.GetFilename()

	if gotFilename != expectedFilename {
		t.Errorf("Test Task.GetFilename() failed. Got %q, expected %q", gotFilename, expectedFilename)
	}
}
