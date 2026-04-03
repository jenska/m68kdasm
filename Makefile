.PHONY: all build test clean fmt ci

all: test build

build:
	go build -v ./...

test:
	go test -v ./...

fmt:
	go fmt ./...

ci:
	go mod verify
	go vet ./...
	go test -v ./...
	go build -v ./...

clean:
	go clean
