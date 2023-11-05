DATE = $(shell date --iso=seconds)
GITHASH = $(shell git rev-parse --short HEAD)
VERSION = "1.0.0"
SOURCES = inmemory.go redis.go storage.go helper.go proxy.go queue.go visited.go crawler.go parser.go main.go

test:
	go test

test-with-race:
	go test -race


image: $(SOURCES)
	docker build . --file Dockerfile --tag crwl:$(GITHASH) \
		--build-arg GITHASH=$(GITHASH) \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILDAT=$(DATE)

	docker build . --file Dockerfile --tag crwl:latest \
		--build-arg GITHASH=$(GITHASH) \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILDAT=$(DATE)


build: test
	go build -o bin/crwl \
		-ldflags "-X main.version=$(VERSION) -X 'main.buildat=$(DATE)' -X 'main.githash=$(GITHASH)'" \
		${SOURCES}

build-with-race: test-with-race
	go build -race -o bin/crwl \
		-ldflags "-X main.version=$(VERSION) -X 'main.buildat=$(DATE)' -X 'main.githash=$(GITHASH)'" \
		${SOURCES}		

run: build
	bin/crwl -domain habr.com \
			 -use-redis \
			 -statsd-addr 192.168.1.140:8125 \
			 -store-path /tmp/crwl \
			 -redis-addr 192.168.1.140:6379 \
			 -redis-base 0 \
			 -redis-pass ddlmaster

run-with-race: build-with-race
	bin/crwl -domain habr.com \
			 -use-redis \
			 -statsd-addr 192.168.1.140:8125 \
			 -store-path /tmp/crwl \
			 -redis-addr 192.168.1.140:6379 \
			 -redis-base 0 \
			 -redis-pass ddlmaster

image-nas-repo: image
	docker tag crwl:$(GITHASH) 192.168.1.140:6088/crwl:$(GITHASH)
	docker push 192.168.1.140:6088/crwl:$(GITHASH)