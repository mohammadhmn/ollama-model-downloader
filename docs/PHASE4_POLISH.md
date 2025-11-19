# Phase 4: Polish & Documentation

**Duration**: 2-3 days  
**Tasks**: 7  
**Effort**: Low-Medium  
**Prerequisite**: Phase 2 (or Phase 3) Complete  
**Status**: Not Started

## Goal

Final cleanup, comprehensive documentation, error handling refinement, and performance optimization before release.

---

## Task 1: Update Documentation

**Duration**: 3-4 hours  
**Status**: ⬜ Not Started

### 1.1: Update README.md

**Current state**: Ollama-specific  
**Target state**: Generic file downloader

**Changes needed:**

````markdown
# File Download Manager

A powerful, open-source download manager with web UI and CLI support.

## Features

- ✅ Download any HTTP/HTTPS file
- ✅ Resume interrupted downloads
- ✅ Multiple concurrent downloads
- ✅ Real-time speed and ETA tracking
- ✅ Download queue management with priorities
- ✅ Comprehensive download history
- ✅ Advanced search and filtering
- ✅ Bandwidth limiting
- ✅ Custom HTTP headers
- ✅ Web UI and CLI modes
- ✅ Dark/Light theme
- ✅ Desktop notifications

## Quick Start

### Web UI Mode

```bash
go build
./downloader -port 8080
```
````

Opens http://localhost:8080

### CLI Mode

```bash
./downloader https://example.com/file.zip
./downloader https://example.com/file.zip -o myname.zip -retries 5
```

## Building from Source

```bash
# Requires Go 1.21+
git clone ...
cd download-manager
go build -o downloader
```

## Usage

### Web Interface

1. Enter download URL
2. Click "دانلود جدید"
3. Monitor progress in Active tab
4. View history and statistics

### Command Line

```bash
./downloader [flags] <url>

Flags:
  -o string           output filename
  -output-dir string  save location (default "downloads")
  -port int          web server port
  -retries int       retry attempts (default 3)
  -v                 verbose logging
```

## Configuration

All settings available in UI:

- Download folder location
- Max concurrent downloads (1-32)
- Bandwidth limiting
- Auto-start downloads
- Theme preference
- Notification settings

See Settings panel in web UI.

## Platform Support

Built with Go - works on:

- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

## Requirements

- Go 1.21 or higher
- Modern web browser for UI

## License

MIT License - see LICENSE file

## Contributing

See CONTRIBUTING.md for development setup and guidelines.

````

**Sections to remove:**
- Ollama-specific features
- Docker/OCI registry information
- Platform selection
- Model library references

**Sections to add:**
- Features list (generic)
- Quick start (web + CLI)
- Configuration section
- Multiple platform support

**Files Modified**: `README.md`
**Testing**: Read through, verify all examples work

---

### 1.2: Update CONTRIBUTING.md

**Update project structure section:**

```markdown
## Project Structure

````

download-manager/
├── config/ # (Optional) Configuration files
├── docs/ # Documentation
│ ├── TASKS.md # Task list for development
│ ├── PHASE1_MVP.md # MVP implementation guide
│ ├── PHASE2_MANAGER.md # Manager features
│ ├── PHASE3_ADVANCED.md # Optional features
│ ├── CONVERSION_PLAN.md # Architecture overview
│ └── IMPLEMENTATION_STEPS.md # Step-by-step guide
├── internal/ # Internal packages
│ ├── download/ # Download logic
│ └── history/ # History management
├── templates/ # HTML templates
│ └── index.html # Web UI
├── download_generic.go # Generic HTTP download
├── download_manager.go # Queue management
├── speed_tracker.go # Speed calculation
├── history.go # History persistence
├── progress.go # Progress tracking
├── main.go # CLI and web server
├── go.mod # Module definition
├── Makefile # Build automation
└── README.md # Project documentation

````

## Development Workflow

1. Create feature branch
2. Implement feature with tests
3. Update documentation
4. Submit pull request

## Code Style

- Follow Go conventions (gofmt, golint)
- Write meaningful comments
- Keep functions focused
- Use error handling properly

## Testing

```bash
make test              # Run tests
make test-coverage     # With coverage
````

## Key Files to Modify

When adding features:

- Backend: create new file or modify existing
- Frontend: modify templates/index.html
- API: add handlers in main.go
- Tests: create corresponding test files

````

**Files Modified**: `CONTRIBUTING.md`

---

### 1.3: Create FEATURES.md

Create comprehensive feature documentation:

```markdown
# Features Overview

## Core Features (MVP)

### Single File Download
- Download any HTTP/HTTPS URL
- Streaming download with progress tracking
- Automatic filename extraction
- Customizable output filename

### Pause & Resume
- Pause downloads mid-transfer
- Resume from exact byte position
- Uses HTTP Range headers
- Preserves .part file on pause

### Session Persistence
- Downloads survive app restart
- Progress preserved across restarts
- Session recovery on startup

### Error Handling
- Graceful handling of network errors
- Automatic retries (configurable)
- Clear error messages to user
- Network timeout handling

## Manager Features

### Queue Management
- Add multiple downloads
- Queue with FIFO processing
- Priority system (1-10)
- Pause/Resume individual downloads
- Bulk pause/resume all

### Concurrent Downloads
- Configurable max concurrent (1-32)
- Worker pool architecture
- Automatic queue processing
- No download starvation

### Speed Tracking
- Real-time download speed (MB/s)
- Moving average for smooth display
- Byte counter with total progress
- Visual progress bar

### ETA Calculation
- Time remaining estimation
- Updates every second
- Accuracy within 20%
- Formatted as "5m 30s"

### Download History
- Persistent history (JSON format)
- Auto-save after each completion
- Search by filename/URL
- Filter by status, date, type
- Delete individual entries

### Statistics
- Total files and bytes downloaded
- Today's vs all-time stats
- Top domains by bandwidth
- File type distribution
- Average download speed

## Advanced Features (Optional)

### Bandwidth Limiting
- Global speed cap (KB/s)
- Per-download limits
- Throttled write with sleep

### Custom Headers
- Support Authorization headers
- Custom User-Agent
- Cookie headers
- Preset templates

### Batch Import
- Paste multiple URLs at once
- One URL per line
- Validation before adding
- Preview before import

### Keyboard Shortcuts
- Ctrl+N: New download
- Ctrl+A: Select all
- Ctrl+D: Delete selected
- Ctrl+P: Pause/Resume
- Ctrl+Shift+P: Pause all
- Ctrl+F: Search
- Del: Delete selected

### Dark/Light Theme
- Auto-detection from system
- Manual toggle
- Persistent preference
- CSS variable based

### Settings Panel
- Download folder selection
- Performance tuning
- Behavior options
- Appearance preferences
- Advanced options

## Architecture

See CONVERSION_PLAN.md for detailed architecture information.
````

**Files Created**: `FEATURES.md`

---

## Task 2: Improve Error Handling

**Duration**: 2-3 hours  
**Status**: ⬜ Not Started

### 2.1: Review All Error Messages

**Current state**: May be technical  
**Target state**: User-friendly with suggestions

**Common errors to improve:**

```go
// Bad: Technical error
"context deadline exceeded"

// Good: User-friendly with suggestion
"Download took too long. Try again or increase timeout in settings."

---

// Bad: Unclear
"invalid URL: parse error"

// Good: Helpful
"Invalid download URL. Please check:
  - URL starts with http:// or https://
  - No special characters without encoding
  - URL is not behind a redirect (try copying from browser)"

---

// Bad: No suggestion
"file not found"

// Good: Suggests action
"404 Not Found - The file doesn't exist at that URL.
  - Check if URL is correct
  - Try a different source
  - File may have been removed or moved"

---

// Bad: Generic
"network error"

// Good: Specific with action
"Network error: No internet connection
  - Check your network
  - Try again in a moment
  - Check firewall/proxy settings"
```

**Implement wrapper function:**

```go
func userFriendlyError(err error) string {
    switch {
    case errors.Is(err, context.DeadlineExceeded):
        return "Download timeout. Try increasing timeout in settings."
    case strings.Contains(err.Error(), "connection refused"):
        return "Connection refused. Check if server is online."
    case strings.Contains(err.Error(), "no such host"):
        return "Invalid domain. Check the URL spelling."
    case strings.Contains(err.Error(), "404"):
        return "File not found (404). Check URL is correct."
    default:
        return fmt.Sprintf("Error: %v", err)
    }
}
```

**Update all error displays to use this.**

**Files Modified**: `main.go`, `download_generic.go`, etc  
**Testing**: Trigger various errors, verify messages are helpful

---

### 2.2: Add Retry Logic

**Current state**: May be basic  
**Target state**: Smart retries with backoff

```go
func downloadWithRetries(ctx context.Context, url, path string, maxRetries int) error {
    for attempt := 1; attempt <= maxRetries; attempt++ {
        err := downloadFile(ctx, url, path, nil)

        if err == nil {
            return nil  // Success
        }

        // Check if error is retryable
        if !isRetryable(err) {
            return err  // Don't retry
        }

        if attempt < maxRetries {
            // Exponential backoff: 1s, 2s, 4s, 8s...
            backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
            time.Sleep(backoff)
        }
    }

    return fmt.Errorf("failed after %d attempts", maxRetries)
}

func isRetryable(err error) bool {
    // Retryable errors
    if err == nil {
        return false
    }

    // Timeout - retryable
    if errors.Is(err, context.DeadlineExceeded) {
        return true
    }

    // Network error - retryable
    var netErr net.Error
    if errors.As(err, &netErr) && netErr.Temporary() {
        return true
    }

    // 5xx server errors - retryable
    if strings.Contains(err.Error(), "50") {
        return true
    }

    // 429 too many requests - retryable
    if strings.Contains(err.Error(), "429") {
        return true
    }

    return false  // Not retryable
}
```

**Files Modified**: `download_generic.go`  
**Testing**: Simulate network errors, verify retries work

---

## Task 3: Performance Optimization

**Duration**: 2-3 hours  
**Status**: ⬜ Not Started

### 3.1: Profile and Optimize

**Check for bottlenecks:**

```bash
# CPU profile
go tool pprof cpu.prof

# Memory profile
go tool pprof mem.prof

# Run with profiling
go run -cpuprofile=cpu.prof -memprofile=mem.prof .

# Build with optimization
go build -ldflags "-s -w"
```

**Common optimizations:**

```go
// Bad: Creates new slice for each iteration
var items []Item
for _, id := range ids {
    items = append(items, getItem(id))
}

// Good: Pre-allocate if size known
items := make([]Item, 0, len(ids))
for _, id := range ids {
    items = append(items, getItem(id))
}

---

// Bad: Locks entire list for iteration
func (dm *DownloadManager) GetAll() []*Download {
    dm.mu.Lock()
    defer dm.mu.Unlock()

    for _, d := range dm.downloads {
        // ... process
    }
}

// Good: Only lock for copy
func (dm *DownloadManager) GetAll() []*Download {
    dm.mu.RLock()
    downloads := make([]*Download, 0, len(dm.downloads))
    for _, d := range dm.downloads {
        downloads = append(downloads, d)
    }
    dm.mu.RUnlock()

    // Processing outside lock
    return downloads
}

---

// Bad: JSON marshalling in hot path
for _, d := range downloads {
    data, _ := json.Marshal(d)
    w.Write(data)
}

// Good: Marshal once
data, _ := json.Marshal(downloads)
w.Write(data)
```

**UI Performance:**

```js
// Bad: Updates DOM in loop
for (const dl of downloads) {
    container.innerHTML += renderCard(dl);
}

// Good: Batch DOM updates
let html = '';
for (const dl of downloads) {
    html += renderCard(dl);
}
container.innerHTML = html;

---

// Bad: Updates every change
addEventListener('input', (e) => {
    applyFilter(e.target.value);  // Updates immediately
});

// Good: Debounced updates
addEventListener('input', debounce((e) => {
    applyFilter(e.target.value);  // Updates after 300ms
}, 300));

function debounce(func, wait) {
    let timeout;
    return function(...args) {
        clearTimeout(timeout);
        timeout = setTimeout(() => func.apply(this, args), wait);
    };
}
```

**Files Modified**: Various  
**Testing**: Monitor app under load with many downloads

---

### 3.2: Optimize History

**For large history files:**

```go
// Bad: Load all into memory
func (hm *HistoryManager) Load() error {
    data, _ := os.ReadFile(hm.file)
    return json.Unmarshal(data, &hm.entries)  // All at once
}

// Good: Lazy load or paginate
func (hm *HistoryManager) GetEntries(limit, offset int) []*HistoryEntry {
    // Load only requested range
    // Or use database for large files
}

// Consider SQLite for history if:
// - File becomes > 1MB
// - Searching performance matters
// - Query flexibility needed
```

**Files Modified**: `history.go`  
**Testing**: Monitor with 10,000+ history entries

---

## Task 4: Handle Edge Cases

**Duration**: 2-3 hours  
**Status**: ⬜ Not Started

### 4.1: File Conflict Resolution

**Current**: May overwrite  
**Target**: Smart conflict handling

```go
func handleFileConflict(desiredPath string) (string, error) {
    info, err := os.Stat(desiredPath)
    if err == nil {
        // File exists - need to decide
        return resolveConflict(desiredPath)
    }
    if os.IsNotExist(err) {
        return desiredPath, nil  // OK, doesn't exist
    }
    return "", err
}

func resolveConflict(path string) (string, error) {
    // Strategy 1: Auto-rename
    ext := filepath.Ext(path)
    base := strings.TrimSuffix(path, ext)

    for i := 1; i <= 999; i++ {
        newPath := fmt.Sprintf("%s (%d)%s", base, i, ext)
        if _, err := os.Stat(newPath); os.IsNotExist(err) {
            return newPath, nil
        }
    }

    return "", fmt.Errorf("could not find available filename")
}
```

**UI Prompt:**

```html
<!-- When conflict detected -->
<div id="conflict-dialog" class="modal">
  <p>فایل "file.zip" قبلاً وجود دارد</p>

  <button onclick="resolveConflict('overwrite')">جایگزین کن</button>
  <button onclick="resolveConflict('rename')">تغییر نام</button>
  <button onclick="resolveConflict('cancel')">لغو</button>
</div>
```

**Files Modified**: `download_manager.go`, `templates/index.html`  
**Testing**: Create file, try to download same name

---

### 4.2: Low Disk Space Handling

```go
func checkDiskSpace(path string, requiredBytes int64) error {
    stat := syscall.Statfs_t{}
    syscall.Statfs(path, &stat)

    availableBytes := stat.Bavail * uint64(stat.Bsize)

    if int64(availableBytes) < requiredBytes {
        return fmt.Errorf("low disk space: need %d MB, have %d MB",
            requiredBytes/1e6, availableBytes/1e6)
    }

    return nil
}
```

**UI Warning:**

```html
<div id="disk-warning" class="warning hidden">
  ⚠️ فضای دیسک کافی نیست. حداقل {{required}} MB لازم است.
  <button onclick="selectFolder()">انتخاب پوشه دیگر</button>
</div>
```

**Files Modified**: `download_generic.go`, `templates/index.html`

---

### 4.3: Special Characters in Filenames

```go
func sanitizeFilename(name string) string {
    // Remove/replace invalid characters
    invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}

    result := name
    for _, char := range invalid {
        result = strings.ReplaceAll(result, char, "_")
    }

    // Limit length
    if len(result) > 255 {
        result = result[:255]
    }

    return result
}
```

**Testing:**

- [ ] Download file with special chars in URL
- [ ] Verify filename sanitized
- [ ] File saves correctly

---

### 4.4: Very Large Files (>10GB)

```go
// Increase buffer size for large files
const bufferSize = 32 * 1024 * 1024  // 32MB

func downloadFile(ctx context.Context, url, path string, p *progress) error {
    // ... existing code ...

    // Copy with larger buffer
    writer := bufio.NewWriterSize(f, bufferSize)
    defer writer.Flush()

    io.Copy(io.MultiWriter(writer, p), resp.Body)
}
```

**Files Modified**: `download_generic.go`  
**Testing**: Download file > 5GB

---

### 4.5: Unknown Content-Length

```go
func downloadFile(ctx context.Context, url, path string, p *progress) error {
    // ... setup ...

    // Check Content-Length
    if resp.ContentLength > 0 {
        p.SetTotal(resp.ContentLength)
    } else {
        // Unknown size - show downloading but no %
        p.SetTotal(-1)  // Indicate unknown
    }

    // ... continue download ...
}
```

**UI Display:**

```html
<!-- When total unknown -->
<p class="progress-text">12 MB / ? (downloading...)</p>
```

**Files Modified**: `download_generic.go`, `progress.go`, `templates/index.html`

---

## Task 5: Security Review

**Duration**: 1-2 hours  
**Status**: ⬜ Not Started

### 5.1: URL Validation

```go
func validateURL(urlStr string) error {
    u, err := url.Parse(urlStr)
    if err != nil {
        return fmt.Errorf("invalid URL format: %w", err)
    }

    // Only HTTP/HTTPS
    if u.Scheme != "http" && u.Scheme != "https" {
        return fmt.Errorf("only HTTP/HTTPS supported, got %s", u.Scheme)
    }

    // Must have host
    if u.Host == "" {
        return fmt.Errorf("URL must include host")
    }

    // Warn on localhost/private IPs (for download manager, usually ok)
    // But block file:// and other protocols

    return nil
}
```

### 5.2: Path Traversal Prevention

```go
func safePath(baseDir, filename string) (string, error) {
    // Resolve to absolute path
    cleanFilename := filepath.Clean(filename)

    // Prevent ../ attacks
    if strings.Contains(cleanFilename, "..") {
        return "", fmt.Errorf("invalid filename: contains ..")
    }

    fullPath := filepath.Join(baseDir, cleanFilename)
    fullPath = filepath.Clean(fullPath)

    // Ensure result is within baseDir
    if !strings.HasPrefix(fullPath, filepath.Clean(baseDir)) {
        return "", fmt.Errorf("path traversal detected")
    }

    return fullPath, nil
}
```

### 5.3: Input Validation

```go
// Validate all user inputs
func validateDownloadRequest(url, filename string, retries int) error {
    // URL
    if url == "" {
        return errors.New("URL required")
    }
    if len(url) > 2000 {
        return errors.New("URL too long")
    }
    if err := validateURL(url); err != nil {
        return err
    }

    // Filename
    if len(filename) > 255 {
        return errors.New("filename too long")
    }
    if filename != "" && strings.ContainsAny(filename, "/\\:*?\"<>|") {
        return errors.New("invalid filename")
    }

    // Retries
    if retries < 0 || retries > 100 {
        return errors.New("retries must be 0-100")
    }

    return nil
}
```

**Files Modified**: `download_generic.go`, `main.go`  
**Testing**: Try various malicious inputs

---

## Task 6: Comprehensive Testing

**Duration**: 3-4 hours  
**Status**: ⬜ Not Started

### 6.1: Manual Testing Checklist

```
CORE FUNCTIONALITY
[ ] Download small file (100MB)
[ ] Download large file (1GB+)
[ ] Pause and resume
[ ] Resume after app restart
[ ] Cancel download
[ ] Verify file integrity (size matches)

QUEUE MANAGEMENT
[ ] Add 5 downloads
[ ] Verify max concurrent respected
[ ] Pause/resume individual
[ ] Pause/resume all
[ ] Reorder by dragging
[ ] Verify priority respected

ERROR HANDLING
[ ] Invalid URL rejected
[ ] 404 handled
[ ] Network timeout
[ ] Disk full warning
[ ] Special chars in filename

PERFORMANCE
[ ] 100+ downloads in history
[ ] Switch between tabs smoothly
[ ] Search doesn't lag
[ ] Real-time updates smooth
[ ] Memory usage reasonable

UI/UX
[ ] All tabs work
[ ] Bulk operations
[ ] Keyboard shortcuts
[ ] Theme toggle
[ ] Settings persist

PERSISTENCE
[ ] History survives restart
[ ] Settings survive restart
[ ] Session recovery works
[ ] .part files created properly
```

### 6.2: Automated Testing (Future)

Create test files:

```go
// download_test.go
func TestDownloadSmallFile(t *testing.T) {
    // Download 1MB test file
    // Verify size
    // Verify no errors
}

func TestResume(t *testing.T) {
    // Start download
    // Interrupt at 50%
    // Verify .part file
    // Resume
    // Verify completion
}

func TestQueueManagement(t *testing.T) {
    // Add 5 downloads
    // Verify order
    // Reorder
    // Verify new order
}
```

**Files Created**: `download_test.go`, etc  
**Run tests**: `go test ./...`

---

## Task 7: Final Commit & Cleanup

**Duration**: 2 hours  
**Status**: ⬜ Not Started

### 7.1: Code Cleanup

**Checklist:**

```bash
# Format code
go fmt ./...

# Run linter
golint ./...

# Build test
go build -o downloader

# Check for unused imports
go mod tidy

# Verify no Ollama references
grep -r "ollama\|OCI\|registry" --include="*.go" --include="*.html" .
# Should return 0 results (except maybe comments)

# Check error handling
grep -r "error" --include="*.go" . | grep "log\|fmt\|return"
```

**Remove:**

- [ ] Debug print statements
- [ ] Commented-out code blocks
- [ ] Unused variables/functions
- [ ] Test files (unless integrated)
- [ ] TODOs without context

---

### 7.2: Document All Changes

Create CHANGELOG.md:

```markdown
# Changelog

## [1.0.0] - 2024-11-14

### Added

- Generic HTTP/HTTPS file downloading
- Pause and resume with Range header support
- Queue management with priority system
- Multiple concurrent downloads (configurable)
- Real-time speed tracking and ETA calculation
- Download history with persistence
- Advanced search and filtering
- Settings panel for customization
- Web UI with Persian language
- CLI mode for batch operations
- Bulk operations (pause all, resume all, delete)
- Statistics dashboard
- Error recovery with retries
- Keyboard shortcuts

### Changed

- Converted from Ollama-specific to generic downloader
- Updated UI labels and form inputs
- Simplified session management
- Improved error messages

### Removed

- OCI/Docker registry logic
- Platform-specific handling
- Model library references
- Unzip functionality
```

---

### 7.3: Final Commit

```bash
git add -A
git commit -m "docs: polish and final optimizations

- Update README.md with generic features and usage
- Update CONTRIBUTING.md with current structure
- Create FEATURES.md with comprehensive feature list
- Improve error messages with helpful suggestions
- Add smart retry logic with exponential backoff
- Performance optimizations (memory, CPU)
- Handle edge cases (file conflicts, low disk space)
- Sanitize filenames and prevent path traversal
- Handle unknown Content-Length
- Security review and hardening
- Comprehensive manual testing checklist
- Code cleanup and linting
- Create CHANGELOG.md
- Final optimization and polish"

git log --oneline | head -5
# Should show:
# - Polish commit
# - Manager commit
# - MVP commit
# - Previous commits
```

---

### 7.4: Prepare for Release

**Create release checklist:**

```
RELEASE CHECKLIST
[ ] All tests pass
[ ] No warnings or errors
[ ] Documentation complete
[ ] CHANGELOG updated
[ ] Version bumped (if applicable)
[ ] README verified
[ ] Screenshots updated
[ ] Feature list complete
[ ] Installation instructions work
[ ] Example commands work
[ ] Tested on at least 2 platforms
[ ] License file present
[ ] Contributing guide present
[ ] Code reviewed
[ ] All edge cases handled
```

---

## Summary

| Task                     | Duration   | Status |
| ------------------------ | ---------- | ------ |
| Update Documentation     | 3-4h       | ⬜     |
| Improve Error Handling   | 2-3h       | ⬜     |
| Performance Optimization | 2-3h       | ⬜     |
| Handle Edge Cases        | 2-3h       | ⬜     |
| Security Review          | 1-2h       | ⬜     |
| Comprehensive Testing    | 3-4h       | ⬜     |
| Final Commit & Cleanup   | 2h         | ⬜     |
| **Total**                | **15-22h** |        |

**Recommended**: 2-3 days of work, do before any public release.

---

## Success Criteria

✅ All MVP and Manager features working  
✅ No compile errors or warnings  
✅ Error messages helpful and actionable  
✅ Performance acceptable with large queues  
✅ Edge cases handled gracefully  
✅ Documentation complete and accurate  
✅ README reflects actual features  
✅ CONTRIBUTING.md up to date  
✅ Code clean and well-organized  
✅ Manual testing checklist passed

---

## Next Steps

1. **Complete MVP and Manager phases** (Days 1-4)
2. **Choose Phase 3 features** if desired (Days 5+)
3. **Polish and document** (Days final)
4. **Release or share** with confidence

---

## Links

- See `PHASE1_MVP.md` for MVP details
- See `PHASE2_MANAGER.md` for Manager details
- See `PHASE3_ADVANCED.md` for optional features
- See `QUICK_START.md` for overview
- See `CHECKLIST.md` for quick reference

---

**Ready for release!** 🚀
