# Build local binary with version info
VERSION := $(shell git describe --tags --always)
COMMIT := $(shell git rev-parse HEAD)
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

build:
	go build -ldflags="-X 'github.com/yoohya/terracotta/cmd.version=$(VERSION)' -X 'github.com/yoohya/terracotta/cmd.commit=$(COMMIT)' -X 'github.com/yoohya/terracotta/cmd.date=$(DATE)'" -o bin/terracotta main.go

# Clean build artifacts
clean:
	rm -rf bin dist

# Create a release using Goreleaser
release:
	goreleaser release --clean
