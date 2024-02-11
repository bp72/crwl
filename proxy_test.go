package main

import (
	"testing"
)

func TestProxy(t *testing.T) {
	p := &Proxy{User: "user", Password: "pass", Host: "127.0.0.1", Port: 8080}
	expected := "user:pass@127.0.0.1:8080"
	if p.String() != expected {
		t.Errorf("Test 'New Proxy. String method' failed. Wrong output. Got %s, expected %s", p.String(), expected)
	}
}
