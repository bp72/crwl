package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
	"golang.org/x/net/html"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"
)

const userAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/113.0.0.0 Safari/537.36"
const gbUserAgent = "Mozilla/5.0 (Linux; Android 6.0.1; Nexus 5X Build/MMB29P) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.5735.179 Mobile Safari/537.36 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"

var (
	Log           *slog.Logger
	Domain        = flag.String("domain", "", "domain to scan")
	MaxDepth      = flag.Int("max-depth", 7, "set max depth for crawling")
	MaxWorkers    = flag.Int("max-workers", 20, "set max concurrent workers")
	UseGooglebot  = flag.Bool("use-google-bot", false, "Run as Googlebot mode")
	DoNotUseProxy = flag.Bool("do-not-use-proxy", false, "Do not use proxy")
	DoNotStore    = flag.Bool("do-not-store", false, "Do not store content")
	UseHttp       = flag.Bool("use-http", false, "use http proto")
	Limit         = flag.Int("max-crawl", 100000, "set max amount of page to crawl")
)

func init() {
	Log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

type Crawler struct {
	TTL           time.Duration
	UserAgent     string
	Clients       []*fasthttp.Client
	Stack         []*Task
	TaskChan      chan *Task
	Wg            sync.WaitGroup
	Q             *Queue
	DoSaveContent bool
	StorePath     string
	Visited       map[string]bool
	lock          sync.Mutex
}

func (c *Crawler) GetRandomClient() *fasthttp.Client {
	return c.Clients[rand.Intn(len(c.Clients))]
}

func (c *Crawler) Crawl(WorkerNo int) {
	task, err := c.Q.Get()
	if err != nil {
		time.Sleep(1 * time.Second)
		Log.Error("error", "w", WorkerNo, "err", err)
		return
	}

	Log.Info("crawl task", "w", WorkerNo, "task.uri", task.Uri, "task.depth", task.Depth, "qsize", c.Q.Size())

	reader, err := c.Get(task.GetUrl())
	if err != nil {
		c.Q.TaskDone()
		Log.Error("error", "w", WorkerNo, "err", err)
		return
	}

	links := c.Parse(reader)
	added := 0
	for _, link := range links {
		nt, err := task.Site.NewTask(link.Href, task.Depth+1)
		if err != nil {
			continue
		}
		err = c.Enqueue(nt)
		if err == nil {
			added++
		}
	}
	Log.Info("new tasks", "w", WorkerNo, "added-urls", added, "total-urls", len(links))

	if c.DoSaveContent {
		err := os.MkdirAll(c.StorePath, os.ModePerm)
		if err != nil {
			Log.Error("error", "w", WorkerNo, "err", err)
		}

		f, err := os.Create(fmt.Sprintf("%s/%s", c.StorePath, task.GetFilename()))

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
	}

	c.Q.TaskDone()
}

func (c *Crawler) Get(Url string) (*bytes.Reader, error) {
	req := fasthttp.AcquireRequest()
	req.SetRequestURI(Url)
	req.Header.SetMethod(fasthttp.MethodGet)
	req.Header.SetUserAgent(c.UserAgent)
	resp := fasthttp.AcquireResponse()
	client := c.GetRandomClient()
	err := client.DoRedirects(req, resp, 5)
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("url=%s code=%d", Url, resp.StatusCode())
	}

	return bytes.NewReader(resp.Body()), nil
}

func getHref(t html.Token) (ok bool, href string) {
	// Iterate over token attributes until we find an "href"
	for _, a := range t.Attr {
		if a.Key == "href" {
			href = a.Val
			ok = true
		}
	}

	// fmt.Printf("getHref %+v %+v\n", t, tt)
	// "bare" return will return the variables (ok, href) as
	// defined in the function definition
	return
}

func (c *Crawler) Parse(reader *bytes.Reader) []*Link {
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

			// anchorTextToken := tokenizer.Next()
			// Make sure the url begines in http**
			// hasProto := strings.Index(url, "http") == 0
			// log.Printf("%s %v %v %v\n", url, hasProto, anchorTextToken, t)
			// if hasProto {
			// 	ch <- url
			// }
		case tt == html.TextToken:
			t := tokenizer.Token()
			// log.Printf("%s\n", t)
			if l != nil && t.Data != "" {
				l.Anchor = append(l.Anchor, t.Data)
			}
		case tt == html.EndTagToken:
			t := tokenizer.Token()
			// Check if the token is an <a> tag
			isAnchor := t.Data == "a"
			if !isAnchor {
				continue
			}
			if l != nil {
				links = append(links, l)

				// fmt.Printf("%v %s %v\n", l, l.GetAnchor(), l.IsKeyword("/db/"))
				l = nil
			}
			// log.Printf("%v\n", t)
		}
	}

	return links

}

func (c *Crawler) Run(MaxGoroutine int) {
	Log.Info("start crawler.run")
	guard := make(chan struct{}, MaxGoroutine)
	i := 0

	for c.Q.Size() > 0 {
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
}

func (c *Crawler) Enqueue(t *Task) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if _, exists := c.Visited[t.Uri]; exists {
		return fmt.Errorf("uri=%s already visited or enqueued", t.Uri)
	}
	c.Visited[t.Uri] = true
	c.Q.Add(t)

	return nil
}

func NewCrawler() *Crawler {
	ttl, _ := time.ParseDuration("30m")

	readTimeout, _ := time.ParseDuration("6000ms")
	writeTimeout, _ := time.ParseDuration("6000ms")
	maxIdleConnDuration, _ := time.ParseDuration("1h")
	maxConnWaitTimeout, _ := time.ParseDuration("6000ms")

	cwlr := &Crawler{
		TTL:           ttl,
		UserAgent:     userAgent,
		Clients:       make([]*fasthttp.Client, len(proxies)),
		Stack:         make([]*Task, 0),
		TaskChan:      make(chan *Task),
		Visited:       make(map[string]bool),
		Q:             &Queue{},
		DoSaveContent: !*DoNotStore,
		StorePath:     fmt.Sprintf("/home/bp/crawler/%s", *Domain),
	}

	for pos, proxyStr := range proxies {
		dial := fasthttpproxy.FasthttpHTTPDialerTimeout(proxyStr, writeTimeout)
		if *DoNotUseProxy {
			dial = (&fasthttp.TCPDialer{
				Concurrency:      4096,
				DNSCacheDuration: time.Hour,
			}).Dial
		}
		cwlr.Clients[pos] = &fasthttp.Client{
			ReadTimeout:                   readTimeout,
			WriteTimeout:                  writeTimeout,
			MaxIdleConnDuration:           maxIdleConnDuration,
			MaxConnWaitTimeout:            maxConnWaitTimeout,
			NoDefaultUserAgentHeader:      true,
			DisableHeaderNamesNormalizing: true,
			DisablePathNormalizing:        true,
			Dial:                          dial,
		}
	}

	return cwlr
}

func main() {
	flag.Parse()
	Log.Info("start crawler")

	crawler := NewCrawler()

	if *UseGooglebot {
		crawler.UserAgent = gbUserAgent
	}

	baseUrl := fmt.Sprintf("https://%s", *Domain) // "https://hindiclips.com"
	if *UseHttp {
		baseUrl = fmt.Sprintf("http://%s", *Domain)
	}

	Log.Info("crawler", "crwl", crawler, "base-url", baseUrl)

	site := &Site{BaseUrl: baseUrl, MaxDepth: *MaxDepth, KeywordPrefix: "/db/"}
	task, err := site.NewTask("/", 0)
	if err != nil {
		Log.Error("error", "err", err)
	}
	crawler.Q.Add(task)
	crawler.Run(*MaxWorkers)
}
