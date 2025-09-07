.PHONY: build test clean install release

BINARY_NAME=loopback-manager
VERSION := $(shell git describe --tags --always --dirty)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o $(BINARY_NAME) .

test:
	go test ./...

clean:
	go clean
	rm -f $(BINARY_NAME)

install: build
	sudo cp $(BINARY_NAME) /usr/local/bin/

build-all:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 .
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe .

dev: build
	./$(BINARY_NAME) list

release: clean build-all
