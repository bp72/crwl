package main

// "context"

type Storage interface {
	IsVisited() bool
	IsSeen() bool
	Add(Host int, Uri string)
}
