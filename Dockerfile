FROM golang:1.21-alpine as builder

ARG VERSION=1.0.0
ARG GITHASH="unknown"
ARG BUILDAT="unknown"


RUN apk update && apk add --no-cache git make gcc g++
WORKDIR $GOPATH/src/crwl/
COPY . .
RUN go get -d -v

RUN go build -o /go/bin/crwl \
    -ldflags "-X main.version=${VERSION} -X 'main.buildat=${BUILDAT}' -X 'main.githash=${GITHASH}'" \
    inmemory.go redis.go storage.go helper.go proxy.go queue.go visited.go crawler.go main.go

FROM alpine

RUN apk update 

COPY --from=builder /go/bin/crwl /bin/crwl

ENTRYPOINT ["/bin/crwl"]
