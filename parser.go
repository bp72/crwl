package main

import (
	"bytes"
	"context"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/cactus/go-statsd-client/v5/statsd"
)

type Article struct {
	Title    string
	URL      string
	Category string
}

type LinkParser interface {
	ParseLinks(reader *bytes.Reader, q Queue, t *Task, stats statsd.Statter)
}

type HtmlLinkParser struct {
}

func (p *HtmlLinkParser) ParseLinks(reader *bytes.Reader, q Queue, t *Task, stats statsd.Statter) {
	start := time.Now()
	Log.Info("start parsing", "owner", "HtmlLinkParser")
	defer stats.TimingDuration("crawl.parser", time.Since(start), 1.0, statsd.Tag{"domain", *Domain})
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
					localStart := time.Now()
					q.Put(context.Background(), nt)
					defer stats.TimingDuration("queue.put", time.Since(localStart), 1.0, statsd.Tag{"domain", *Domain})
				}()
			}
		}
	})
}
