package main

import (
	"bytes"
	"context"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/cactus/go-statsd-client/v5/statsd"
	"golang.org/x/net/html"
)

type Article struct {
	Title    string
	URL      string
	Category string
}

type LinkParser interface {
	ParseLinks(reader *bytes.Reader, t *Task)
}

type HtmlLinkParser struct {
	q     Queue
	stats statsd.Statter
}

func (p *HtmlLinkParser) ParseLinks(reader *bytes.Reader, t *Task) {
	start := time.Now()
	Log.Info("start parsing", "owner", "HtmlLinkParser")
	defer p.stats.TimingDuration("crawl.parser", time.Since(start), 1.0, statsd.Tag{"domain", *Domain})
	doc, err := goquery.NewDocumentFromReader(reader)

	if err != nil {
		Log.Error("create document from reader failed", "err", err)
	}

	Log.Info("parse document", "doc", doc)
	Unique := make(map[string]bool)

	doc.Find("a").Each(func(i int, sel *goquery.Selection) {
		Href, _ := sel.Attr("href")
		if _, exists := Unique[Href]; !exists {
			Unique[Href] = true
			nt, err := t.Site.NewTask(Href, t.Depth+1)
			if err == nil {
				go func() {
					Log.Info("new", "task", nt)
					localStart := time.Now()
					p.q.Put(context.Background(), nt)
					defer p.stats.TimingDuration("queue.put", time.Since(localStart), 1.0, statsd.Tag{"domain", *Domain})
				}()
			}
		}
	})
}

type HtmlLinkParser2 struct {
	q     Queue
	stats statsd.Statter
}

func (p *HtmlLinkParser2) ParseLinks(reader *bytes.Reader, t *Task) {
	doc, err := html.Parse(reader)

	if err != nil {
		Log.Error("create document from reader failed", "parser", "HtmlLinkParser2", "err", err)
	}

	// Visit all nodes and extract links
	var links []string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					links = append(links, a.Val)
					newtask, err := t.Site.NewTask(a.Val, t.Depth+1)
					if err == nil {
						p.q.Put(context.Background(), newtask)
					}
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
}
