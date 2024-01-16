<h1>Crawler</h1>

**crwl** is an open source web crawler in Golang which allows you to traverse entire site. Using it, you can scan, benchmark and validate your site, for example evaluate [connected component](https://en.wikipedia.org/wiki/Component_(graph_theory)) or [internal pagerank](https://en.wikipedia.org/wiki/PageRank)

### Motivation
I faced problem to crawl site as-is for various reason: create set site structure as graph, validate it, benchmark.

# Crawler arch
![alt text](https://github.com/bp72/crwl/blob/feature/update-readme-to-provide-more-context/crawler-arc.png?raw=true)


# Web Crawler
- Start from the root domain and crawl the web pages with a specified depth.
- Save the pages which contain a keyword(if provided) into database (TODO)
- Support logging and statsd metrics

```
bin/crwl -domain example.com -use-internal-cache -max-depth 3 -max-workers 5
```

# Options

#### Benchmark/Test mode
Sometime you just need to traverse your site without storing the content, just to check everything works fine or how far you can go. In this case you can use **-do-not-store** option, it disables content storing function :
```
bin/crwl -do-not-store
```

#### Setting up limits

Maximum crawls limitation
Option allows to limit number of crawls with exact number, by default it's 100k pages to crawl
```
bin/crwl -max-crawl 1234
```

Maximum depth allows to set limitation on how deep crawler can go, by default it's 7
```
bin/crwl -max-depth 1
```

Maximum number of worker sets the limit of concurrent cralwers to run, by default it's 20
```
bin/crwl -max-workers 2
```

#### Run without any external service dependancy
Crawler can be run standalone (without other services), however this configuration has memory limitation, since it's maintaince urls queue and visitied url in memory.
```
bin/crwl -use-internal-cache
```


### Roadmap
[x] Define crawler arch
[x] Implement initial crawler version

