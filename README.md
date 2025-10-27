# ollama-model-downloader

A tiny Go CLI to fetch an Ollama model (manifest + blobs) directly from the Ollama registry and produce a `.zip` that mirrors the `~/.ollama/models` layout. Extract the zip into your Ollama `models` directory to make the model available without running `ollama pull`.

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

Opens a web browser to `http://localhost:<port>` with a Persian UI for downloading models.

Examples:

```
# Download by tag
./ollama-model-downloader embeddinggemma:latest

# Download by digest
./ollama-model-downloader embeddinggemma@sha256:abcd... -o embeddinggemma-digest.zip
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

- Talks to `registry.ollama.ai` using the Docker Registry (OCI) API.
- Handles bearer token authentication via the `WWW-Authenticate` challenge.
- Resolves multi-arch image indices and selects the manifest for your platform.
- Concurrent blob downloads with SHA-256 verification.
- Simple overall progress bar using manifest sizes.
- Downloads all referenced blobs (`config` + `layers`) and stores them as `blobs/sha256-<digest>`.
- Writes the selected manifest JSON under `manifests/<host>/<repo>/<tag or sha256-...>`.
- Zips the `models/` contents so it can be extracted straight into `~/.ollama/models`.

## Notes

- Default repository namespace is `library/` if none is provided (e.g. `llama3:latest`).
- If you specify a digest (`@sha256:...`), the manifest is stored under a digest filename (e.g. `sha256-...`).
- Public models should work without credentials; private registries are not supported.
- If the registry returns a multi-arch index, this tool chooses `linux/amd64` or `linux/arm64` based on your host (or `-platform`).
