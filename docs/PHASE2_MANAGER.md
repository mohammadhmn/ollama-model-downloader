# Phase 2: Manager Features

**Duration**: 2-3 days  
**Tasks**: 27  
**Effort**: High  
**Prerequisite**: Phase 1 MVP Complete  
**Status**: Not Started

## Goal

Add queue management, multi-download support, speed tracking, and history persistence.

---

## Queue Manager (4-5 hours)

### 5.1: Create `download_manager.go`

**Duration**: 2 hours  
**Status**: ⬜ Not Started

Create new file with download queue management:

```go
type DownloadManager struct {
    downloads     map[string]*Download  // ID → Download
    queue         []string              // IDs in order
    running       []string              // Currently active IDs
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
    Status        string              // queued, active, paused, completed, error
    Progress      int64               // bytes downloaded
    Total         int64               // total bytes (0 if unknown)
    StartTime     time.Time
    CompletedTime time.Time
    Error         string
    Priority      int                 // 1-10, higher = process first
    Speed         int64               // bytes/sec
    ETA           time.Duration
}
```

**Implementation:**

- [ ] `NewDownloadManager(maxConcurrent int)` constructor
- [ ] Thread-safe access with RWMutex
- [ ] Context for graceful shutdown
- [ ] WaitGroup for goroutine tracking

**Files Created**: `download_manager.go`  
**Testing**: Compile without errors

---

### 5.2: Define Download Struct

**Duration**: 30 minutes  
**Status**: ⬜ Not Started

**Full struct definition:**

```go
type Download struct {
    ID            string            `json:"id"`
    URL           string            `json:"url"`
    Filename      string            `json:"filename"`
    OutputPath    string            `json:"outputPath"`
    Status        string            `json:"status"`
    Priority      int               `json:"priority"`
    Progress      int64             `json:"progress"`
    Total         int64             `json:"total"`
    StartTime     time.Time         `json:"startTime"`
    ResumedAt     *time.Time        `json:"resumedAt,omitempty"`
    CompletedTime *time.Time        `json:"completedTime,omitempty"`
    Error         string            `json:"error,omitempty"`
    Speed         int64             `json:"speed"`
    ETA           int64             `json:"eta"` // seconds
    Retries       int               `json:"retries"`
    MaxRetries    int               `json:"maxRetries"`
}
```

**Status values:**

- "queued" - waiting to download
- "active" - currently downloading
- "paused" - user paused
- "completed" - finished successfully
- "error" - failed after retries

**JSON serialization:**

- [ ] All fields JSON-serializable
- [ ] Omit nil/zero fields for cleaner JSON
- [ ] Timestamp format consistent

**Files Modified**: `download_manager.go`  
**Testing**: JSON marshaling works

---

### 5.3: Implement DownloadManager Methods

**Duration**: 2 hours  
**Status**: ⬜ Not Started

**Essential methods:**

```go
// Add new download to queue
func (dm *DownloadManager) AddDownload(url, filename, outputPath string, retries int) (string, error)
  - Validate inputs
  - Generate unique ID
  - Create Download struct
  - Set status to "queued"
  - Add to queue
  - Return ID
  - Trigger queue processing

// Remove download completely
func (dm *DownloadManager) RemoveDownload(id string) error
  - Validate ID exists
  - If active: cancel context
  - Remove from queue and running
  - Return error if not found

// Pause specific download
func (dm *DownloadManager) PauseDownload(id string) error
  - Find download
  - Change status to "paused"
  - Don't remove from running (let finish current chunk)
  - Return on completion

// Resume paused download
func (dm *DownloadManager) ResumeDownload(id string) error
  - Find download
  - Change status back to "queued"
  - Trigger queue processing
  - Resume from .part file

// Get single download
func (dm *DownloadManager) GetDownload(id string) *Download
  - Return copy to avoid data races
  - Nil if not found

// Get all downloads
func (dm *DownloadManager) GetAll() []*Download
  - Return slice of all downloads
  - Make copies to avoid races
  - Optional: sort by priority, then by created time

// Get downloads by status
func (dm *DownloadManager) GetByStatus(status string) []*Download
  - Return only downloads matching status
  - Active, Queued, Completed, Error

// Pause all downloads
func (dm *DownloadManager) PauseAll() error
  - Set all active/queued to "paused"
  - Don't cancel, just pause processing

// Resume all paused downloads
func (dm *DownloadManager) ResumeAll() error
  - Set all paused to "queued"
  - Trigger processing

// Get statistics
func (dm *DownloadManager) GetStatistics() Statistics
  - Count completed
  - Sum downloaded bytes
  - Calculate averages
```

**Files Modified**: `download_manager.go`  
**Testing**: Each method tested individually

---

### 5.4: Implement Multi-Goroutine Worker Pool

**Duration**: 2 hours  
**Status**: ⬜ Not Started

**Worker pool logic:**

```go
// Main queue processor
func (dm *DownloadManager) ProcessQueue()
  - Lock mutex
  - Count currently running
  - For each queued download:
    - If running < maxConcurrent:
      - Change status to "active"
      - Add to running list
      - Spawn downloadWorker goroutine
  - Unlock
  - This is called when download completes

// Worker goroutine
func (dm *DownloadManager) downloadWorker(id string)
  - defer dm.wg.Done()
  - Get download struct
  - Call downloadFile() with progress tracking
  - Handle retries if failed
  - Update status and timestamps
  - Remove from running list
  - Save history entry
  - Call ProcessQueue() again
  - Handle context cancellation (pause/cancel)

// Handle pause/resume
func (dm *DownloadManager) handleContextCancellation(ctx context.Context)
  - Check context.Done()
  - If cancelled: cleanup resources
  - Preserve .part file for resume
```

**Implementation details:**

- [ ] Use goroutines for concurrent downloads
- [ ] Respect maxConcurrent limit
- [ ] Auto-start next when one finishes
- [ ] Handle pause by pausing worker, not stopping
- [ ] Handle cancel by stopping worker
- [ ] Preserve .part file on pause/error
- [ ] Log errors with retry count
- [ ] Graceful shutdown with context

**Files Modified**: `download_manager.go`  
**Testing**: 4 concurrent downloads, add 5th, verify queued

---

## Speed Tracker (2-3 hours)

### 6.1: Create `speed_tracker.go`

**Duration**: 1 hour  
**Status**: ⬜ Not Started

Create speed and ETA calculator:

```go
type SpeedTracker struct {
    samples []*SpeedSample
    mu      sync.RWMutex
}

type SpeedSample struct {
    timestamp time.Time
    bytes     int64
}

func NewSpeedTracker() *SpeedTracker
  - Initialize empty samples slice
  - Capacity of 10 (rolling window)

func (st *SpeedTracker) Record(bytes int64)
  - Add new sample with current time
  - Keep only last 10 samples
  - Automatic sliding window

func (st *SpeedTracker) GetSpeed() int64
  - Return bytes per second
  - If < 2 samples: return 0
  - Calculate: (last.bytes - first.bytes) / time_delta

func (st *SpeedTracker) GetETA(total, downloaded int64) time.Duration
  - Get current speed
  - If speed == 0: return 0
  - remaining := total - downloaded
  - seconds := remaining / speed
  - Return duration
```

**Thread safety:**

- [ ] RWMutex for concurrent access
- [ ] Lock on read operations
- [ ] Lock on write operations

**Files Created**: `speed_tracker.go`  
**Testing**: Calculate speed manually, verify accuracy

---

### 6.2: Implement Speed & ETA Calculation

**Duration**: 1-2 hours  
**Status**: ⬜ Not Started

**Enhancements:**

```go
// Moving average for smoother speed
func (st *SpeedTracker) GetAverageSpeed() int64
  - Current: calculate raw speed
  - Better: use moving average
  - Formula: (current_speed * 0.3) + (previous_speed * 0.7)
  - Smooths sudden changes

// Format for display
func FormatSpeed(bytesPerSecond int64) string
  - 0-1MB/s: format as KB/s
  - 1-1024MB/s: format as MB/s
  - 1024+MB/s: format as GB/s
  - Example: 2548576 → "2.4 MB/s"

func FormatDuration(d time.Duration) string
  - Format as "5m 30s"
  - Or "2h 15m"
  - Or "45s"

func FormatSize(bytes int64) string
  - 0-1024: bytes
  - 1024-1048576: KB
  - 1048576+: MB or GB
  - Example: 1234567 → "1.2 MB"

// Real-time updates
Update Download struct every 1 second:
  - download.Speed = speedTracker.GetSpeed()
  - download.ETA = speedTracker.GetETA(total, downloaded)
```

**Integration:**

- [ ] Call Record() every chunk written
- [ ] Update UI every 1 second with speed/ETA
- [ ] Format speed for display
- [ ] Handle edge cases (speed 0, unknown total)

**Files Modified**: `download_manager.go`, `speed_tracker.go`  
**Testing**: Download file, monitor speed updates

---

## History Manager (3-4 hours)

### 7.1: Create `history.go`

**Duration**: 1 hour  
**Status**: ⬜ Not Started

Create persistent download history:

```go
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
    Status       string    `json:"status"`   // completed, error
    Error        string    `json:"error,omitempty"`
}

type Statistics struct {
    TotalFiles      int                `json:"totalFiles"`
    TotalBytes      int64              `json:"totalBytes"`
    TotalTime       int64              `json:"totalTime"` // seconds
    AverageSpeed    int64              `json:"averageSpeed"`
    TodayFiles      int                `json:"todayFiles"`
    TodayBytes      int64              `json:"todayBytes"`
    TopDomains      map[string]int64   `json:"topDomains"`
    FileTypeStats   map[string]int64   `json:"fileTypeStats"`
}

func NewHistoryManager(dir string) *HistoryManager
  - Create directory if not exists
  - Return manager pointing to history.json file

func (hm *HistoryManager) Load() error
  - Read from .history/history.json
  - Unmarshal JSON
  - Handle file not found gracefully
```

**Directory structure:**

```
downloaded-files/
├── .history/
│   └── history.json
├── .cache/
│   └── statistics.json (future)
└── files/
    ├── file1.zip
    └── file2.iso
```

**Files Created**: `history.go`  
**Testing**: Create, save, load history

---

### 7.2: Implement Persistence

**Duration**: 1 hour  
**Status**: ⬜ Not Started

**Core methods:**

```go
func (hm *HistoryManager) AddEntry(entry *HistoryEntry) error
  - Lock mutex
  - Append to entries slice
  - Unlock
  - Save to disk
  - Return error if save fails

func (hm *HistoryManager) Save() error
  - Lock mutex (read)
  - Marshal entries to JSON (indented)
  - Unlock
  - Write to .history/history.json
  - Create directory if needed
  - Return error if write fails

func (hm *HistoryManager) Load() error
  - Lock mutex (write)
  - Read from file
  - Unmarshal JSON
  - Unlock
  - Return error if fails (ignore not found)

func (hm *HistoryManager) GetEntries() []*HistoryEntry
  - Lock mutex (read)
  - Return copy of entries slice
  - Unlock

func (hm *HistoryManager) DeleteEntry(id string) error
  - Find and remove by ID
  - Save
  - Return error if not found
```

**Auto-save strategy:**

- [ ] Save after each entry added (simpler for MVP)
- [ ] Or: save every 10 entries (batch mode)
- [ ] Choose simpler approach first

**Files Modified**: `history.go`  
**Testing**: Add entry, restart app, verify loaded

---

### 7.3: Implement Statistics Calculation

**Duration**: 1-2 hours  
**Status**: ⬜ Not Started

**Statistics method:**

```go
func (hm *HistoryManager) GetStatistics() Statistics
  - Lock mutex (read)
  - Iterate entries
  - For each completed entry:
    - Add to totalFiles
    - Add fileSize to totalBytes
    - Add duration to totalTime
    - Check if downloadedAt is today
    - Add to todayFiles/todayBytes if today
    - Extract domain from URL
    - Add to TopDomains[domain]
    - Extract file extension
    - Add to FileTypeStats[extension]
  - Calculate averages
  - Return Statistics struct

  // After iteration:
  if totalTime > 0:
    averageSpeed = totalBytes / totalTime
```

**Helper functions:**

```go
func extractDomain(url string) string
  - Parse URL
  - Return host only (no path)
  - Example: "https://github.com/file" → "github.com"

func extractFileType(filename string) string
  - Get extension with dot
  - Example: "archive.tar.gz" → ".tar.gz" or ".gz"
  - Return ".unknown" if no extension

func isToday(t time.Time) bool
  - Get today's date at midnight
  - Check if t is after today's start
```

**Files Modified**: `history.go`  
**Testing**: Add several entries, calculate stats

---

## API Endpoints (3 hours)

### 8.1: Implement Management Endpoints

**Duration**: 2 hours  
**Status**: ⬜ Not Started

Add REST API endpoints in `main.go`:

```go
// Get all downloads
GET /api/downloads
Response:
{
  "downloads": [
    {
      "id": "dl-123",
      "url": "https://...",
      "filename": "file.zip",
      "status": "active",
      "progress": 524288000,
      "total": 1073741824,
      "speed": 5242880,
      "eta": 105
    }
  ]
}

// Add new download
POST /api/download/add
Body: form or JSON
  url: "https://..."
  filename: "file.zip" (optional)
  retries: 3
Response: { "id": "dl-123" }

// Pause single
POST /api/download/{id}/pause
Response: { "status": "ok" }

// Resume single
POST /api/download/{id}/resume
Response: { "status": "ok" }

// Cancel single
POST /api/download/{id}/cancel
Response: { "status": "ok" }

// Delete from queue
DELETE /api/download/{id}
Response: { "status": "ok" }

// Pause all
POST /api/downloads/pause-all
Response: { "status": "ok" }

// Resume all
POST /api/downloads/resume-all
Response: { "status": "ok" }
```

**Implementation pattern:**

```go
http.HandleFunc("/api/download/{id}/pause", func(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")  // or parse from path
    err := downloadMgr.PauseDownload(id)
    if err != nil {
        http.Error(w, err.Error(), http.StatusNotFound)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
})
```

**Error handling:**

- [ ] 400 Bad Request for missing params
- [ ] 404 Not Found for invalid IDs
- [ ] 200 OK for success
- [ ] Error messages in JSON

**Files Modified**: `main.go`  
**Testing**: curl commands for each endpoint

---

### 8.2: Implement Statistics Endpoint

**Duration**: 1 hour  
**Status**: ⬜ Not Started

Add statistics and history endpoints:

```go
// Get statistics
GET /api/statistics
Response:
{
  "totalFiles": 45,
  "totalBytes": 134217728,
  "averageSpeed": 5242880,
  "todayFiles": 3,
  "todayBytes": 1073741824,
  "topDomains": {
    "github.com": 67108864,
    "releases.ubuntu.com": 67108864
  },
  "fileTypeStats": {
    ".zip": 45,
    ".iso": 12,
    ".tar": 89
  }
}

// Get history entries
GET /api/history?limit=50&offset=0
Response:
{
  "entries": [
    {
      "id": "hist-123",
      "filename": "file.zip",
      "fileSize": 1073741824,
      "downloadedAt": "2024-11-14T10:30:00Z",
      "duration": 200,
      "speed": 5242880,
      "status": "completed"
    }
  ],
  "total": 150
}
```

**Implementation:**

```go
http.HandleFunc("/api/statistics", func(w http.ResponseWriter, r *http.Request) {
    stats := downloadMgr.GetStatistics()
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(stats)
})

http.HandleFunc("/api/history", func(w http.ResponseWriter, r *http.Request) {
    limit := parseQueryInt(r, "limit", 50)
    offset := parseQueryInt(r, "offset", 0)
    entries := historyMgr.GetEntries()
    // Apply limit/offset
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(entries[offset:offset+limit])
})
```

**Files Modified**: `main.go`  
**Testing**: Browser or curl to test endpoints

---

## Web UI - Tab Structure (4-5 hours)

### 9.1: Create Tab Navigation

**Duration**: 30 minutes  
**Status**: ⬜ Not Started

Update `templates/index.html`:

```html
<div class="tabs-container">
  <div class="tabs">
    <button class="tab-btn active" onclick="switchTab('active')">
      فعال <span class="badge" id="count-active">0</span>
    </button>
    <button class="tab-btn" onclick="switchTab('queue')">
      صف <span class="badge" id="count-queue">0</span>
    </button>
    <button class="tab-btn" onclick="switchTab('completed')">
      انجام‌شده <span class="badge" id="count-completed">0</span>
    </button>
    <button class="tab-btn" onclick="switchTab('history')">تاریخچه</button>
  </div>

  <div class="tabs-content">
    <div id="tab-active" class="tab-pane active"></div>
    <div id="tab-queue" class="tab-pane"></div>
    <div id="tab-completed" class="tab-pane"></div>
    <div id="tab-history" class="tab-pane"></div>
  </div>
</div>

<script>
  function switchTab(tabName) {
    // Hide all
    document.querySelectorAll(".tab-pane").forEach((el) => {
      el.classList.remove("active");
    });
    // Show selected
    document.getElementById(`tab-${tabName}`).classList.add("active");
  }
</script>
```

**CSS styling:**

- [ ] Tab buttons with borders
- [ ] Active tab highlight
- [ ] Smooth transitions
- [ ] Badge styling for counts

**Files Modified**: `templates/index.html`  
**Testing**: Click tabs, they switch

---

### 9.2: Implement Active Tab

**Duration**: 1 hour  
**Status**: ⬜ Not Started

Display currently downloading files:

```html
<div id="tab-active">
  <div id="active-downloads" class="downloads-list">
    <!-- Rendered from API -->
  </div>
</div>
```

**Card template (for each download):**

```html
<div class="download-card active" data-id="dl-123">
  <div class="card-header">
    <h3>file.zip</h3>
    <div class="speed-eta">
      <span class="speed">↓ 2.5 MB/s</span>
      <span class="eta">5m 30s</span>
    </div>
  </div>

  <div class="progress-container">
    <div class="progress-bar">
      <div class="progress-fill" style="width: 43%"></div>
    </div>
    <p class="progress-text">524 MB / 1.2 GB (43%)</p>
  </div>

  <div class="card-actions">
    <button onclick="pauseDownload('dl-123')">مکث</button>
    <button onclick="resumeDownload('dl-123')">ادامه</button>
    <button onclick="cancelDownload('dl-123')">لغو</button>
  </div>
</div>
```

**JavaScript:**

```js
async function fetchDownloads() {
  const res = await fetch("/api/downloads");
  const data = await res.json();

  // Update active tab
  const activeDownloads = data.filter((d) => d.status === "active");
  renderActiveTab(activeDownloads);

  // Update other tabs
  // Update badge counts
}

function renderActiveTab(downloads) {
  const container = document.getElementById("active-downloads");
  container.innerHTML = "";

  downloads.forEach((dl) => {
    const card = createDownloadCard(dl);
    container.appendChild(card);
  });
}

// Auto-refresh every 1 second
setInterval(fetchDownloads, 1000);
```

**Files Modified**: `templates/index.html`  
**Testing**: Start download, see real-time updates

---

### 9.3: Implement Queue Tab

**Duration**: 1 hour  
**Status**: ⬜ Not Started

List downloads waiting to download:

```html
<div id="tab-queue">
  <div id="queue-downloads" class="downloads-list sortable"></div>
</div>
```

**Features:**

- [ ] List all queued downloads
- [ ] Show priority (1-10)
- [ ] Drag to reorder
- [ ] Priority selector per item
- [ ] Move to Top/Bottom buttons
- [ ] Pause/Resume/Cancel buttons

**Card template:**

```html
<div class="download-card queued draggable" draggable="true">
  <div class="drag-handle">⋮</div>
  <div class="card-content">
    <h3>file.zip</h3>
    <p>https://example.com/file.zip</p>
    <p class="file-size">1.2 GB</p>
  </div>
  <div class="priority-selector">
    <label>اولویت:</label>
    <select onchange="setPriority('id', this.value)">
      <option value="1">بسیار پایین</option>
      <option value="5" selected>عادی</option>
      <option value="10">بسیار بالا</option>
    </select>
  </div>
  <div class="card-actions">
    <button onclick="moveToTop('id')">⬆ بالا</button>
    <button onclick="moveToBottom('id')">⬇ پایین</button>
    <button onclick="pauseDownload('id')">مکث</button>
    <button onclick="removeDownload('id')">حذف</button>
  </div>
</div>
```

**Drag & Drop:**

```js
// Sort by drag
const sortable = document.getElementById("queue-downloads");
const sortableInstance = Sortable.create(sortable, {
  animation: 150,
  onEnd: function (evt) {
    const newOrder = Array.from(sortable.children).map((el) => el.dataset.id);
    // Send to backend
  },
});
```

**Files Modified**: `templates/index.html`  
**Testing**: Add to queue, drag to reorder

---

### 9.4: Implement Completed Tab

**Duration**: 1 hour  
**Status**: ⬜ Not Started

Show successfully downloaded files:

```html
<div id="tab-completed">
  <div id="completed-downloads" class="downloads-list"></div>
</div>
```

**Card template:**

```html
<div class="download-card completed">
  <div class="card-content">
    <h3>file.zip</h3>
    <p>اندازه: 1.2 GB</p>
    <p>مدت‌زمان: 3m 45s</p>
    <p>سرعت: 5.2 MB/s</p>
    <p class="download-date">۱۴ نوامبر ۲۰۲۴</p>
  </div>
  <div class="card-actions">
    <button onclick="openFile('file.zip')">📂 باز کن</button>
    <button onclick="openFolder('id')">📁 پوشه</button>
    <button onclick="redownload('id')">🔄 دوباره</button>
    <button onclick="deleteFile('id')">🗑️ حذف</button>
  </div>
</div>
```

**Files Modified**: `templates/index.html`  
**Testing**: Complete a download, see in tab

---

### 9.5: Implement History Tab

**Duration**: 1 hour  
**Status**: ⬜ Not Started

Display all past downloads with search:

```html
<div id="tab-history">
  <div class="history-toolbar">
    <input
      type="text"
      placeholder="جستجو..."
      id="history-search"
      onkeyup="filterHistory(this.value)"
    />

    <div class="stats-widget">
      <p>کل دانلود: <strong id="stat-total-files">0</strong> فایل</p>
      <p>کل داده: <strong id="stat-total-bytes">0</strong></p>
      <p>سرعت متوسط: <strong id="stat-avg-speed">0</strong></p>
      <p>دانلود‌های امروز: <strong id="stat-today-files">0</strong></p>
    </div>
  </div>

  <table id="history-table" class="history-table">
    <thead>
      <tr>
        <th onclick="sortHistory('filename')">فایل</th>
        <th onclick="sortHistory('fileSize')">اندازه</th>
        <th onclick="sortHistory('downloadedAt')">تاریخ</th>
        <th onclick="sortHistory('duration')">مدت</th>
        <th>سرعت</th>
        <th>عملیات</th>
      </tr>
    </thead>
    <tbody id="history-tbody"></tbody>
  </table>
</div>
```

**Functionality:**

- [ ] Load history from `/api/history`
- [ ] Search by filename
- [ ] Sort by column
- [ ] Display statistics
- [ ] Delete entry button
- [ ] Export option (future)

**Files Modified**: `templates/index.html`  
**Testing**: View history, search, sort

---

### 9.6: Add Bulk Operations

**Duration**: 1 hour  
**Status**: ⬜ Not Started

Implement multi-select with actions:

```html
<div class="bulk-toolbar hidden" id="bulk-toolbar">
  <label>
    <input type="checkbox" id="select-all" onchange="selectAll(this.checked)" />
    انتخاب همه
  </label>

  <div class="bulk-actions">
    <span id="selected-count">0 انتخاب شده</span>
    <button onclick="bulkPause()">مکث</button>
    <button onclick="bulkResume()">ادامه</button>
    <button onclick="bulkDelete()">حذف</button>
    <button onclick="bulkMoveToTop()">⬆ بالا</button>
  </div>
</div>

<!-- Add checkbox to each card -->
<div class="download-card">
  <input type="checkbox" class="card-checkbox" onchange="updateSelection()" />
  <!-- Rest of card -->
</div>
```

**JavaScript:**

```js
let selectedDownloads = new Set();

function toggleDownload(id) {
  if (selectedDownloads.has(id)) {
    selectedDownloads.delete(id);
  } else {
    selectedDownloads.add(id);
  }
  updateSelection();
}

function updateSelection() {
  // Show/hide bulk toolbar
  const toolbar = document.getElementById("bulk-toolbar");
  toolbar.classList.toggle("hidden", selectedDownloads.size === 0);

  // Update count
  document.getElementById(
    "selected-count"
  ).textContent = `${selectedDownloads.size} انتخاب شده`;
}

async function bulkPause() {
  for (const id of selectedDownloads) {
    await fetch(`/api/download/${id}/pause`, { method: "POST" });
  }
  selectedDownloads.clear();
  updateSelection();
  fetchDownloads();
}
```

**Files Modified**: `templates/index.html`  
**Testing**: Select multiple, pause/delete

---

## Testing (4-5 hours)

### 10.1: Test Queue Management

**Duration**: 1 hour  
**Status**: ⬜ Not Started

**Test case:**

1. Start app and add 5 URLs to queue
2. Verify all show as "queued"
3. Verify counts in tab: "صف (5)"
4. Verify order maintained
5. Drag to reorder in UI
6. Verify backend reflects new order
7. Set priorities (high, medium, low)
8. Verify higher priority downloads first
9. Complete one download
10. Verify next starts

**Test script:**

```bash
# Add 5 downloads
for i in {1..5}; do
  curl -X POST http://localhost:8080/api/download/add \
    -d "url=https://example.com/file$i.zip&retries=3"
done

# Check status
curl http://localhost:8080/api/downloads | jq '.[] | {id, status, priority}'
```

**Files Modified**: None  
**Result**: ✅ Queue behaves correctly

---

### 10.2: Test Concurrent Downloads

**Duration**: 1 hour  
**Status**: ⬜ Not Started

**Test case:**

1. Set max concurrent to 4
2. Add 6 downloads
3. Verify exactly 4 are "active"
4. Verify 2 are "queued"
5. Monitor speed/ETA on each
6. Pause one active download
7. Verify queued moves to active
8. Resume paused
9. Verify it goes back to queue or restarts
10. Let all complete

**Expected behavior:**

- Never more than 4 downloading
- Queue processed in order
- Speed fairly distributed
- No downloads left hanging

**Files Modified**: None  
**Result**: ✅ Concurrency works

---

### 10.3: Test Pause/Resume Operations

**Duration**: 1 hour  
**Status**: ⬜ Not Started

**Individual pause/resume:**

1. Start download
2. Pause at 30%
3. Verify status shows "paused"
4. Verify progress preserved
5. Verify .part file preserved
6. Resume
7. Verify continues from exact offset
8. Verify no bytes redownloaded
9. Complete and verify file correct

**Pause/Resume All:**

1. Start 3 downloads
2. Click "pause all"
3. Verify all show "paused"
4. Verify progress preserved
5. Click "resume all"
6. Verify all resume together
7. Let all complete

**Files Modified**: None  
**Result**: ✅ Pause/resume preserves progress

---

### 10.4: Test Speed and ETA Accuracy

**Duration**: 1 hour  
**Status**: ⬜ Not Started

**Test procedure:**

1. Download 1GB file to stable connection
2. Record speed reading every 10 seconds
3. Verify speed moving average smooths spikes
4. Verify ETA updates correctly
5. Compare ETA vs actual completion time
6. Acceptable error: ±20%

**Manual calculation:**

```
Downloaded: 524 MB
Total: 1024 MB
Speed: 5 MB/s
ETA should be: (1024 - 524) / 5 = ~100 seconds
```

**Verification:**

- [ ] Speed shows as "X.X MB/s"
- [ ] ETA shows as "5m 30s"
- [ ] ETA decreases as download progresses
- [ ] ETA accuracy within 20% of actual
- [ ] Speed smoothly updates (no jitter)

**Files Modified**: None  
**Result**: ✅ Speed/ETA accurate

---

### 10.5: Test History and Persistence

**Duration**: 1 hour  
**Status**: ⬜ Not Started

**Test case:**

1. Complete 5 downloads
2. Verify entries appear in History tab
3. Check `.history/history.json` file exists
4. Verify JSON is valid
5. Restart app
6. Go to History tab
7. Verify entries still there
8. Statistics should show all 5 downloads
9. Search history by filename
10. Filter by date range

**History file structure:**

```json
[
  {
    "id": "hist-001",
    "filename": "file.zip",
    "fileSize": 1073741824,
    "downloadedAt": "2024-11-14T10:30:00Z",
    "duration": 600,
    "speed": 5242880,
    "status": "completed"
  }
]
```

**Statistics verification:**

- [ ] totalFiles = 5
- [ ] totalBytes = sum of all sizes
- [ ] averageSpeed = reasonable average
- [ ] topDomains shows correct domains
- [ ] fileTypeStats counts extensions correctly

**Files Modified**: None  
**Result**: ✅ History persists correctly

---

### 10.6: Test UI Functionality

**Duration**: 1 hour  
**Status**: ⬜ Not Started

**Tab switching:**

- [ ] Click each tab, content appears
- [ ] Counts update in tab labels
- [ ] Active tab highlighted

**Bulk operations:**

- [ ] Check boxes appear on cards
- [ ] Select individual, counts update
- [ ] Select all checkbox works
- [ ] Bulk toolbar appears/hides
- [ ] Pause selected → only selected pause
- [ ] Resume selected → only selected resume
- [ ] Delete selected → confirms then deletes

**Real-time updates:**

- [ ] Progress bars move smoothly
- [ ] Speed updates every 1 second
- [ ] ETA decreases
- [ ] Download moves from Active → Completed
- [ ] Tab counts update automatically

**Search/Filter:**

- [ ] Type in search, results filter
- [ ] Results match filename and URL
- [ ] Sort by column in history
- [ ] Filter by date range (future feature)

**Files Modified**: None  
**Result**: ✅ UI responsive and functional

---

## Finalization

### 11.1: Code Review & Testing

**Duration**: 2 hours  
**Status**: ⬜ Not Started

**Full test suite:**

- [ ] All existing features still work
- [ ] No regressions from Phase 1
- [ ] New features work as specified
- [ ] Performance acceptable
- [ ] Memory usage reasonable
- [ ] No goroutine leaks
- [ ] Graceful error handling

**Code quality:**

- [ ] No compile errors
- [ ] No warnings
- [ ] Consistent style (gofmt)
- [ ] Comments are clear
- [ ] Error messages helpful
- [ ] Logging appropriate

---

### 11.2: Commit Changes

**Duration**: 30 minutes  
**Status**: ⬜ Not Started

```bash
git add -A
git commit -m "feat: implement full-featured download manager

- Add DownloadManager for queue handling with concurrent downloads
- Implement SpeedTracker for real-time speed and ETA calculation
- Create HistoryManager for persistent download records (JSON)
- Support multiple concurrent downloads (configurable limit)
- Add new tabs: Active, Queue, Completed, History
- Implement bulk operations (pause all, resume all, delete)
- Add comprehensive statistics dashboard
- Add advanced search and filtering
- Implement new API endpoints for all operations
- Support pause/resume for individual and all downloads
- Add real-time progress updates (1 second interval)
- Implement priority system for queue ordering
- Add download persistence across restarts
- Full testing and validation of all features"
```

---

## Manager Phase Summary

**Completed Tasks**: 27  
**Total Duration**: 20-25 hours  
**Next Phase**: Phase 3 - Advanced Features (optional)

**What Works Now:**
✅ Multiple concurrent downloads  
✅ Queue management with priority  
✅ Real-time speed and ETA tracking  
✅ Download history and statistics  
✅ Advanced tab-based UI  
✅ Bulk operations  
✅ Complete API for integration

**What's Coming:**
→ Bandwidth limiting  
→ Custom headers support  
→ Batch URL input  
→ Drag & drop  
→ Advanced search filters  
→ Keyboard shortcuts  
→ Dark/Light theme  
→ Statistics charts

---

## Links

- See `PHASE3_ADVANCED.md` for optional features
- See `PHASE4_POLISH.md` for final polish
- See `CHECKLIST.md` for quick reference
- See `FULL_FEATURED_ROADMAP.md` for architecture
