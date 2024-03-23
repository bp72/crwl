package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/cactus/go-statsd-client/v5/statsd"
	"github.com/valyala/fasthttp"
	"golang.org/x/net/html"
)

type Crawler struct {
	TTL           time.Duration
	UserAgent     string
	Stack         []*Task
	TaskChan      chan *Task
	Wg            sync.WaitGroup
	Q             Queue
	V             Visited
	P             LinkParser
	DoSaveContent bool
	StorePath     string
	statsd        statsd.Statter
	Pool          sync.Pool
}

func (c *Crawler) EnqeueMany(t *Task, ch <-chan *Link, wg *sync.WaitGroup) (int, int) {
	start := time.Now()
	defer c.statsd.TimingDuration("queue.enqueuemany", time.Since(start), 1.0, statsd.Tag{"domain", *Domain})

	added := 0
	total := 0

	for link := range ch {
		total++
		_, err := t.Site.NewTask(link.Href, t.Depth+1)

		if err != nil {
			wg.Done()
			continue
		}

		go func() {
			defer wg.Done()
			// c.Enqueue(nt)
		}()

		added++
	}

	return added, total
}

func (c *Crawler) Crawl(WorkerNo int) {
	c.statsd.Inc("crawl.req.total", 1, 1.0, statsd.Tag{"domain", *Domain})
	Log.Info("crawl.req.total", "qsize", c.Q.Size(context.Background()))
	start := time.Now()
	task, err := c.Q.Take(context.Background())
	c.statsd.TimingDuration("queue.take", time.Since(start), 1.0, statsd.Tag{"domain", *Domain})

	if err != nil {
		time.Sleep(1 * time.Second)
		Log.Error("error", "w", WorkerNo, "err", err, "qsize", c.Q.Size(context.Background()))
		c.statsd.Inc("err", 1, 1.0, statsd.Tag{"domain", *Domain}, statsd.Tag{"type", "queue-take"})
		return
	}

	startCheck := time.Now()
	if c.V.Exists(context.Background(), task.Uri); err != nil {
		Log.Info("already visited", "uri", task.Uri)
		c.statsd.Inc("crawl.req.alreadyvisited", 1, 1.0, statsd.Tag{"domain", *Domain})
		return
	}
	c.statsd.TimingDuration("cache.check", time.Since(startCheck), 1.0, statsd.Tag{"domain", *Domain})

	Log.Info("crawl task", "w", WorkerNo, "task.uri", task.Uri, "task.depth", task.Depth, "qsize", c.Q.Size(context.Background()))

	reader, err := c.Get(task.GetUrl())
	if err != nil {
		c.Q.TaskDone(context.Background())
		Log.Error("error", "w", WorkerNo, "err", err)
		c.statsd.Inc("err", 1, 1.0, statsd.Tag{"domain", *Domain}, statsd.Tag{"type", "http-request"})
		return
	}

	go func() {
		startPut := time.Now()
		c.V.Add(context.Background(), task.Uri)
		c.statsd.TimingDuration("cache.put", time.Since(startPut), 1.0, statsd.Tag{"domain", *Domain})
	}()

	c.P.ParseLinks(reader, task)

	saveStart := time.Now()
	if c.DoSaveContent {
		go func() {
			dirPath := fmt.Sprintf("%s/%s", c.StorePath, task.GetSubTree())
			err := os.MkdirAll(dirPath, os.ModePerm)
			if err != nil {
				Log.Error("error", "w", WorkerNo, "err", err)
			}

			f, err := os.Create(fmt.Sprintf("%s/%s", dirPath, task.GetFilename()))

			if err != nil {
				panic(err)
			}
			defer f.Close()

			reader.Seek(0, 0)
			bytesWrote, err := reader.WriteTo(f)
			if err != nil {
				panic(err)
			}
			Log.Info("save content", "w", WorkerNo, "path", c.StorePath, "bytes", bytesWrote)
			c.statsd.Inc("crawl.req.saved", 1, 1.0, statsd.Tag{"domain", *Domain})
		}()
	}
	c.statsd.TimingDuration("save", time.Since(saveStart), 1.0, statsd.Tag{"domain", *Domain})

	c.Q.TaskDone(context.Background())
	c.statsd.TimingDuration("crawl", time.Since(start), 1.0, statsd.Tag{"domain", *Domain})
	c.statsd.Inc("crawl.req.ok", 1, 1.0, statsd.Tag{"domain", *Domain})
	Log.Info("task done", "w", WorkerNo, "exec-time", time.Since(start))
}

func (c *Crawler) Get(Url string) (*bytes.Reader, error) {
	c.statsd.Inc("crawl.client.total", 1, 1.0, statsd.Tag{"domain", *Domain})
	start := time.Now()
	req := fasthttp.AcquireRequest()

	req.SetRequestURI(Url)
	req.Header.SetMethod(fasthttp.MethodGet)
	req.Header.SetUserAgent(c.UserAgent)

	resp := fasthttp.AcquireResponse()

	client := c.Pool.Get().(*fasthttp.Client)
	err := client.DoRedirects(req, resp, 5)

	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	if err != nil {
		c.statsd.Inc("crawl.client.err", 1, 1.0, statsd.Tag{"domain", *Domain})
		return nil, err
	}

	c.statsd.Inc("crawl.client.code", 1, 1.0, statsd.Tag{"domain", *Domain}, statsd.Tag{"http_code", strconv.Itoa(resp.StatusCode())})

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("url=%s code=%d", Url, resp.StatusCode())
	}

	c.statsd.TimingDuration("crawl.client", time.Since(start), 1.0, statsd.Tag{"domain", *Domain})
	c.statsd.Inc("crawl.client.ok", 1, 1.0, statsd.Tag{"domain", *Domain})
	return bytes.NewReader(resp.Body()), nil
}

func getHref(t html.Token) (ok bool, href string) {
	for _, a := range t.Attr {
		if a.Key == "href" {
			href = a.Val
			ok = true
		}
	}

	return
}

func (c *Crawler) Parse(reader *bytes.Reader, ch chan<- *Link, wg *sync.WaitGroup) []*Link {
	start := time.Now()
	defer c.statsd.Timing("crawl.parser", int64(time.Since(start)), 1.0, statsd.Tag{"domain", *Domain})
	tokenizer := html.NewTokenizer(reader)
	links := make([]*Link, 0)
	var l *Link

	for {
		tt := tokenizer.Next()

		switch {
		case tt == html.ErrorToken:
			return links
		case tt == html.StartTagToken:
			t := tokenizer.Token()

			// Check if the token is an <a> tag
			isAnchor := t.Data == "a"
			if !isAnchor {
				continue
			}

			l = NewLink()

			// Extract the href value, if there is one
			ok, url := getHref(t)
			if !ok {
				continue
			}

			l.Href = url
		case tt == html.TextToken:
			t := tokenizer.Token()
			if l != nil && t.Data != "" {
				l.Anchor = append(l.Anchor, t.Data)
			}
		case tt == html.EndTagToken:
			t := tokenizer.Token()
			isAnchor := t.Data == "a"
			if !isAnchor {
				continue
			}
			if l != nil {
				// ch <- l
				// wg.Add(1)
				links = append(links, l)
				l = nil
			}
		}
	}
}

func (c *Crawler) Run(MaxGoroutine int) {
	Log.Info("start crawler.run", "max", MaxGoroutine)

	guard := make(chan struct{}, MaxGoroutine)

	i := 0
	for c.Q.Size(context.Background()) > 0 {
		guard <- struct{}{}
		go func(no int) {
			c.Crawl(no)
			<-guard
		}(i)
		i++
		if i >= *Limit {
			break
		}
	}

	Log.Info("stop crawler.run", "max", MaxGoroutine)
}

func (c *Crawler) Enqueue(t *Task) error {
	start := time.Now()
	defer c.statsd.TimingDuration("queue.enqueue", time.Since(start), 1.0, statsd.Tag{"domain", *Domain})

	startCheck := time.Now()
	if c.V.Exists(context.Background(), t.Uri) {
		return fmt.Errorf("uri=%s already visited or enqueued", t.Uri)
	}
	c.statsd.TimingDuration("cache.check", time.Since(startCheck), 1.0, statsd.Tag{"domain", *Domain})

	startPut := time.Now()
	c.V.Add(context.Background(), t.Uri)
	c.statsd.TimingDuration("cache.put", time.Since(startPut), 1.0, statsd.Tag{"domain", *Domain})

	go func() {
		startPut = time.Now()
		c.Q.Put(context.Background(), t)
		c.statsd.TimingDuration("queue.put", time.Since(startPut), 1.0, statsd.Tag{"domain", *Domain})
		c.statsd.Gauge("crawl.queue.size", int64(c.Q.Size(context.Background())), 1.0, statsd.Tag{"domain", *Domain})
	}()

	return nil
}

func NewCrawler(Q Queue, V Visited, StorePath string, statsd statsd.Statter) *Crawler {
	ttl, _ := time.ParseDuration("30m")

	readTimeout, _ := time.ParseDuration("6000ms")
	writeTimeout, _ := time.ParseDuration("6000ms")
	maxIdleConnDuration, _ := time.ParseDuration("1h")
	maxConnWaitTimeout, _ := time.ParseDuration("6000ms")

	cwlr := &Crawler{
		TTL:       ttl,
		UserAgent: GetUserAgent(*UseGooglebot),
		// Clients:       make([]*fasthttp.Client, len(proxies)),
		Stack:         make([]*Task, 0),
		TaskChan:      make(chan *Task),
		V:             V,
		Q:             Q,
		P:             &HtmlLinkParser{q: Q, stats: statsd},
		DoSaveContent: !*DoNotStore,
		StorePath:     StorePath,
		Pool: sync.Pool{
			New: func() interface{} {
				// proxy := proxies[LastProxy%len(proxies)]
				LastProxy++
				// dial := fasthttpproxy.FasthttpHTTPDialerTimeout(proxy, writeTimeout)

				// if *DoNotUseProxy {
				dial := (&fasthttp.TCPDialer{
					Concurrency:      4096,
					DNSCacheDuration: time.Hour,
				}).Dial
				// }
				return &fasthttp.Client{
					ReadTimeout:                   readTimeout,
					WriteTimeout:                  writeTimeout,
					MaxIdleConnDuration:           maxIdleConnDuration,
					MaxConnWaitTimeout:            maxConnWaitTimeout,
					NoDefaultUserAgentHeader:      true,
					DisableHeaderNamesNormalizing: true,
					DisablePathNormalizing:        true,
					Dial:                          dial,
				}
			},
		},
	}

	// for pos, proxyStr := range proxies {
	// 	dial := fasthttpproxy.FasthttpHTTPDialerTimeout(proxyStr, writeTimeout)
	// 	if *DoNotUseProxy {
	// 		dial = (&fasthttp.TCPDialer{
	// 			Concurrency:      4096,
	// 			DNSCacheDuration: time.Hour,
	// 		}).Dial
	// 	}
	// 	cwlr.Clients[pos] = &fasthttp.Client{
	// 		ReadTimeout:                   readTimeout,
	// 		WriteTimeout:                  writeTimeout,
	// 		MaxIdleConnDuration:           maxIdleConnDuration,
	// 		MaxConnWaitTimeout:            maxConnWaitTimeout,
	// 		NoDefaultUserAgentHeader:      true,
	// 		DisableHeaderNamesNormalizing: true,
	// 		DisablePathNormalizing:        true,
	// 		Dial:                          dial,
	// 	}
	// }

	return cwlr
}
