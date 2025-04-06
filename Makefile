# Build local binary
build:
	go build -o bin/terracotta main.go

# Clean build artifacts
clean:
	rm -rf bin dist

# Create a release using Goreleaser
release:
	goreleaser release --clean
