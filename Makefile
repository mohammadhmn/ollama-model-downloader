.PHONY: build build-linux build-windows build-macos build-macos-arm64 build-linux-arm64 clean all test lint fmt run dev

# Build for current platform
build:
	go build -o ollama-model-downloader .

# Cross-platform builds
build-linux:
	GOOS=linux GOARCH=amd64 go build -o ollama-model-downloader-linux-amd64 .

build-windows:
	GOOS=windows GOARCH=amd64 go build -o ollama-model-downloader-windows-amd64.exe .

build-macos:
	GOOS=darwin GOARCH=amd64 go build -o ollama-model-downloader-macos-amd64 .

build-macos-arm64:
	GOOS=darwin GOARCH=arm64 go build -o ollama-model-downloader-macos-arm64 .

build-linux-arm64:
	GOOS=linux GOARCH=arm64 go build -o ollama-model-downloader-linux-arm64 .

# Clean build artifacts
clean:
	rm -f ollama-model-downloader*

# Build for all platforms
all: build-linux build-windows build-macos build-macos-arm64 build-linux-arm64

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Run linter and formatter
check: fmt lint

# Run the application
run: build
	./ollama-model-downloader

# Development mode with hot reload (requires air)
dev:
	air -c .air.toml

# Install dependencies
deps:
	go mod download
	go mod tidy

# Build release versions with optimization
release:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o ollama-model-downloader .

# Build all release versions
release-all: clean
	$(foreach target,$(TARGETS), \
		echo "Building for $(target)"; \
		GOOS=$(word 1,$(subst :, ,$(target))) GOARCH=$(word 2,$(subst :, ,$(target))) \
		CGO_ENABLED=0 go build -ldflags="-s -w" \
		-o ollama-model-downloader-$(word 1,$(subst :, ,$(target)))-$(word 2,$(subst :, ,$(target))) .; \
	)

# Define targets for cross-compilation
TARGETS = linux:amd64 linux:arm64 windows:amd64 darwin:amd64 darwin:arm64

# Docker build
docker-build:
	docker build -t ollama-model-downloader .

# Docker run
docker-run:
	docker run -p 8080:8080 ollama-model-downloader
