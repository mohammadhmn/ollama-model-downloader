# Step-by-Step Implementation Guide

## 🎯 Goal

Convert Ollama Model Downloader → Full-Featured File Download Manager

## 📋 Prerequisites

- ✅ Branch created: `feature/general-purpose-downloader`
- ✅ Plans documented: See `CONVERSION_PLAN.md`, `ARCHITECTURE_OVERVIEW.md`
- ✅ Current codebase understood
- ✅ Git workflow ready

---

## 🚀 MVP Phase (Iteration 1) - Weeks 1-2

### Step 1: Backend Refactoring (2-3 hours)

#### 1.1 Create new `download.go` with generic logic

**File: `download_generic.go`** (Start fresh)

```go
package main

import (
    "context"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "os"
    "path/filepath"
    "strings"
)

// downloadFile downloads a file from URL with resume support
func downloadFile(ctx context.Context, downloadURL, outputPath string, p *progress) error {
    // 1. Validate URL
    u, err := url.Parse(downloadURL)
    if err != nil {
        return fmt.Errorf("invalid URL: %w", err)
    }

    // 2. Create HTTP request with range support
    req, _ := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)

    // 3. Check if file partially exists
    info, _ := os.Stat(outputPath + ".part")
    if info != nil && info.Size() > 0 {
        req.Header.Set("Range", fmt.Sprintf("bytes=%d-", info.Size()))
    }

    // 4. Make request
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return fmt.Errorf("download failed: %w", err)
    }
    defer resp.Body.Close()

    // 5. Create/open file
    f, err := os.OpenFile(outputPath+".part", os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return err
    }
    defer f.Close()

    // 6. Copy with progress
    written, err := io.Copy(io.MultiWriter(f, p), resp.Body)
    if err != nil {
        return fmt.Errorf("copy failed: %w", err)
    }

    // 7. Rename on success
    return os.Rename(outputPath+".part", outputPath)
}

// extractFilenameFromURL gets filename from URL or Content-Disposition
func extractFilenameFromURL(urlStr string) string {
    u, err := url.Parse(urlStr)
    if err != nil {
        return "download"
    }

    path := filepath.Base(u.Path)
    if path == "" || path == "/" {
        return "download"
    }

    return path
}

// validateURL checks if URL is valid HTTP/HTTPS
func validateURL(urlStr string) error {
    u, err := url.Parse(urlStr)
    if err != nil {
        return fmt.Errorf("invalid URL format: %w", err)
    }

    if u.Scheme != "http" && u.Scheme != "https" {
        return fmt.Errorf("only HTTP/HTTPS supported, got %s", u.Scheme)
    }

    if u.Host == "" {
        return fmt.Errorf("invalid URL: missing host")
    }

    return nil
}
```

#### 1.2 Keep useful progress tracking code

- Move `progress` struct to `progress.go` (already good as-is)
- Keep `SpeedSample` structure for later

#### 1.3 Update session metadata structure

**In `main.go`, replace:**

```go
type sessionMeta struct {
    Model       string
    SessionID   string
    OutZip      string
    StagingRoot string
    Registry    string
    Platform    string
    Concurrency int
    Retries     int
    ...
}
```

**With:**

```go
type sessionMeta struct {
    URL          string    `json:"url"`
    Filename     string    `json:"filename"`
    SessionID    string    `json:"sessionId"`
    OutPath      string    `json:"outPath"`
    StagingRoot  string    `json:"stagingRoot"`
    ExpectedSize int64     `json:"expectedSize"`
    Retries      int       `json:"retries"`
    StartedAt    time.Time `json:"startedAt"`
    LastUpdated  time.Time `json:"lastUpdated"`
    State        string    `json:"state"`
    Message      string    `json:"message"`
}
```

#### 1.4 Remove Ollama functions

- [ ] Delete `parseModel()`
- [ ] Delete `getRegistryToken()`
- [ ] Delete `getManifestOrIndex()`
- [ ] Delete `downloadBlob()`
- [ ] Delete `ollamaModelsDir()`
- [ ] Delete `unzipToDir()`
- [ ] Delete OCI/Docker constants

**Checklist for deletion:**

```go
// In download.go, remove these:
❌ const (mtOCIIndex, mtDockerIndex, mtOCIManifest, mtDockerManifest)
❌ type imageIndex struct
❌ type imageManifest struct
❌ type bearerAuth struct
❌ type modelRef struct
❌ func parseModel()
❌ func getRegistryToken()
❌ func parseBearerChallenge()
❌ func getManifestOrIndex()
❌ func downloadBlob()
❌ func dedupeBlobs()
❌ func ensureStagingRoot()
```

### Step 2: CLI Updates (1 hour)

**File: `main.go` - Update flag parsing**

```go
func main() {
    // Old:
    // flag.StringVar(&opt.registry, "registry", defaultRegistry, "...")
    // flag.StringVar(&opt.platform, "platform", defaultPlatform, "...")

    // New:
    var url string
    flag.StringVar(&url, "url", "", "File URL to download")
    flag.StringVar(&opt.outputDir, "output-dir", "downloads", "Directory to save files")
    flag.StringVar(&opt.outZip, "o", "", "Output filename")
    flag.IntVar(&opt.retries, "retries", 3, "Retry attempts")
    flag.IntVar(&opt.port, "port", 0, "Web server port")
    flag.BoolVar(&opt.verbose, "v", false, "Verbose output")
    flag.Parse()

    // Usage:
    // ./downloader https://example.com/file.zip
    // ./downloader -url https://example.com/file.zip -o myfile.zip
    // ./downloader -port 8080  (start web server)
}
```

### Step 3: Web UI Updates - Form (2 hours)

**File: `templates/index.html`**

**Changes:**

1. Update form input:

   ```html
   <!-- Old -->
   <input name="model" placeholder="نام مدل (مثال: llama3.2)" />

   <!-- New -->
   <input name="url" placeholder="آدرس دانلود (مثال: https://...)" />
   <input name="filename" placeholder="نام فایل (اختیاری)" />
   ```

2. Remove platform/registry/concurrency fields:

   ```html
   <!-- Remove -->
   <input name="platform" />
   <input name="registry" />
   <input name="concurrency" />

   <!-- Keep only -->
   <input name="retries" />
   ```

3. Update labels:

   ```
   "مدیریت دانلود مدل‌های Ollama" → "مدیریت دانلود فایل‌ها"
   "دانلود مدل جدید" → "دانلود فایل جدید"
   "نام مدل" → "آدرس دانلود"
   ```

4. Update session display:

   ```html
   <!-- Old -->
   <h3>{{.RunningSession.Model}}</h3>

   <!-- New -->
   <h3>{{.RunningSession.Filename}} - {{.RunningSession.URL}}</h3>
   ```

### Step 4: Web Handler Updates (2 hours)

**File: `main.go` - Update `/download` handler**

```go
http.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
    // Parse form
    downloadURL := r.FormValue("url")
    filename := r.FormValue("filename")

    // Validate URL
    if err := validateURL(downloadURL); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Extract filename if not provided
    if filename == "" {
        filename = extractFilenameFromURL(downloadURL)
    }

    // Create options
    opt := options{
        url:        downloadURL,
        model:      filename, // Reuse for session ID
        outZip:     filepath.Join(opt.outputDir, filename),
        sessionID:  sanitizeModelName(filename),
        stagingDir: filepath.Join(opt.outputDir, sanitizeModelName(filename)+".staging"),
        retries:    retries,
    }

    // Start download
    beginDownloadSession(opt, "در حال دانلود...")
    http.Redirect(w, r, "/", http.StatusFound)
})
```

### Step 5: Testing MVP (2-3 hours)

**Checklist:**

- [ ] Start web server: `go run . -port 8080`
- [ ] Download small file (100MB)
  - [ ] Check progress updates
  - [ ] Pause and resume
  - [ ] File validates correctly
- [ ] Download large file (1GB+)
  - [ ] Resume after interruption
  - [ ] Progress accurate
- [ ] Invalid URL handling
  - [ ] Show error message
  - [ ] Don't crash
- [ ] CLI mode: `./downloader https://example.com/file.zip`
- [ ] Session persistence
  - [ ] Close app during download
  - [ ] Restart and verify session still there
  - [ ] Can resume

### Step 6: Commit MVP (30 mins)

```bash
git add -A
git commit -m "feat: implement MVP generic file downloader

- Remove all Ollama-specific logic (OCI, registry, platform)
- Add generic HTTP download with resume support
- Update CLI to accept URLs directly
- Simplify sessionMeta for generic downloads
- Update web UI form for URL input
- Update labels and placeholders
- Test with various file types and sizes"
```

---

## 🎨 Manager Phase (Iteration 2) - Weeks 3-4

### Step 7: Create Queue Manager (4 hours)

**File: `download_manager.go`** (New)

```go
package main

import (
    "context"
    "sync"
    "time"
)

type DownloadManager struct {
    downloads     map[string]*Download
    queue         []string  // IDs in order
    running       []string  // Currently downloading IDs
    maxConcurrent int
    mu            sync.RWMutex
    wg            sync.WaitGroup
    ctx           context.Context
    cancel        context.CancelFunc
}

type Download struct {
    ID            string
    URL           string
    Filename      string
    OutputPath    string
    Status        string  // queued, active, paused, completed, error
    Progress      int64   // bytes downloaded
    Total         int64   // total bytes
    StartTime     time.Time
    CompletedTime time.Time
    Error         string
    Priority      int     // 1-10
}

func NewDownloadManager(maxConcurrent int) *DownloadManager {
    ctx, cancel := context.WithCancel(context.Background())
    return &DownloadManager{
        downloads:     make(map[string]*Download),
        queue:         []string{},
        running:       []string{},
        maxConcurrent: maxConcurrent,
        ctx:           ctx,
        cancel:        cancel,
    }
}

func (dm *DownloadManager) AddDownload(url, filename, outputPath string) string {
    id := generateID()  // dl-<timestamp>-<random>

    dl := &Download{
        ID:         id,
        URL:        url,
        Filename:   filename,
        OutputPath: outputPath,
        Status:     "queued",
        StartTime:  time.Now(),
        Priority:   5,  // Default medium priority
    }

    dm.mu.Lock()
    dm.downloads[id] = dl
    dm.queue = append(dm.queue, id)
    dm.mu.Unlock()

    dm.processQueue()
    return id
}

func (dm *DownloadManager) processQueue() {
    dm.mu.Lock()

    // Check how many are currently running
    activeCount := len(dm.running)

    // Start new downloads up to maxConcurrent
    for _, id := range dm.queue {
        if activeCount >= dm.maxConcurrent {
            break
        }

        dl := dm.downloads[id]
        if dl.Status == "queued" {
            dl.Status = "active"
            dm.running = append(dm.running, id)
            activeCount++

            // Start goroutine
            dm.wg.Add(1)
            go dm.downloadWorker(id)
        }
    }

    dm.mu.Unlock()
}

func (dm *DownloadManager) downloadWorker(id string) {
    defer dm.wg.Done()

    dm.mu.RLock()
    dl := dm.downloads[id]
    dm.mu.RUnlock()

    // Perform actual download
    err := downloadFile(dm.ctx, dl.URL, dl.OutputPath, nil)

    dm.mu.Lock()
    if err != nil {
        dl.Status = "error"
        dl.Error = err.Error()
    } else {
        dl.Status = "completed"
        dl.CompletedTime = time.Now()
    }

    // Remove from running
    for i, rid := range dm.running {
        if rid == id {
            dm.running = append(dm.running[:i], dm.running[i+1:]...)
            break
        }
    }

    dm.mu.Unlock()

    // Process next in queue
    dm.processQueue()
}

func (dm *DownloadManager) Pause(id string) error {
    dm.mu.Lock()
    defer dm.mu.Unlock()

    dl, exists := dm.downloads[id]
    if !exists {
        return fmt.Errorf("download not found")
    }

    dl.Status = "paused"
    return nil
}

func (dm *DownloadManager) Resume(id string) error {
    dm.mu.Lock()
    defer dm.mu.Unlock()

    dl, exists := dm.downloads[id]
    if !exists {
        return fmt.Errorf("download not found")
    }

    dl.Status = "queued"
    dm.mu.Unlock()

    dm.processQueue()
    return nil
}

func (dm *DownloadManager) GetAll() []*Download {
    dm.mu.RLock()
    defer dm.mu.RUnlock()

    result := make([]*Download, 0, len(dm.downloads))
    for _, dl := range dm.downloads {
        result = append(result, dl)
    }
    return result
}

func (dm *DownloadManager) GetStatistics() Statistics {
    dm.mu.RLock()
    defer dm.mu.RUnlock()

    var stats Statistics
    for _, dl := range dm.downloads {
        if dl.Status == "completed" {
            stats.TotalFiles++
            stats.TotalBytes += dl.Total
            stats.TotalTime += int64(dl.CompletedTime.Sub(dl.StartTime).Seconds())
        }
    }

    if stats.TotalTime > 0 {
        stats.AverageSpeed = stats.TotalBytes / stats.TotalTime
    }

    return stats
}
```

### Step 8: Create Speed Tracker (2 hours)

**File: `speed_tracker.go`** (New)

```go
package main

import (
    "sync"
    "time"
)

type SpeedTracker struct {
    samples []*SpeedSample
    mu      sync.RWMutex
}

type SpeedSample struct {
    timestamp time.Time
    bytes     int64
}

func NewSpeedTracker() *SpeedTracker {
    return &SpeedTracker{
        samples: make([]*SpeedSample, 0, 10),
    }
}

func (st *SpeedTracker) Record(bytes int64) {
    st.mu.Lock()
    defer st.mu.Unlock()

    st.samples = append(st.samples, &SpeedSample{
        timestamp: time.Now(),
        bytes:     bytes,
    })

    // Keep only last 10 samples (10 seconds at 1/sec)
    if len(st.samples) > 10 {
        st.samples = st.samples[1:]
    }
}

func (st *SpeedTracker) GetSpeed() int64 {
    st.mu.RLock()
    defer st.mu.RUnlock()

    if len(st.samples) < 2 {
        return 0
    }

    first := st.samples[0]
    last := st.samples[len(st.samples)-1]

    timeDiff := last.timestamp.Sub(first.timestamp).Seconds()
    if timeDiff == 0 {
        return 0
    }

    bytesDiff := last.bytes - first.bytes
    return int64(float64(bytesDiff) / timeDiff)
}

func (st *SpeedTracker) GetETA(total, downloaded int64) time.Duration {
    speed := st.GetSpeed()
    if speed <= 0 {
        return 0
    }

    remaining := total - downloaded
    seconds := remaining / speed
    return time.Duration(seconds) * time.Second
}
```

### Step 9: Create History Manager (3 hours)

**File: `history.go`** (New) - Use JSON for MVP, SQLite later

```go
package main

import (
    "encoding/json"
    "os"
    "path/filepath"
    "sync"
    "time"
)

type HistoryManager struct {
    entries []*HistoryEntry
    file    string
    mu      sync.RWMutex
}

type HistoryEntry struct {
    ID           string    `json:"id"`
    URL          string    `json:"url"`
    Filename     string    `json:"filename"`
    FileSize     int64     `json:"fileSize"`
    DownloadedAt time.Time `json:"downloadedAt"`
    Duration     int64     `json:"duration"` // seconds
    Speed        int64     `json:"speed"`    // bytes/sec
    Status       string    `json:"status"`
    Error        string    `json:"error,omitempty"`
}

type Statistics struct {
    TotalFiles     int
    TotalBytes     int64
    TotalTime      int64
    AverageSpeed   int64
    TodayFiles     int
    TodayBytes     int64
    TopDomains     map[string]int64
    FileTypeStats  map[string]int64
}

func NewHistoryManager(dir string) *HistoryManager {
    os.MkdirAll(dir, 0755)
    return &HistoryManager{
        entries: make([]*HistoryEntry, 0),
        file:    filepath.Join(dir, "history.json"),
    }
}

func (hm *HistoryManager) Load() error {
    hm.mu.Lock()
    defer hm.mu.Unlock()

    data, err := os.ReadFile(hm.file)
    if err != nil {
        if os.IsNotExist(err) {
            return nil
        }
        return err
    }

    return json.Unmarshal(data, &hm.entries)
}

func (hm *HistoryManager) AddEntry(entry *HistoryEntry) error {
    hm.mu.Lock()
    hm.entries = append(hm.entries, entry)
    hm.mu.Unlock()

    return hm.Save()
}

func (hm *HistoryManager) Save() error {
    hm.mu.RLock()
    defer hm.mu.RUnlock()

    data, err := json.MarshalIndent(hm.entries, "", "  ")
    if err != nil {
        return err
    }

    return os.WriteFile(hm.file, data, 0644)
}

func (hm *HistoryManager) GetStatistics() Statistics {
    hm.mu.RLock()
    defer hm.mu.RUnlock()

    stats := Statistics{
        TopDomains:    make(map[string]int64),
        FileTypeStats: make(map[string]int64),
    }

    now := time.Now()
    today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

    for _, entry := range hm.entries {
        if entry.Status == "completed" {
            stats.TotalFiles++
            stats.TotalBytes += entry.FileSize
            stats.TotalTime += entry.Duration

            if entry.DownloadedAt.After(today) {
                stats.TodayFiles++
                stats.TodayBytes += entry.FileSize
            }

            // Extract domain from URL
            // Extract extension for file type
        }
    }

    if stats.TotalTime > 0 {
        stats.AverageSpeed = stats.TotalBytes / stats.TotalTime
    }

    return stats
}
```

### Step 10: Update API Routes (3 hours)

**File: `main.go` - Add new API endpoints**

```go
// Global download manager
var downloadMgr *DownloadManager

func init() {
    downloadMgr = NewDownloadManager(4)  // Default 4 concurrent
}

// Add API routes
http.HandleFunc("/api/downloads", func(w http.ResponseWriter, r *http.Request) {
    downloads := downloadMgr.GetAll()
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(downloads)
})

http.HandleFunc("/api/download/add", func(w http.ResponseWriter, r *http.Request) {
    url := r.FormValue("url")
    filename := r.FormValue("filename")

    id := downloadMgr.AddDownload(url, filename, outputPath)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"id": id})
})

http.HandleFunc("/api/statistics", func(w http.ResponseWriter, r *http.Request) {
    stats := downloadMgr.GetStatistics()
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(stats)
})

// Individual download control
http.HandleFunc("/api/download/pause", func(w http.ResponseWriter, r *http.Request) {
    id := r.FormValue("id")
    downloadMgr.Pause(id)
    http.Redirect(w, r, "/", http.StatusFound)
})
```

### Step 11: Update UI - Advanced Tabs (4-5 hours)

**File: `templates/index.html` - Add tabs and sections**

**Changes:**

1. Add "Completed" and "History" tabs
2. Add speed/ETA display in active tab
3. Add bulk operations toolbar
4. Add search/filter section
5. Add statistics widget

```html
<!-- New tab structure -->
<div class="tabs">
  <button onclick="switchTab('active')">Active (1)</button>
  <button onclick="switchTab('queue')">Queue (3)</button>
  <button onclick="switchTab('completed')">Completed (45)</button>
  <button onclick="switchTab('history')">History</button>
</div>

<!-- Active tab - with speed/ETA -->
<div id="tab-active">
  <div class="download-card">
    <h3>large-file.zip</h3>
    <p>Speed: 2.5 MB/s ↓ | ETA: 5m 30s</p>
    <div class="progress">
      <div class="bar" style="width: 43%"></div>
    </div>
    <p>524 MB / 1.2 GB (43%)</p>
  </div>
</div>

<!-- Completed tab -->
<div id="tab-completed">
  <table>
    <tr>
      <td>file1.zip</td>
      <td>234 MB</td>
      <td>Completed in 2m 15s</td>
    </tr>
  </table>
</div>

<!-- History tab with search -->
<div id="tab-history">
  <input type="text" placeholder="Search history..." />
  <div class="stats-widget">
    <p>Total: 45 files, 125 GB</p>
    <p>Average speed: 5.2 MB/s</p>
  </div>
</div>
```

### Step 12: Testing Manager Features (3-4 hours)

**Test Cases:**

- [ ] Add 5 URLs at once to queue
- [ ] Verify proper queuing and ordering
- [ ] Pause individual download
- [ ] Resume individual download
- [ ] Pause all / Resume all
- [ ] Speed calculation accuracy
- [ ] ETA display updates
- [ ] History saves properly
- [ ] Statistics calculate correctly
- [ ] Search/filter works
- [ ] Reorder queue by drag

### Step 13: Final Polish & Commit (2 hours)

**Cleanup:**

- [ ] Remove debug logs
- [ ] Update error messages
- [ ] Test error cases
- [ ] Verify UI responsive
- [ ] Check keyboard shortcuts work
- [ ] Optimize database queries (if using SQLite)

```bash
git add -A
git commit -m "feat: implement full-featured download manager

- Add DownloadManager for queue handling
- Implement SpeedTracker for real-time speed/ETA
- Create HistoryManager for persistent download records
- Add multiple concurrent download support (configurable)
- Add new tabs: Active, Queue, Completed, History
- Add bulk operations (pause all, resume all, delete)
- Add statistics dashboard
- Add advanced search and filtering
- Update API with new endpoints
- Implement proper error recovery
- Full testing and validation"
```

---

## 📝 Verification Checklist

After each phase, verify:

### After MVP Phase

```
✅ Single file downloads work
✅ Pause/Resume functionality
✅ Progress display accurate
✅ Session persistence across restarts
✅ Error handling with user messages
✅ No Ollama code remaining
✅ CLI works with URLs
✅ Web UI is clean and functional
```

### After Manager Phase

```
✅ Multiple downloads in queue
✅ Speed tracking works
✅ ETA calculation accurate
✅ History persists across sessions
✅ Statistics dashboard updates
✅ Bulk operations work smoothly
✅ Search and filtering functional
✅ Keyboard shortcuts responsive
✅ No performance degradation
✅ All edge cases handled
```

---

## 🔄 Git Workflow

```bash
# Create branch (already done)
git checkout -b feature/general-purpose-downloader

# After each major step
git add .
git commit -m "descriptive message"

# Before moving to next phase
git log --oneline  # Verify commits

# When ready to merge (after full testing)
git checkout main
git merge feature/general-purpose-downloader
```

---

## 🐛 Debugging Tips

```bash
# Run with verbose logging
go run . -v

# Test specific URL
go run . https://example.com/file.zip

# Check session files
ls -la downloaded-files/.staging/*/session.json

# View history
cat downloaded-files/.history/history.json

# Monitor downloads
watch 'ps aux | grep downloader'
```

---

## ⏱️ Time Estimates

| Phase            | Duration        | Effort |
| ---------------- | --------------- | ------ |
| MVP Backend      | 3-5 hours       | Medium |
| MVP UI           | 2-3 hours       | Medium |
| MVP Testing      | 3-4 hours       | High   |
| MVP Subtotal     | **8-12 hours**  |        |
|                  |                 |        |
| Manager Backend  | 6-8 hours       | High   |
| Manager UI       | 4-5 hours       | Medium |
| Manager Testing  | 4-5 hours       | High   |
| Manager Polish   | 2-3 hours       | Low    |
| Manager Subtotal | **16-21 hours** |        |
|                  |                 |        |
| **Total**        | **24-33 hours** |        |

Estimate: **3-4 working days** for MVP, **2-3 additional days** for Manager

---

## 🎓 Learning Resources

If you need to understand specific parts:

- HTTP Range headers: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Range
- Go context: https://pkg.go.dev/context
- Go goroutines/sync: https://pkg.go.dev/sync
- SQLite driver: https://github.com/mattn/go-sqlite3

---

## ❓ FAQ

**Q: Should I do all changes at once?**
A: No, follow the steps in order. Commit after each step for safety.

**Q: Can I skip Manager Phase?**
A: Yes, MVP is fully functional. Manager adds polish and features.

**Q: How to handle large files (>5GB)?**
A: The current approach using Range headers should handle them fine.

**Q: Should I add authentication?**
A: Not in MVP. Can add in future if needed (basic auth in URL).

**Q: What about torrent support?**
A: Out of scope for now. Can be added as Phase 3 later.

---

## ✨ Success

You'll know you're done when:

1. You can download any HTTP/HTTPS file
2. You can pause and resume
3. You can manage multiple downloads
4. You see real-time speed and ETA
5. History persists across sessions
6. Code is clean and documented

Good luck! 🚀
