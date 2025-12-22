.PHONY: all build test clean fmt

all: test build

build:
	go build -v ./...

test:
	go test -v ./...

fmt:
	go fmt ./...

clean:
	go clean