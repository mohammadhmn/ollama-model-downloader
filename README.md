# ollama-model-downloader

A versatile Go-based download manager that supports both Ollama model downloads and generic HTTP/HTTPS file downloads with advanced queue management, pause/resume capability, and a modern Persian web interface.

## Features

### Ollama Model Downloader (Original)
- Fetch Ollama models directly from the Ollama registry
- Produce `.zip` files that mirror the `~/.ollama/models` layout
- Multi-arch support with automatic platform detection
- Concurrent blob downloads with SHA-256 verification

### Generic File Downloader (New)
- Download any HTTP/HTTPS file with pause/resume support
- Multi-download queue with concurrent downloads (configurable 1-32 simultaneous)
- Real-time speed tracking and ETA calculation
- Download history with persistent storage
- Statistics dashboard (total files, bandwidth, per-domain analytics)
- Modern Persian RTL web interface with tabs (Active/Queue/Library/History)

## Build

- Requires Go 1.21+

```
go build -o ollama-model-downloader
```

Produces a single static binary.

### Cross-platform builds

To build for different platforms, set GOOS and GOARCH:

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o ollama-model-downloader-linux-amd64

# Windows AMD64
GOOS=windows GOARCH=amd64 go build -o ollama-model-downloader-windows-amd64.exe

# macOS AMD64
GOOS=darwin GOARCH=amd64 go build -o ollama-model-downloader-macos-amd64

# macOS ARM64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o ollama-model-downloader-macos-arm64

# Linux ARM64
GOOS=linux GOARCH=arm64 go build -o ollama-model-downloader-linux-arm64
```

## Usage

### CLI Mode

```
./ollama-model-downloader [flags] <model[:tag] | model@sha256:digest>

Flags:
-o string              output zip path (default: <model>.zip)
-output-dir string     directory to save downloaded models (default "downloaded-models")
-registry string       registry base URL (default "https://registry.ollama.ai")
-platform string       target platform (default derives from host, e.g. linux/amd64)
  -concurrency int       concurrent blob downloads (default 4)
  -retries int           number of retry attempts (default 3)
  -port int              port to listen on for web UI (0 for random)
  -v                     verbose logging
  -keep-staging          keep staging directory after zip
```

### Web UI Mode

Run without arguments to start the web interface:

```
./ollama-model-downloader
```

Opens a web browser to `http://localhost:<port>` with a Persian UI for downloading files.

#### Ollama Model Examples:

```bash
# Download by tag
./ollama-model-downloader llama3:latest

# Download by digest
./ollama-model-downloader embeddinggemma@sha256:abcd... -o embeddinggemma-digest.zip
```

#### Generic File Download Examples:

```bash
# Download any HTTP/HTTPS file
./ollama-model-downloader https://example.com/file.zip

# Download with custom output name
./ollama-model-downloader https://example.com/file.tar.gz -o myfile.tar.gz

# Download with custom retry count
./ollama-model-downloader https://example.com/large-file.iso -retries 5
```

### Multi-Download Manager (Web UI)

The web interface provides advanced download management:

- **Active Tab**: View currently downloading files with real-time progress, speed, and ETA
- **Queue Tab**: Manage queued downloads (up to 32 simultaneous by default)
- **Library Tab**: Browse completed downloads with file management options
- **Actions**: Pause, resume, cancel individual downloads or bulk operations

#### API Endpoints

The application exposes REST API endpoints for programmatic access:

```bash
# Get all downloads
curl http://localhost:8080/api/downloads

# Add new download
curl -X POST http://localhost:8080/api/download/add \
  -d "url=https://example.com/file.zip&retries=3"

# Pause download
curl -X POST http://localhost:8080/api/download/{id}/pause

# Resume download
curl -X POST http://localhost:8080/api/download/{id}/resume

# Cancel download
curl -X POST http://localhost:8080/api/download/{id}/cancel

# Get statistics
curl http://localhost:8080/api/statistics

# Get history
curl http://localhost:8080/api/history?limit=50&offset=0
```

The resulting zip contains the following root structure (ready to extract into `~/.ollama/models`):

```
blobs/
  sha256-<...>
manifests/
  registry.ollama.ai/
    library/
      <name>/<tag or sha>
```

To install the model, extract the zip directly into your `~/.ollama/models` directory (or your Ollama data directory on your platform). If Ollama is running, you may need to restart it to pick up new files.

## How it works

### Ollama Mode
- Talks to `registry.ollama.ai` using the Docker Registry (OCI) API
- Handles bearer token authentication via the `WWW-Authenticate` challenge
- Resolves multi-arch image indices and selects the manifest for your platform
- Concurrent blob downloads with SHA-256 verification
- Downloads all referenced blobs (`config` + `layers`) and stores them as `blobs/sha256-<digest>`
- Writes the selected manifest JSON under `manifests/<host>/<repo>/<tag or sha256-...>`
- Zips the `models/` contents so it can be extracted straight into `~/.ollama/models`

### Generic Download Mode
- **URL Detection**: Automatically detects HTTP/HTTPS URLs vs Ollama model names
- **Download Manager**: Queue-based system with configurable concurrent downloads (default: 4)
- **Resume Support**: Uses HTTP Range headers and `.part` files for pause/resume capability
- **Speed Tracking**: Rolling window (10 samples) for smooth speed calculation and ETA estimation
- **History Manager**: Persistent JSON storage in `.history/history.json` with per-download metadata
- **Statistics**: Tracks total downloads, bandwidth usage, per-domain analytics, and file type distribution
- **Worker Pool**: Multi-goroutine architecture with context-based cancellation for graceful shutdown
- **Real-time Updates**: WebSocket-free polling (1 second interval) for UI updates

## Project Structure

```
ollama-model-downloader/
├── main.go                 # Entry point, web server, HTTP handlers
├── download.go             # Ollama-specific OCI registry logic
├── download_generic.go     # Generic HTTP/HTTPS download engine
├── download_manager.go     # Queue management, concurrent downloads
├── speed_tracker.go        # Speed calculation and ETA estimation
├── history.go              # Persistent download history
├── templates/index.html    # Persian RTL web interface
├── docs/                   # Phase 2-5 implementation documentation
└── downloaded-models/      # Default output directory
    ├── .history/          # Download history JSON files
    └── *.zip / files      # Downloaded content
```

## Architecture

### Backend (Go)
- **Zero external dependencies**: Uses only Go standard library
- **Thread-safe**: All managers use `sync.RWMutex` for concurrent access
- **Context-based**: Graceful cancellation via `context.Context`
- **Stateless API**: REST endpoints for UI integration
- **Session persistence**: Downloads survive app restarts via `.part` files and session metadata

### Frontend (Modern Web)
- **Framework**: Vanilla JavaScript + Tailwind CSS (CDN)
- **Typography**: Vazirmatn font (Google Fonts) for Persian support
- **Real-time**: AJAX polling (1s interval) for live updates
- **Responsive**: Tab-based interface (Active/Queue/Library)
- **Notifications**: Toast messages for user feedback

## Notes

- Default repository namespace is `library/` if none is provided (e.g. `llama3:latest`)
- If you specify a digest (`@sha256:...`), the manifest is stored under a digest filename (e.g. `sha256-...`)
- Public models should work without credentials; private registries are not supported
- If the registry returns a multi-arch index, this tool chooses `linux/amd64` or `linux/arm64` based on your host (or `-platform`)
- Generic downloads use direct file output (no ZIP compression)
- Download history is stored in JSON format at `downloaded-models/.history/history.json`
- Maximum concurrent downloads can be modified in `main.go` (currently 4)
