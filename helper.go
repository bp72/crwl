package main

import (
	"fmt"
	"path"
	"strings"
)

type Site struct {
	BaseUrl       string
	MaxDepth      int
	KeywordPrefix string
}

func (s *Site) NewTask(Uri string, Depth int) (*Task, error) {
	if s.MaxDepth <= Depth {
		return nil, fmt.Errorf("MaxDepth for %s reached: %d", s.BaseUrl, s.MaxDepth)
	}

	if strings.HasPrefix(Uri, s.BaseUrl) {
		Uri = strings.Replace(Uri, s.BaseUrl, "", -1)
	}

	return &Task{Site: s, Uri: Uri, Depth: Depth}, nil
}

type Task struct {
	Uri   string
	Site  *Site
	Depth int
}

func (t *Task) GetUrl() string {
	if t.Uri[0] == '/' {
		return fmt.Sprintf("%s%s", t.Site.BaseUrl, t.Uri)
	}
	return fmt.Sprintf("%s/%s", t.Site.BaseUrl, t.Uri)
}

func (t *Task) GetSubTree() string {
	filename := path.Base(t.Uri)

	if len(filename) == 0 {
		return "i/n/d"
	}

	if len(filename) == 1 {
		return filename + "/0/0"
	}

	if len(filename) == 2 {
		return filename[:1] + "/" + filename[1:2] + "/0"
	}

	return filename[:1] + "/" + filename[1:2] + "/" + filename[2:3]
}

func (t *Task) GetFilename() string {
	if t.Uri == "/" || t.Uri == "" {
		return "index"
	}

	filename := strings.Replace(t.Uri, "/", "__", -1)
	if len(filename) > 255 {
		return filename[:255]
	}
	return filename
}

type Link struct {
	Href   string
	Anchor []string
}

func NewLink() *Link {
	l := Link{
		Anchor: make([]string, 0),
	}
	return &l
}

func (l *Link) GetAnchor() string {
	return strings.TrimSpace(strings.Join(l.Anchor, " "))
}

func (l *Link) IsKeyword(Prefix string) bool {
	return strings.HasPrefix(l.Href, Prefix)
}
