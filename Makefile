.PHONY: build build-linux build-windows build-macos build-macos-arm64 build-linux-arm64 clean all

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
