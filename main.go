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
)

var (
	version          = "alpha"
	buildat          = "unknown"
	githash          = "unknown"
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
	UseInternalCache = flag.Bool("use-internal-cache", false, "Use internal cache instead of Redis")
	UseInternalQueue = flag.Bool("use-internal-queue", false, "Use internal queue instead of Redis")
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

func GetPrefix(useHttp bool) string {
	if useHttp {
		return "http://"
	}
	return "https://"
}

func GetBaseUrl(domain string, useHttp bool) string {
	return GetPrefix(useHttp) + *Domain
}

func main() {
	flag.Parse()
	Log.Info("start crawler", "version", version, "git", githash, "build-at", buildat)
	Log.Info("start-option", "use-internal-queue", *UseInternalQueue)
	Log.Info("start-option", "use-internal-cache", *UseInternalCache)

	statsdCfg := &statsd.ClientConfig{
		Address: *StatsdAddr,
		Prefix:  GetMetricsPrefix(),
	}

	statsdClient, err := statsd.NewClientWithConfig(statsdCfg)
	if err != nil {
		Log.Error("create statsd client error", "msg", err)
	}
	defer statsdClient.Close()

	Log.Info("use", "redis-addr", *RedisAddr, "base", *RedisBase, "password", *RedisPass)

	q := NewQueue(*UseInternalQueue)
	v := NewVisited(*UseInternalCache)

	Log.Info("use", "queue", "redis", "size", q.Size(context.Background()))

	StorePathForDomain := fmt.Sprintf("%s/%s", *StorePath, *Domain)

	baseUrl := GetBaseUrl(*Domain, *UseHttp)
	site := &Site{BaseUrl: baseUrl, MaxDepth: *MaxDepth, KeywordPrefix: "/db/"}
	task, err := site.NewTask("/", 0)
	if err != nil {
		Log.Error("error", "err", err)
	}
	q.Put(context.Background(), task)

	// if *TaskUri == "" {
	// 	task, err := site.NewTask("/", 0)
	// 	if err != nil {
	// 		Log.Error("error", "err", err)
	// 	}
	// 	q.Put(context.Background(), task)
	// } else {
	// 	// q.LoadFromFile(site, *TaskUri)
	// }

	if q.Size(context.Background()) == 0 {
		if task, err := site.NewTask("/", 0); err == nil {
			q.Put(context.Background(), task)
			// time.Sleep(1 * time.Second)
		}
	}

	crawler := NewCrawler(q, v, StorePathForDomain, statsdClient)
	crawler.statsd = statsdClient
	crawler.UserAgent = GetUserAgent(*UseGooglebot)

	Log.Info("crawler", "crwl", crawler, "base-url", baseUrl)

	crawler.Run(*MaxWorkers)
}
