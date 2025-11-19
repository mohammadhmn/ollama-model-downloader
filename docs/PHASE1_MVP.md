# Phase 1: MVP (Minimum Viable Product)

**Duration**: 3-4 days  
**Tasks**: 16  
**Effort**: High  
**Status**: Not Started

## Goal

Convert Ollama-specific downloader into generic HTTP/HTTPS file downloader with pause/resume support.

---

## Backend Refactoring (5-6 hours)

### 1.1: Create `download_generic.go`

**Duration**: 2-3 hours  
**Status**: ⬜ Not Started

Create new file with generic HTTP download functions:

```go
// Main download function
func downloadFile(ctx context.Context, downloadURL, outputPath string, p *progress) error
  - Validate URL first
  - Check for existing .part file
  - Add Range header if resuming
  - Stream download with progress tracking
  - Rename .part to final filename on success

// Extract filename from URL
func extractFilenameFromURL(urlStr string) string
  - Parse URL path
  - Get filename from path
  - Handle empty paths (fallback to "download")
  - Handle query parameters

// Validate URL format
func validateURL(urlStr string) error
  - Parse URL
  - Verify HTTP/HTTPS scheme only
  - Check host is present
  - Return helpful error messages
```

**Files Modified**: Create new file  
**Dependencies**: `net/http`, `net/url`, `os`, `io`, `context`  
**Testing**: Basic validation with test URLs

---

### 1.2: Remove Ollama-Specific Code

**Duration**: 1-2 hours  
**Status**: ⬜ Not Started

Remove from `download.go`:

**Functions to delete:**

- [ ] `parseModel()` - Ollama model parsing
- [ ] `getRegistryToken()` - OCI bearer token
- [ ] `parseBearerChallenge()` - Auth challenge parsing
- [ ] `getManifestOrIndex()` - OCI manifest fetching
- [ ] `downloadBlob()` - OCI blob download
- [ ] `dedupeBlobs()` - Blob deduplication
- [ ] `ensureStagingRoot()` - Staging directory
- [ ] `ollamaModelsDir()` - Ollama models path
- [ ] `unzipToDir()` - Unzip functionality

**Constants to delete:**

- [ ] `mtOCIIndex` - OCI index media type
- [ ] `mtDockerIndex` - Docker index media type
- [ ] `mtOCIManifest` - OCI manifest media type
- [ ] `mtDockerManifest` - Docker manifest media type
- [ ] `defaultRegistry` - Ollama registry URL
- [ ] `defaultPlatform` - Default platform

**Types to delete:**

- [ ] `imageIndex` struct
- [ ] `imageManifest` struct
- [ ] `bearerAuth` struct
- [ ] `modelRef` struct

**Files Modified**: `download.go`  
**Verification**: Search for "ollama", "registry", "OCI" - should find only comments

---

### 1.3: Update `sessionMeta` Struct

**Duration**: 30 minutes  
**Status**: ⬜ Not Started

**In `main.go`:**

```go
type sessionMeta struct {
    // Add these fields
    URL          string    `json:"url"`
    Filename     string    `json:"filename"`
    ExpectedSize int64     `json:"expectedSize"`

    // Keep these
    SessionID    string    `json:"sessionId"`
    OutPath      string    `json:"outPath"`
    StagingRoot  string    `json:"stagingRoot"`
    Retries      int       `json:"retries"`

    // Keep these for tracking
    StartedAt    time.Time `json:"startedAt"`
    LastUpdated  time.Time `json:"lastUpdated"`
    State        string    `json:"state"`        // "active", "paused", "completed", "error"
    Message      string    `json:"message"`

    // Remove these (Ollama-specific)
    // Registry    string
    // Platform    string
    // Concurrency int
    // ConfigPath  string
}
```

**Files Modified**: `main.go`  
**Impact**: All code using sessionMeta needs review

---

### 1.4: Implement Resume Support

**Duration**: 1-2 hours  
**Status**: ⬜ Not Started

**In `download_generic.go`:**

```go
// Check for partial file
info, _ := os.Stat(outputPath + ".part")
if info != nil && info.Size() > 0 {
    // File exists and has content - resume
    offset := info.Size()
    req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
}

// Handle response codes
switch resp.StatusCode {
case 200:  // Full download
case 206:  // Partial content (resume)
case 416:  // Range not satisfiable
}

// Append to existing .part file if resuming
var flags int
if info != nil {
    flags = os.O_APPEND  // Append mode
} else {
    flags = os.O_CREATE  // Create new
}
f, err := os.OpenFile(outputPath+".part", flags|os.O_WRONLY, 0644)
```

**Files Modified**: `download_generic.go`  
**Testing**: Interrupt and resume download

---

### 1.5: Add URL Validation

**Duration**: 30 minutes  
**Status**: ⬜ Not Started

**Validation checklist:**

- [ ] HTTP/HTTPS only (reject ftp://, file://, etc)
- [ ] Valid URL format
- [ ] Host is present and valid
- [ ] Extract filename safely
- [ ] Handle query parameters
- [ ] Handle fragments
- [ ] Sanitize filename (remove invalid chars)

**Files Modified**: `download_generic.go`  
**Function**: `validateURL()`, `extractFilenameFromURL()`

---

## CLI Updates (1-2 hours)

### 2.1: Update Flag Definitions

**Duration**: 30 minutes  
**Status**: ⬜ Not Started

**In `main.go` main() function:**

```go
// Remove these flags
// flag.StringVar(&opt.registry, "registry", defaultRegistry, "...")
// flag.StringVar(&opt.platform, "platform", defaultPlatform, "...")
// flag.IntVar(&opt.concurrency, "concurrency", 4, "...")

// Keep these
flag.StringVar(&opt.outZip, "o", "", "output filename")
flag.StringVar(&opt.outputDir, "output-dir", "downloads", "...")
flag.IntVar(&opt.retries, "retries", 3, "...")
flag.IntVar(&opt.port, "port", 0, "...")
flag.BoolVar(&opt.verbose, "v", false, "...")

// New (optional)
flag.StringVar(&url, "url", "", "File URL to download (web mode)")
```

**Files Modified**: `main.go`  
**Testing**: `./downloader -h` should show updated flags

---

### 2.2: Accept URL as Positional Argument

**Duration**: 30 minutes  
**Status**: ⬜ Not Started

**CLI behavior:**

```bash
# Web server mode (no arguments)
./downloader -port 8080
→ Opens web interface

# CLI download mode
./downloader https://example.com/file.zip
→ Downloads directly

# With options
./downloader https://example.com/file.zip -o myfile.zip
→ Downloads to specific filename

# With retries
./downloader https://example.com/file.zip -retries 5
→ Retries up to 5 times on failure
```

**Implementation:**

```go
args := flag.Args()
if len(args) > 0 {
    url := args[0]  // First positional arg
    // CLI download mode
} else {
    // Web server mode
}
```

**Files Modified**: `main.go`  
**Testing**: Try all variations above

---

## Web UI Updates (3-4 hours)

### 3.1: Update Form Inputs

**Duration**: 1 hour  
**Status**: ⬜ Not Started

**File**: `templates/index.html`

**Replace this:**

```html
<input name="model" placeholder="نام مدل (مثال: llama3.2)" />
```

**With this:**

```html
<input name="url" placeholder="آدرس دانلود (مثال: https://...)" required />
<input name="filename" placeholder="نام فایل (اختیاری)" />
```

**Keep:**

```html
<input name="retries" type="number" min="1" max="10" value="3" />
```

**Files Modified**: `templates/index.html`  
**Testing**: Form renders and accepts URL

---

### 3.2: Remove Unnecessary Form Fields

**Duration**: 30 minutes  
**Status**: ⬜ Not Started

**Remove from form:**

- [ ] Platform selector
- [ ] Registry field
- [ ] Concurrency input
- [ ] Any Ollama-specific fields

**Keep only:**

- [ ] URL input (required)
- [ ] Filename input (optional)
- [ ] Retries selector
- [ ] Download button

**Files Modified**: `templates/index.html`  
**Testing**: Form is cleaner, fewer inputs

---

### 3.3: Update Persian UI Labels

**Duration**: 1 hour  
**Status**: ⬜ Not Started

**Update labels:**

| Old                          | New                   |
| ---------------------------- | --------------------- |
| مدیریت دانلود مدل‌های Ollama | مدیریت دانلود فایل‌ها |
| دانلود مدل جدید              | دانلود فایل جدید      |
| نام مدل                      | آدرس دانلود (URL)     |
| نام کتابخانه                 | نام فایل (اختیاری)    |
| بستر                         | (remove)              |
| رجیستری                      | (remove)              |
| تعداد دانلود موازی           | (remove for MVP)      |
| فایل‌های دانلود شده          | دانلود‌های انجام شده  |
| استخراج                      | (remove for MVP)      |

**Files Modified**: `templates/index.html`  
**Verification**: All forms display in Persian correctly

---

### 3.4: Update Download Card Display

**Duration**: 1 hour  
**Status**: ⬜ Not Started

**Change from:**

```html
<h3>{{.RunningSession.Model}}</h3>
<p>نسخه: {{.RunningSession.Platform}}</p>
<p>رجیستری: {{.RunningSession.Registry}}</p>
```

**Change to:**

```html
<h3>{{.RunningSession.Filename}}</h3>
<p>{{.RunningSession.URL}}</p>
<p>اندازه: {{humanSize .RunningSession.ExpectedSize}}</p>
<p>دانلود شده: {{humanSize .RunningSession.Downloaded}}</p>
```

**Files Modified**: `templates/index.html`  
**Testing**: Download card shows correct info

---

### 3.5: Update `/download` Handler

**Duration**: 1 hour  
**Status**: ⬜ Not Started

**In `main.go`:**

```go
http.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
    // Parse form
    downloadURL := r.FormValue("url")
    filename := r.FormValue("filename")

    // Validate URL
    if err := validateURL(downloadURL); err != nil {
        http.Error(w, fmt.Sprintf("خطا: %v", err), http.StatusBadRequest)
        return
    }

    // Extract filename if not provided
    if filename == "" {
        filename = extractFilenameFromURL(downloadURL)
    }

    // Validate filename
    if filename == "" || filename == "download" {
        http.Error(w, "نمی‌توان نام فایل را تعیین کنید", http.StatusBadRequest)
        return
    }

    // Create options
    opt := options{
        model:      filename,
        url:        downloadURL,
        outZip:     filepath.Join(opt.outputDir, filename),
        sessionID:  sanitizeFilename(filename),
        stagingDir: filepath.Join(opt.outputDir, sanitizeFilename(filename)+".staging"),
        retries:    parseRetries(r.FormValue("retries")),
    }

    // Start download
    beginDownloadSession(opt, "در حال دانلود...")

    // Redirect to home
    http.Redirect(w, r, "/", http.StatusFound)
})
```

**Files Modified**: `main.go`  
**Testing**: Form submission works, download starts

---

## Testing (3-4 hours)

### 4.1: Test Small File Download

**Duration**: 1 hour  
**Status**: ⬜ Not Started

**Test case:**

1. Start web server: `go run . -port 8080`
2. Open http://localhost:8080
3. Enter URL to 50-100MB file
4. Click download
5. Verify progress updates every 1 second
6. Click pause after 30%
7. Verify downloads stops
8. Click resume
9. Verify continues from where paused
10. Let complete
11. Verify file exists and correct size
12. Delete file

**Test URLs:**

- https://releases.ubuntu.com/22.04/ubuntu-22.04.1-desktop-amd64.iso (small, ~4GB - too big)
- Use a smaller file like 100MB test file

**Files Modified**: None  
**Result**: ✅ Pass or document issues

---

### 4.2: Test Large File Download with Resume

**Duration**: 1 hour  
**Status**: ⬜ Not Started

**Test case:**

1. Start download of 1GB+ file
2. After 30% downloaded, kill the app: `Ctrl+C`
3. Check that `.part` file exists in staging
4. Restart app: `go run .`
5. Go to home page
6. Verify previous download session shows
7. Click resume
8. Verify continues from correct offset (check `.part` file size)
9. Let complete
10. Verify `.part` renamed to final filename
11. Verify file correct size

**Expected behavior:**

- `.part` file preserves downloaded bytes
- Resume requests correct byte range
- No data redownloaded
- Final file correct size

**Files Modified**: None  
**Result**: ✅ Pass or document issues

---

### 4.3: Test Error Handling

**Duration**: 1 hour  
**Status**: ⬜ Not Started

**Test cases:**

1. **Invalid URL format:**

   - Input: "not a url"
   - Expected: Error message in UI
   - [ ] Error displays

2. **404 Not Found:**

   - Input: "https://httpbin.org/status/404"
   - Expected: Error in UI, download marked failed
   - [ ] Error handled

3. **Network timeout:**

   - Input: "https://httpbin.org/delay/30" (with 5s timeout)
   - Expected: Timeout error
   - [ ] Error shown gracefully

4. **Missing Content-Length:**

   - Input: URL without Content-Length header
   - Expected: Download with unknown size
   - [ ] Progress shows "? MB"

5. **Invalid URL scheme:**
   - Input: "ftp://example.com/file"
   - Expected: Rejected error
   - [ ] Error message clear

**Files Modified**: None  
**Result**: ✅ All errors handled

---

### 4.4: Test CLI Mode

**Duration**: 30 minutes  
**Status**: ⬜ Not Started

**Test cases:**

```bash
# Build binary first
go build -o downloader

# Test 1: Direct download
./downloader https://example.com/file.zip
# Expected: Downloads and completes

# Test 2: With output filename
./downloader https://example.com/file.zip -o myfile.zip
# Expected: Saves as myfile.zip

# Test 3: With retries
./downloader https://example.com/file.zip -retries 5
# Expected: Retries up to 5 times on failure

# Test 4: Web server mode
./downloader -port 8888
# Expected: Opens http://localhost:8888

# Test 5: Multiple file types
./downloader https://example.com/archive.tar.gz
./downloader https://example.com/image.iso
./downloader https://example.com/document.pdf
# Expected: All download correctly
```

**Files Modified**: None  
**Result**: ✅ All CLI modes work

---

### 4.5: Test Session Persistence

**Duration**: 1 hour  
**Status**: ⬜ Not Started

**Test case:**

1. Start download: `go run .`
2. In browser, start downloading large file (enter URL)
3. Wait for download to reach 30%
4. Kill app: `Ctrl+C` (don't gracefully shutdown)
5. Check session file exists: `ls -la downloaded-models/.staging/*/session.json`
6. Restart app: `go run .`
7. Go to home page
8. Verify download session appears with same progress
9. Verify can resume
10. Let complete

**Expected behavior:**

- Session saved every ~5 seconds
- Progress preserved
- Can resume after restart
- Session cleaned up after completion

**Files Modified**: None  
**Result**: ✅ Sessions persist correctly

---

## Finalization

### 5.1: Code Review & Cleanup

**Duration**: 1 hour  
**Status**: ⬜ Not Started

**Checklist:**

- [ ] No Ollama code remains (search for "ollama", "registry", "OCI")
- [ ] No compile errors
- [ ] No warnings
- [ ] Consistent code style (gofmt)
- [ ] Comments are clear
- [ ] Error messages are helpful

**Commands:**

```bash
# Check for Ollama references
grep -r "ollama\|registry\|OCI" --include="*.go"

# Format code
go fmt ./...

# Build
go build -o downloader

# Run tests if any
go test ./...
```

**Files Modified**: All modified files  
**Result**: ✅ Code clean and ready

---

### 5.2: Commit Changes

**Duration**: 30 minutes  
**Status**: ⬜ Not Started

**Commands:**

```bash
git add -A
git commit -m "feat: implement MVP generic file downloader

- Remove all Ollama-specific logic (OCI, registry, platform, bearer auth)
- Implement generic HTTP/HTTPS download with streaming
- Add resume support via Range headers and .part files
- Update sessionMeta for generic downloads (add URL, Filename, ExpectedSize)
- Update CLI flags and accept URL as positional argument
- Update web UI form to accept download URLs instead of model names
- Update Persian labels from Ollama-specific to generic file downloader terms
- Implement URL validation and filename extraction
- Update download handler for generic URL processing
- Test with various file sizes and types (50MB, 1GB+)
- Test pause/resume functionality
- Test error handling (invalid URL, 404, timeout, etc)
- Test session persistence across app restarts
- Cleanup: remove all Ollama references from comments and code"
```

**Files Modified**: Multiple  
**Result**: ✅ Changes committed

---

## MVP Summary

**Completed Tasks**: 16  
**Total Duration**: 10-15 hours  
**Next Phase**: Phase 2 - Manager Features

**What Works Now:**
✅ Download any HTTP/HTTPS file  
✅ Pause and resume downloads  
✅ Accurate progress tracking  
✅ Session persistence  
✅ CLI and web UI modes  
✅ Error handling with user messages

**What's Next:**
→ Multiple concurrent downloads  
→ Queue management  
→ Speed tracking and ETA  
→ Download history  
→ Advanced filtering and search

---

## Links

- See `PHASE2_MANAGER.md` for next phase
- See `CHECKLIST.md` for quick reference
- See `CONVERSION_PLAN.md` for architectural details
