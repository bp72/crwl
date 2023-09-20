DATE = $(shell date --iso=seconds)
GITHASH = $(shell git rev-parse --short HEAD)
VERSION = "1.0.0"
SOURCES = inmemory.go redis.go storage.go helper.go proxy.go queue.go visited.go crawler.go main.go

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

run: build
	# bin/crwl -domain www.xvideos.com -use-redis
	bin/crwl -domain hindiclips.com -use-redis