# Phase 5: Automation & Scheduling

**Duration**: 1-2 weeks (optional)  
**Tasks**: 8  
**Effort**: High  
**Prerequisite**: Phase 4 (Polish) Complete  
**Status**: Not Started

## Goal

Add automated scheduling, torrent support, mirror fallback, CLI scripting, API webhooks, and deployment tools for enterprise use cases.

---

## Task 1: Download Scheduling

**Duration**: 2-3 hours  
**Status**: ⬜ Not Started

### 1.1: Create `scheduler.go`

Create scheduled download functionality:

```go
type Schedule struct {
    ID          string
    Name        string
    URL         string
    Filename    string
    CronExpr    string                  // "0 2 * * *" = 2 AM daily
    Enabled     bool
    LastRun     time.Time
    NextRun     time.Time
    MaxRetries  int
    CreatedAt   time.Time
}

type Scheduler struct {
    schedules   map[string]*Schedule
    mu          sync.RWMutex
    cron        *cron.Cron
    dm          *DownloadManager
}

func NewScheduler(dm *DownloadManager) *Scheduler {
    return &Scheduler{
        schedules: make(map[string]*Schedule),
        cron:      cron.New(),
        dm:        dm,
    }
}

// Add schedule
func (s *Scheduler) AddSchedule(schedule *Schedule) error {
    if schedule.ID == "" {
        schedule.ID = generateID()
    }

    // Parse cron expression
    _, err := cron.ParseStandard(schedule.CronExpr)
    if err != nil {
        return fmt.Errorf("invalid cron: %w", err)
    }

    s.mu.Lock()
    defer s.mu.Unlock()

    s.schedules[schedule.ID] = schedule

    if schedule.Enabled {
        s.cron.AddFunc(schedule.CronExpr, func() {
            s.executeSchedule(schedule.ID)
        })
    }

    return nil
}

// Execute scheduled download
func (s *Scheduler) executeSchedule(id string) error {
    s.mu.RLock()
    schedule := s.schedules[id]
    s.mu.RUnlock()

    if schedule == nil {
        return fmt.Errorf("schedule not found")
    }

    // Add to download manager
    dl := &Download{
        URL:         schedule.URL,
        Filename:    schedule.Filename,
        MaxRetries:  schedule.MaxRetries,
    }

    if err := s.dm.AddDownload(dl); err != nil {
        return err
    }

    // Update last run
    s.mu.Lock()
    schedule.LastRun = time.Now()
    schedule.NextRun = calculateNextRun(schedule.CronExpr)
    s.mu.Unlock()

    return nil
}

// List all schedules
func (s *Scheduler) ListSchedules() []*Schedule {
    s.mu.RLock()
    defer s.mu.RUnlock()

    schedules := make([]*Schedule, 0, len(s.schedules))
    for _, sched := range s.schedules {
        schedules = append(schedules, sched)
    }
    return schedules
}

// Delete schedule
func (s *Scheduler) DeleteSchedule(id string) error {
    s.mu.Lock()
    delete(s.schedules, id)
    s.mu.Unlock()

    // TODO: Remove from cron
    return nil
}

// Persistence
func (s *Scheduler) SaveSchedules(path string) error {
    // Save to JSON
    data, _ := json.Marshal(s.ListSchedules())
    return os.WriteFile(path, data, 0644)
}

func (s *Scheduler) LoadSchedules(path string) error {
    data, err := os.ReadFile(path)
    if err != nil {
        return err
    }

    var schedules []*Schedule
    if err := json.Unmarshal(data, &schedules); err != nil {
        return err
    }

    for _, schedule := range schedules {
        s.AddSchedule(schedule)
    }
    return nil
}
```

**Files Created**: `internal/scheduler/scheduler.go`  
**Dependencies**: `github.com/robfig/cron/v3`  
**Testing**: Create schedule, verify it runs at specified time

---

### 1.2: Add Scheduler API Endpoints

Add to `main.go`:

```go
// GET /api/schedules
func handleListSchedules(w http.ResponseWriter, r *http.Request) {
    schedules := scheduler.ListSchedules()
    json.NewEncoder(w).Encode(schedules)
}

// POST /api/schedules
func handleAddSchedule(w http.ResponseWriter, r *http.Request) {
    var schedule Schedule
    json.NewDecoder(r.Body).Decode(&schedule)

    if err := scheduler.AddSchedule(&schedule); err != nil {
        http.Error(w, err.Error(), 400)
        return
    }

    json.NewEncoder(w).Encode(schedule)
}

// DELETE /api/schedules/{id}
func handleDeleteSchedule(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    scheduler.DeleteSchedule(id)
    w.WriteHeader(http.StatusOK)
}

// PUT /api/schedules/{id}
func handleUpdateSchedule(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    var schedule Schedule
    json.NewDecoder(r.Body).Decode(&schedule)

    schedule.ID = id
    scheduler.UpdateSchedule(&schedule)
    json.NewEncoder(w).Encode(schedule)
}
```

**Files Modified**: `main.go`  
**Testing**: Create/read/update/delete schedules via API

---

### 1.3: Add Scheduler UI Tab

Add to `templates/index.html`:

```html
<!-- Schedules Tab -->
<div id="schedules-tab" class="tab-content hidden">
  <div class="controls">
    <button onclick="showScheduleDialog()">➕ جدول بندی جدید</button>
  </div>

  <table id="schedules-table">
    <thead>
      <tr>
        <th>نام</th>
        <th>زمان بندی</th>
        <th>آخرین اجرا</th>
        <th>بعدی</th>
        <th>عملیات</th>
      </tr>
    </thead>
    <tbody id="schedules-body"></tbody>
  </table>
</div>

<!-- Schedule Dialog -->
<div id="schedule-dialog" class="modal hidden">
  <div class="modal-content">
    <h3>جدول بندی جدید</h3>

    <input type="text" id="schedule-name" placeholder="نام جدول بندی" />
    <input type="text" id="schedule-url" placeholder="URL دانلود" />
    <input type="text" id="schedule-filename" placeholder="نام فایل" />
    <input type="text" id="schedule-cron" placeholder="Cron (0 2 * * *)" />

    <button onclick="saveSchedule()">ذخیره</button>
    <button onclick="closeScheduleDialog()">لغو</button>
  </div>
</div>

<script>
  async function loadSchedules() {
    const res = await fetch("/api/schedules");
    const schedules = await res.json();

    const tbody = document.getElementById("schedules-body");
    tbody.innerHTML = "";

    schedules.forEach((schedule) => {
      const row = tbody.insertRow();
      row.innerHTML = `
            <td>${schedule.Name}</td>
            <td>${schedule.CronExpr}</td>
            <td>${formatDate(schedule.LastRun)}</td>
            <td>${formatDate(schedule.NextRun)}</td>
            <td>
                <button onclick="deleteSchedule('${schedule.ID}')">حذف</button>
            </td>
        `;
    });
  }

  async function saveSchedule() {
    const schedule = {
      Name: document.getElementById("schedule-name").value,
      URL: document.getElementById("schedule-url").value,
      Filename: document.getElementById("schedule-filename").value,
      CronExpr: document.getElementById("schedule-cron").value,
      Enabled: true,
    };

    const res = await fetch("/api/schedules", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(schedule),
    });

    loadSchedules();
    closeScheduleDialog();
  }

  function showScheduleDialog() {
    document.getElementById("schedule-dialog").classList.remove("hidden");
  }

  function closeScheduleDialog() {
    document.getElementById("schedule-dialog").classList.add("hidden");
  }
</script>
```

**Files Modified**: `templates/index.html`  
**Testing**: Create schedule, verify it appears in list

---

## Task 2: Mirror Fallback Support

**Duration**: 2-3 hours  
**Status**: ⬜ Not Started

### 2.1: Create `mirror_manager.go`

```go
type Mirror struct {
    URL       string
    Priority  int       // 1-10, higher = try first
    LastSeen  time.Time
    Healthy   bool
}

type DownloadWithMirrors struct {
    Primary   string
    Mirrors   []Mirror
    RetryWith []string  // Queue of mirrors to try
}

type MirrorManager struct {
    mirrors map[string][]Mirror
    mu      sync.RWMutex
}

func (mm *MirrorManager) AddMirror(filename string, mirror Mirror) {
    mm.mu.Lock()
    defer mm.mu.Unlock()

    mm.mirrors[filename] = append(mm.mirrors[filename], mirror)
    // Sort by priority
    sort.SliceStable(mm.mirrors[filename], func(i, j int) bool {
        return mm.mirrors[filename][i].Priority > mm.mirrors[filename][j].Priority
    })
}

func (mm *MirrorManager) GetNextMirror(filename string) (string, error) {
    mm.mu.RLock()
    mirrors := mm.mirrors[filename]
    mm.mu.RUnlock()

    for _, mirror := range mirrors {
        if mirror.Healthy {
            return mirror.URL, nil
        }
    }

    return "", fmt.Errorf("no healthy mirrors available")
}

// Test mirror health
func (mm *MirrorManager) TestMirror(url string) bool {
    resp, err := http.Head(url)
    if err != nil || resp.StatusCode >= 400 {
        return false
    }
    return true
}

// Periodic health check
func (mm *MirrorManager) StartHealthCheck(interval time.Duration) {
    ticker := time.NewTicker(interval)
    go func() {
        for range ticker.C {
            mm.mu.Lock()
            for filename, mirrors := range mm.mirrors {
                for i := range mirrors {
                    mirrors[i].Healthy = mm.TestMirror(mirrors[i].URL)
                }
            }
            mm.mu.Unlock()
        }
    }()
}
```

**Files Created**: `internal/mirror/mirror_manager.go`  
**Testing**: Add mirrors, verify fallback on failure

---

### 2.2: Integrate Mirror Fallback

Update `download_generic.go`:

```go
func downloadFileWithMirrors(ctx context.Context, urls []string, path string, p *progress) error {
    var lastErr error

    for _, url := range urls {
        err := downloadFile(ctx, url, path, p)
        if err == nil {
            return nil  // Success
        }

        lastErr = err
        // Try next mirror
    }

    return fmt.Errorf("all mirrors failed: %w", lastErr)
}
```

**Files Modified**: `download_generic.go`  
**Testing**: Simulate mirror failure, verify fallback

---

## Task 3: Batch Import from Text

**Duration**: 1-2 hours  
**Status**: ⬜ Not Started

### 3.1: Add Batch Import UI

```html
<!-- Batch Import Button -->
<button onclick="showBatchImport()">📋 وارد کردن دسته‌ای</button>

<!-- Batch Import Dialog -->
<div id="batch-import-dialog" class="modal hidden">
  <div class="modal-content">
    <h3>وارد کردن دسته‌ای</h3>

    <textarea id="batch-urls" placeholder="یک URL در هر سطر..."></textarea>

    <div id="batch-preview">
      <h4>پیش‌نمایش</h4>
      <ul id="batch-preview-list"></ul>
    </div>

    <button onclick="importBatch()">وارد کردن</button>
    <button onclick="closeBatchImport()">لغو</button>
  </div>
</div>

<script>
  function showBatchImport() {
    document.getElementById("batch-import-dialog").classList.remove("hidden");
  }

  function closeBatchImport() {
    document.getElementById("batch-import-dialog").classList.add("hidden");
  }

  document.getElementById("batch-urls").addEventListener("input", (e) => {
    const urls = e.target.value.split("\n").filter((u) => u.trim());
    const list = document.getElementById("batch-preview-list");
    list.innerHTML = urls.map((u) => `<li>${u}</li>`).join("");
  });

  async function importBatch() {
    const urls = document
      .getElementById("batch-urls")
      .value.split("\n")
      .filter((u) => u.trim());

    for (const url of urls) {
      await fetch("/api/downloads", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ url }),
      });
    }

    closeBatchImport();
    loadDownloads();
  }
</script>
```

**Files Modified**: `templates/index.html`  
**Testing**: Import multiple URLs at once

---

### 3.2: Add Validation

```go
func validateBatchURLs(urls []string) []string {
    valid := make([]string, 0)

    for _, url := range urls {
        url = strings.TrimSpace(url)
        if url == "" {
            continue
        }

        if err := validateURL(url); err == nil {
            valid = append(valid, url)
        }
    }

    return valid
}
```

**Files Modified**: `download_generic.go`

---

## Task 4: CLI Improvements

**Duration**: 2-3 hours  
**Status**: ⬜ Not Started

### 4.1: Add Advanced CLI Flags

Update `main.go`:

```go
func init() {
    flag.StringVar(&cliUrl, "url", "", "Download URL")
    flag.StringVar(&cliOutput, "o", "", "Output filename")
    flag.StringVar(&cliOutputDir, "output-dir", "downloads", "Download directory")
    flag.IntVar(&cliPort, "port", 8080, "Web server port")
    flag.IntVar(&cliRetries, "retries", 3, "Max retries")
    flag.BoolVar(&cliVerbose, "v", false, "Verbose output")
    flag.BoolVar(&cliDaemon, "daemon", false, "Run as daemon (background)")
    flag.BoolVar(&cliHeadless, "headless", false, "CLI-only mode")
    flag.IntVar(&cliMaxConcurrent, "max-concurrent", 3, "Max concurrent downloads")
    flag.Int64Var(&cliBandwidth, "bandwidth", 0, "Bandwidth limit (bytes/sec)")
    flag.StringVar(&cliScheduleFile, "schedule", "", "Schedule file (JSON)")
}

// Example: ./downloader -url "http://..." -o "file.zip" -retries 5 -bandwidth 1000000
```

**Files Modified**: `main.go`  
**Testing**: Test various flag combinations

---

### 4.2: Add Status Output Formats

```go
type OutputFormat string

const (
    FormatText   OutputFormat = "text"
    FormatJSON   OutputFormat = "json"
    FormatCSV    OutputFormat = "csv"
)

func outputDownloads(downloads []*Download, format OutputFormat) {
    switch format {
    case FormatJSON:
        json.NewEncoder(os.Stdout).Encode(downloads)
    case FormatCSV:
        fmt.Println("ID,URL,Status,Progress,Total")
        for _, dl := range downloads {
            fmt.Printf("%s,%s,%s,%d/%d\n",
                dl.ID, dl.URL, dl.Status, dl.Progress, dl.Total)
        }
    default:
        for _, dl := range downloads {
            fmt.Printf("[%s] %s: %d%%\n", dl.Status, dl.URL, percentage(dl))
        }
    }
}
```

**Files Modified**: `main.go`

---

## Task 5: Webhook Notifications

**Duration**: 2-3 hours  
**Status**: ⬜ Not Started

### 5.1: Create `webhook.go`

```go
type Webhook struct {
    ID      string
    URL     string
    Events  []string  // "download.started", "download.completed", "download.failed"
    Headers map[string]string
    Active  bool
}

type WebhookManager struct {
    webhooks map[string]*Webhook
    mu       sync.RWMutex
    dm       *DownloadManager
}

func NewWebhookManager(dm *DownloadManager) *WebhookManager {
    wm := &WebhookManager{
        webhooks: make(map[string]*Webhook),
        dm:       dm,
    }

    // Listen to download events
    dm.OnDownloadStart(func(dl *Download) {
        wm.triggerWebhooks("download.started", dl)
    })

    dm.OnDownloadComplete(func(dl *Download) {
        wm.triggerWebhooks("download.completed", dl)
    })

    dm.OnDownloadError(func(dl *Download) {
        wm.triggerWebhooks("download.failed", dl)
    })

    return wm
}

func (wm *WebhookManager) triggerWebhooks(event string, dl *Download) {
    wm.mu.RLock()
    webhooks := wm.webhooks
    wm.mu.RUnlock()

    payload := map[string]interface{}{
        "event":      event,
        "timestamp":  time.Now(),
        "download":   dl,
    }

    body, _ := json.Marshal(payload)

    for _, hook := range webhooks {
        if !contains(hook.Events, event) || !hook.Active {
            continue
        }

        // Non-blocking send
        go func(hook *Webhook) {
            req, _ := http.NewRequest("POST", hook.URL, bytes.NewReader(body))
            req.Header.Set("Content-Type", "application/json")

            for k, v := range hook.Headers {
                req.Header.Set(k, v)
            }

            http.DefaultClient.Do(req)
        }(hook)
    }
}

func (wm *WebhookManager) AddWebhook(webhook *Webhook) {
    wm.mu.Lock()
    defer wm.mu.Unlock()

    if webhook.ID == "" {
        webhook.ID = generateID()
    }

    wm.webhooks[webhook.ID] = webhook
}

func (wm *WebhookManager) TestWebhook(id string) error {
    wm.mu.RLock()
    webhook := wm.webhooks[id]
    wm.mu.RUnlock()

    if webhook == nil {
        return fmt.Errorf("webhook not found")
    }

    payload := map[string]interface{}{
        "event": "test",
        "timestamp": time.Now(),
    }

    body, _ := json.Marshal(payload)

    req, _ := http.NewRequest("POST", webhook.URL, bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return err
    }

    defer resp.Body.Close()

    if resp.StatusCode >= 400 {
        return fmt.Errorf("webhook returned %d", resp.StatusCode)
    }

    return nil
}
```

**Files Created**: `internal/webhook/webhook.go`  
**Testing**: Create webhook, trigger download, verify payload received

---

### 5.2: Add Webhook API

```go
// POST /api/webhooks
func handleAddWebhook(w http.ResponseWriter, r *http.Request) {
    var webhook Webhook
    json.NewDecoder(r.Body).Decode(&webhook)
    webhookManager.AddWebhook(&webhook)
    json.NewEncoder(w).Encode(webhook)
}

// GET /api/webhooks
func handleListWebhooks(w http.ResponseWriter, r *http.Request) {
    webhooks := webhookManager.ListWebhooks()
    json.NewEncoder(w).Encode(webhooks)
}

// POST /api/webhooks/{id}/test
func handleTestWebhook(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    err := webhookManager.TestWebhook(id)
    if err != nil {
        http.Error(w, err.Error(), 400)
        return
    }
    w.WriteHeader(http.StatusOK)
}
```

**Files Modified**: `main.go`

---

## Task 6: Docker Support

**Duration**: 1-2 hours  
**Status**: ⬜ Not Started

### 6.1: Create Multi-stage Dockerfile

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.* .
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o downloader .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/downloader .
COPY templates/ templates/
COPY config/ config/

EXPOSE 8080

VOLUME ["/app/downloads"]

CMD ["./downloader", "-port", "8080"]
```

**Files Created**: `Dockerfile`  
**Testing**: Build image, run container, verify downloads work

---

### 6.2: Create docker-compose.yml

```yaml
version: "3.8"

services:
  downloader:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - downloads:/app/downloads
      - ./config:/app/config
    environment:
      - DOWNLOAD_DIR=/app/downloads
      - PORT=8080
    restart: unless-stopped

volumes:
  downloads:
```

**Files Created**: `docker-compose.yml`  
**Testing**: `docker-compose up`, verify web UI accessible

---

## Task 7: Metrics & Monitoring

**Duration**: 2-3 hours  
**Status**: ⬜ Not Started

### 7.1: Create `metrics.go`

```go
type Metrics struct {
    StartTime             time.Time
    TotalDownloads        int64
    CompletedDownloads    int64
    FailedDownloads       int64
    TotalBytesDownloaded  int64
    AverageSpeed          int64
    TotalRuntime          time.Duration
    ActiveDownloads       int
    QueuedDownloads       int
}

type MetricsCollector struct {
    metrics map[string]interface{}
    mu      sync.RWMutex
    dm      *DownloadManager
}

func NewMetricsCollector(dm *DownloadManager) *MetricsCollector {
    mc := &MetricsCollector{
        metrics: make(map[string]interface{}),
        dm:      dm,
    }

    // Collect metrics every minute
    ticker := time.NewTicker(time.Minute)
    go func() {
        for range ticker.C {
            mc.collect()
        }
    }()

    return mc
}

func (mc *MetricsCollector) collect() {
    mc.mu.Lock()
    defer mc.mu.Unlock()

    downloads := mc.dm.ListDownloads()

    metrics := Metrics{
        StartTime:        time.Now(),
        ActiveDownloads:  len(mc.dm.Running()),
        QueuedDownloads:  len(mc.dm.Queue()),
    }

    for _, dl := range downloads {
        metrics.TotalDownloads++
        metrics.TotalBytesDownloaded += dl.Progress

        switch dl.Status {
        case "completed":
            metrics.CompletedDownloads++
        case "error":
            metrics.FailedDownloads++
        }
    }

    if metrics.TotalDownloads > 0 {
        metrics.AverageSpeed = metrics.TotalBytesDownloaded / int64(len(downloads))
    }

    mc.metrics["current"] = metrics
}

func (mc *MetricsCollector) GetMetrics() Metrics {
    mc.mu.RLock()
    defer mc.mu.RUnlock()

    if m, ok := mc.metrics["current"]; ok {
        return m.(Metrics)
    }
    return Metrics{}
}
```

**Files Created**: `internal/metrics/metrics.go`  
**Testing**: Collect metrics, verify accuracy

---

### 7.2: Add Metrics API

```go
// GET /api/metrics
func handleMetrics(w http.ResponseWriter, r *http.Request) {
    metrics := metricsCollector.GetMetrics()
    json.NewEncoder(w).Encode(metrics)
}

// GET /api/health
func handleHealth(w http.ResponseWriter, r *http.Request) {
    health := map[string]interface{}{
        "status": "healthy",
        "timestamp": time.Now(),
        "uptime": time.Since(startTime),
        "downloads_active": len(downloadManager.Running()),
    }
    json.NewEncoder(w).Encode(health)
}
```

**Files Modified**: `main.go`  
**Testing**: Check `/api/metrics` endpoint

---

## Task 8: Documentation & Examples

**Duration**: 1-2 hours  
**Status**: ⬜ Not Started

### 8.1: Create `AUTOMATION.md`

Document all automation features with examples:

````markdown
# Automation & Scheduling Guide

## Scheduling Downloads

Schedule downloads to run at specific times:

### Simple Schedule

- Name: "Daily Backup"
- URL: "https://example.com/backup.zip"
- Cron: "0 2 \* \* \*" (2 AM daily)

### Cron Expressions

- "0 2 \* \* \*" = Daily at 2 AM
- "0 _/6 _ \* \*" = Every 6 hours
- "0 0 \* \* 0" = Weekly on Sunday
- "0 0 1 \* \*" = Monthly on 1st

## Mirror Fallback

Automatically try alternative sources:

```json
{
  "url": "https://primary.com/file.zip",
  "mirrors": [
    { "url": "https://mirror1.com/file.zip", "priority": 9 },
    { "url": "https://mirror2.com/file.zip", "priority": 8 }
  ]
}
```
````

## Webhooks

Get notifications when downloads complete:

```bash
curl -X POST http://localhost:8080/api/webhooks \
  -d '{
    "url": "https://your-server.com/webhook",
    "events": ["download.completed", "download.failed"],
    "headers": {"Authorization": "Bearer token"}
  }'
```

## Batch Operations

Import multiple URLs at once:

```bash
cat << EOF | ./downloader -headless -
https://example.com/file1.zip
https://example.com/file2.zip
https://example.com/file3.zip
EOF
```

## API Examples

### List Schedules

```bash
curl http://localhost:8080/api/schedules
```

### Add Schedule

```bash
curl -X POST http://localhost:8080/api/schedules \
  -d '{
    "name": "Nightly",
    "url": "https://example.com/data.zip",
    "cronExpr": "0 2 * * *"
  }'
```

### Test Webhook

```bash
curl -X POST http://localhost:8080/api/webhooks/{id}/test
```

### Get Metrics

```bash
curl http://localhost:8080/api/metrics
```

````

**Files Created**: `docs/AUTOMATION.md`

---

### 8.2: Create Example Scripts

**`examples/daily-backup.sh`:**

```bash
#!/bin/bash

URL="https://example.com/backup.zip"
OUTPUT_DIR="./backups"
DATE=$(date +%Y%m%d)

mkdir -p "$OUTPUT_DIR"

curl -X POST http://localhost:8080/api/downloads \
  -H "Content-Type: application/json" \
  -d "{
    \"url\": \"$URL\",
    \"filename\": \"backup-$DATE.zip\"
  }"

echo "Download queued: backup-$DATE.zip"
````

**`examples/webhook-receiver.py`:**

```python
#!/usr/bin/env python3
from flask import Flask, request
import json

app = Flask(__name__)

@app.route('/webhook', methods=['POST'])
def webhook():
    data = request.json
    print(f"Download {data['event']}: {data['download']['URL']}")
    return {'status': 'ok'}

if __name__ == '__main__':
    app.run(port=5000)
```

**Files Created**: `examples/` directory with scripts  
**Testing**: Run examples, verify functionality

---

## Summary

| Task                     | Duration   | Status |
| ------------------------ | ---------- | ------ |
| Download Scheduling      | 2-3h       | ⬜     |
| Mirror Fallback Support  | 2-3h       | ⬜     |
| Batch Import             | 1-2h       | ⬜     |
| CLI Improvements         | 2-3h       | ⬜     |
| Webhook Notifications    | 2-3h       | ⬜     |
| Docker Support           | 1-2h       | ⬜     |
| Metrics & Monitoring     | 2-3h       | ⬜     |
| Documentation & Examples | 1-2h       | ⬜     |
| **Total**                | **16-23h** |        |

**Recommended**: 2-3 weeks of optional work for enterprise/advanced users.

---

## Success Criteria

✅ Downloads can be scheduled with cron expressions  
✅ Multiple mirrors fallback on primary failure  
✅ Batch import handles multiple URLs  
✅ CLI supports advanced flags and output formats  
✅ Webhooks notify external systems  
✅ Docker container builds and runs  
✅ Metrics API provides statistics  
✅ Examples demonstrate all features

---

## Integration Notes

- Builds on Phase 4 (Polish) completion
- All features optional (enterprise-focused)
- Each feature can be implemented independently
- No breaking changes to existing API
- Backward compatible with Phase 1-4 features

---

## Next Steps

1. Choose 2-3 features to implement first
2. Implement scheduling + webhooks (most requested)
3. Add Docker support for deployment
4. Document with examples
5. Release as optional enterprise features

---

## Links

- See `PHASE1_MVP.md` for MVP
- See `PHASE2_MANAGER.md` for manager
- See `PHASE3_ADVANCED.md` for advanced features
- See `PHASE4_POLISH.md` for polish
- See `AUTOMATION.md` for detailed guide

---

**Enterprise features for power users!** 🚀
