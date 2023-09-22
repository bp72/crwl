/*
TODO:
[ ]	Add Redis as External "Visited"
[ ] Set "Visited" TTL to 86400*7
[ ] Load proxy from external file
*/
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/cactus/go-statsd-client/v5/statsd"

	// "math/rand"

	"os"
	"time"
)

const userAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/113.0.0.0 Safari/537.36"
const gbUserAgent = "Mozilla/5.0 (Linux; Android 6.0.1; Nexus 5X Build/MMB29P) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.5735.179 Mobile Safari/537.36 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"

var (
	Log              *slog.Logger
	Domain           = flag.String("domain", "", "domain to scan")
	MaxDepth         = flag.Int("max-depth", 7, "set max depth for crawling")
	MaxWorkers       = flag.Int("max-workers", 20, "set max concurrent workers")
	UseGooglebot     = flag.Bool("use-google-bot", false, "Run as Googlebot mode")
	DoNotUseProxy    = flag.Bool("do-not-use-proxy", false, "Do not use proxy")
	DoNotStore       = flag.Bool("do-not-store", false, "Do not store content")
	UseHttp          = flag.Bool("use-http", false, "use http proto")
	Limit            = flag.Int("max-crawl", 100000, "set max amount of page to crawl")
	RedisAddr        = flag.String("redis-addr", "127.0.0.1:6379", "redis addr")
	RedisBase        = flag.Int("redis-base", 0, "redis base")
	RedisPass        = flag.String("redis-pass", "", "redis pass")
	UseInternalCache = flag.Bool("use-internal-cache", false, "Use internal cache insted of Redis")
	TaskUri          = flag.String("task-uri", "", "tasks with uri")
	UseRedis         = flag.Bool("use-redis", true, "use redis as queue backend")
	StatsdAddr       = flag.String("statsd-addr", "127.0.0.1:8125", "statsd collector addr")
	StorePath        = flag.String("store-path", "/tmp", "store pages to this dir")
	LastProxy        = 0
)

func init() {

	replace := func(groups []string, a slog.Attr) slog.Attr {
		// Remove time.
		if a.Key == slog.TimeKey && len(groups) == 0 {
			return slog.Attr{}
		}
		// Remove the directory from the source's filename.
		if a.Key == slog.SourceKey {
			source := a.Value.Any().(*slog.Source)
			source.File = filepath.Base(source.File)
		}
		return a
	}
	LogOpts := &slog.HandlerOptions{Level: slog.LevelInfo, AddSource: true, ReplaceAttr: replace}
	Log = slog.New(slog.NewTextHandler(os.Stdout, LogOpts))
}

func GetMetricsPrefix() string {
	return "crawler"
}

func main() {
	flag.Parse()
	Log.Info("start crawler")

	statsdCfg := &statsd.ClientConfig{
		Address: *StatsdAddr,
		Prefix:  GetMetricsPrefix(),
	}

	statsdClient, err := statsd.NewClientWithConfig(statsdCfg)
	if err != nil {
		Log.Error("create statsd client error", "msg", err)
	}
	defer statsdClient.Close()

	q := NewRedisQueue(
		RedisConnectionParams{
			Addr:     *RedisAddr,
			Base:     *RedisBase,
			Password: *RedisPass,
			Domain:   *Domain,
			Timeout:  2000 * time.Millisecond,
		},
	)

	v := NewRedisCache(
		RedisConnectionParams{
			Addr:     *RedisAddr,
			Base:     *RedisBase,
			Password: *RedisPass,
			Domain:   *Domain,
			Timeout:  2000 * time.Millisecond,
		},
	)

	Log.Info("use", "queue", "redis", "size", q.Size(context.Background()))
	// q := &InmemQueue{}
	StorePathForDomain := fmt.Sprintf("%s/%s", *StorePath, *Domain)
	baseUrl := fmt.Sprintf("https://%s", *Domain) // "https://hindiclips.com"
	if *UseHttp {
		baseUrl = fmt.Sprintf("http://%s", *Domain)
	}
	site := &Site{BaseUrl: baseUrl, MaxDepth: *MaxDepth, KeywordPrefix: "/db/"}

	if *TaskUri == "" {
		task, err := site.NewTask("/", 0)
		if err != nil {
			Log.Error("error", "err", err)
		}
		q.Put(context.Background(), task)
	} else {
		// q.LoadFromFile(site, *TaskUri)
	}

	if q.Size(context.Background()) == 0 {
		if task, err := site.NewTask("/", 0); err == nil {
			q.Put(context.Background(), task)
			time.Sleep(1 * time.Second)
		}
	}

	crawler := NewCrawler(q, v, StorePathForDomain)
	crawler.statsd = statsdClient

	if *UseGooglebot {
		crawler.UserAgent = gbUserAgent
	}

	Log.Info("crawler", "crwl", crawler, "base-url", baseUrl)

	crawler.Run(*MaxWorkers)
}
