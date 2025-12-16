# Build local binary with version info
VERSION := $(shell git describe --tags --always)
COMMIT := $(shell git rev-parse HEAD)
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

build:
	go build -ldflags="-X 'github.com/yoohya/terracotta/cmd.version=$(VERSION)' -X 'github.com/yoohya/terracotta/cmd.commit=$(COMMIT)' -X 'github.com/yoohya/terracotta/cmd.date=$(DATE)'" -o bin/terracotta main.go

# Clean build artifacts
clean:
	rm -rf bin dist

# Clean build artifacts and test cache
clean-all: clean
	go clean -testcache
	rm -f coverage.out coverage.html

# Create a release using Goreleaser
release:
	goreleaser release --clean

# Run tests (disables cache with -count=1)
test:
	go test -count=1 -v ./...

# Run tests with coverage (disables cache with -count=1)
test-coverage:
	go test -count=1 -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run tests with race detector (disables cache with -count=1)
test-race:
	go test -count=1 -v -race ./...

# Run all quality checks (tests + race detector)
test-all: test-race test-coverage
