# Phase 3: Advanced Features (Optional)

**Duration**: 1-2 weeks (as time allows)  
**Tasks**: 10  
**Effort**: Medium  
**Prerequisite**: Phase 2 Manager Complete  
**Status**: Not Started

## Overview

These are power-user features that make the app competitive with commercial download managers like IDM or FDM. Implement only features you have time for.

---

## Feature 1: Bandwidth Limiting

**Duration**: 2-3 hours  
**Priority**: Medium  
**Complexity**: Medium

### Overview

Limit download speed per file or globally to prevent network congestion.

### Implementation

**Backend (`download_manager.go`):**

```go
type DownloadManager struct {
    // ... existing fields ...
    maxBandwidth    int64  // bytes/sec, 0 = unlimited
    perDownloadCap  int64  // cap per download
}

// Limit speed by throttling write
func (dm *DownloadManager) throttleWrite(w io.Writer, maxSpeed int64) io.Writer {
    return &throttledWriter{
        w:       w,
        maxBps:  maxSpeed,
        start:   time.Now(),
    }
}

type throttledWriter struct {
    w       io.Writer
    maxBps  int64
    written int64
    start   time.Time
}

func (tw *throttledWriter) Write(p []byte) (n int, err error) {
    // Calculate sleep needed to maintain maxBps
    elapsed := time.Since(tw.start).Seconds()
    expectedBytes := int64(elapsed * float64(tw.maxBps))

    if tw.written >= expectedBytes {
        // Too fast, sleep
        excess := tw.written - expectedBytes
        sleepDuration := time.Duration(excess/int64(tw.maxBps)) * time.Second
        time.Sleep(sleepDuration)
    }

    n, err = tw.w.Write(p)
    tw.written += int64(n)
    return
}
```

**API Endpoints:**

```
POST /api/settings/bandwidth
  - Set global bandwidth limit (KB/s)
  - Set per-download limit

GET /api/settings/bandwidth
  - Get current limits
```

**UI (`templates/index.html`):**

```html
<div class="settings-section">
  <label>محدودیت پهنای باند:</label>

  <!-- Global limit -->
  <label>
    <input
      type="checkbox"
      id="bandwidth-enabled"
      onchange="toggleBandwidth()"
    />
    فعال کنید
  </label>

  <div id="bandwidth-controls" class="hidden">
    <label>سرعت حداکثر (KB/s):</label>
    <input
      type="number"
      id="bandwidth-limit"
      min="1"
      value="1024"
      onchange="setBandwidthLimit(this.value)"
    />

    <p class="info">مثال: 1024 KB/s = 1 MB/s</p>
  </div>
</div>

<script>
  function setBandwidthLimit(limit) {
    fetch("/api/settings/bandwidth", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ limit: limit }),
    });
  }
</script>
```

**Testing:**

- [ ] Set limit to 1 MB/s
- [ ] Download file
- [ ] Monitor speed, should stabilize at ~1 MB/s
- [ ] Change limit to 2 MB/s
- [ ] Speed should increase

---

## Feature 2: Custom Headers

**Duration**: 2-3 hours  
**Priority**: Medium  
**Complexity**: Easy

### Overview

Support custom HTTP headers for authentication, custom User-Agent, cookies, etc.

### Implementation

**Backend (`download_generic.go`):**

```go
func downloadFileWithHeaders(ctx context.Context, downloadURL, outputPath string,
    headers map[string]string, p *progress) error {

    req, _ := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)

    // Add custom headers
    for key, value := range headers {
        req.Header.Set(key, value)
    }

    // Default headers
    if _, ok := headers["User-Agent"]; !ok {
        req.Header.Set("User-Agent", "Download-Manager/1.0")
    }

    // ... rest of download logic
}
```

**Store headers in session:**

```go
type sessionMeta struct {
    // ... existing ...
    CustomHeaders map[string]string `json:"customHeaders"`
}
```

**API Endpoint:**

```
POST /api/download/add
  url: "https://..."
  headers: {
    "Authorization": "Bearer token",
    "User-Agent": "Custom-Agent"
  }
```

**UI (`templates/index.html`):**

```html
<div class="headers-section">
  <label>سربرگ‌های سفارشی (اختیاری):</label>

  <textarea
    id="custom-headers"
    placeholder="Authorization: Bearer token
User-Agent: Custom-Agent
Cookie: session=xyz"
  ></textarea>

  <button onclick="savePreset()">ذخیره سرنمونه</button>

  <div class="presets">
    <label>پیش‌تعیین‌ها:</label>
    <select id="header-presets" onchange="loadPreset()">
      <option value="">-- انتخاب کنید --</option>
      <option value="auth">احراز هویت</option>
      <option value="custom-ua">User-Agent سفارشی</option>
    </select>
  </div>
</div>
```

**Testing:**

- [ ] Add Authorization header
- [ ] Download from protected URL
- [ ] Verify download succeeds
- [ ] Try invalid header, should error

---

## Feature 3: Batch URL Import

**Duration**: 2-3 hours  
**Priority**: Medium  
**Complexity**: Easy

### Overview

Paste multiple URLs at once, one per line, and add them all to queue.

### Implementation

**Backend (`main.go`):**

```go
http.HandleFunc("/api/downloads/batch", func(w http.ResponseWriter, r *http.Request) {
    urls := r.FormValue("urls")  // newline-separated

    lines := strings.Split(urls, "\n")
    added := []string{}
    errors := []string{}

    for i, line := range lines {
        line = strings.TrimSpace(line)
        if line == "" || strings.HasPrefix(line, "#") {
            continue  // Skip empty/comments
        }

        if err := validateURL(line); err != nil {
            errors = append(errors, fmt.Sprintf("Line %d: %v", i, err))
            continue
        }

        id := downloadMgr.AddDownload(line, extractFilenameFromURL(line), "", 3)
        added = append(added, id)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "added":  len(added),
        "errors": errors,
    })
})
```

**UI (`templates/index.html`):**

```html
<div class="batch-import">
  <h3>دانلود چندگانه:</h3>

  <textarea
    id="batch-urls"
    placeholder="https://example.com/file1.zip
https://example.com/file2.iso
https://example.com/file3.tar.gz"
    rows="8"
  ></textarea>

  <div class="batch-preview" id="batch-preview"></div>

  <button onclick="previewBatch()">پیش‌نمایش</button>
  <button onclick="importBatch()">افزودن همه</button>
</div>

<script>
  async function previewBatch() {
    const urls = document.getElementById("batch-urls").value.split("\n");
    const valid = urls.filter((u) => u.trim() && !u.startsWith("#"));

    const preview = document.getElementById("batch-preview");
    preview.innerHTML = `<p>${valid.length} URL برای دانلود</p>`;

    const list = document.createElement("ul");
    for (const url of valid) {
      const li = document.createElement("li");
      li.textContent = url.trim();
      list.appendChild(li);
    }
    preview.appendChild(list);
  }

  async function importBatch() {
    const urls = document.getElementById("batch-urls").value;
    const res = await fetch("/api/downloads/batch", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ urls }),
    });

    const data = await res.json();
    alert(`${data.added} اضافه شد, ${data.errors.length} خطا`);
    fetchDownloads();
  }
</script>
```

**Testing:**

- [ ] Paste 5 URLs into textarea
- [ ] Click preview, shows 5 URLs
- [ ] Click import, all added to queue
- [ ] Mix valid and invalid, shows errors
- [ ] Empty lines and comments ignored

---

## Feature 4: Drag & Drop

**Duration**: 2-3 hours  
**Priority**: Low  
**Complexity**: Easy

### Overview

Drag URLs or files into the app to start downloads.

### Implementation

**UI (`templates/index.html`):**

```html
<div class="drop-zone" id="drop-zone">
  <p>فایل یا URL را اینجا رها کنید</p>
</div>

<script>
  const dropZone = document.getElementById("drop-zone");

  dropZone.addEventListener("dragover", (e) => {
    e.preventDefault();
    dropZone.classList.add("drag-over");
  });

  dropZone.addEventListener("dragleave", () => {
    dropZone.classList.remove("drag-over");
  });

  dropZone.addEventListener("drop", async (e) => {
    e.preventDefault();
    dropZone.classList.remove("drag-over");

    const text = e.dataTransfer.getData("text/plain");
    const files = e.dataTransfer.files;

    if (text) {
      // Dropped text - assume it's a URL
      const url = text.trim();
      if (url.startsWith("http")) {
        downloadMgr.addDownload(url);
      }
    }

    if (files.length > 0) {
      // Dropped files - create local file URLs
      for (const file of files) {
        const url = URL.createObjectURL(file);
        // ... handle local file download
      }
    }
  });
</script>
```

**CSS:**

```css
.drop-zone {
  border: 2px dashed #ccc;
  border-radius: 8px;
  padding: 20px;
  text-align: center;
  cursor: pointer;
  transition: all 0.3s;
}

.drop-zone.drag-over {
  border-color: #2196f3;
  background: #e3f2fd;
}
```

**Testing:**

- [ ] Drag URL from browser address bar into drop zone
- [ ] Download starts
- [ ] Drag local file
- [ ] Creates download from file

---

## Feature 5: Settings Panel

**Duration**: 3-4 hours  
**Priority**: Medium  
**Complexity**: Medium

### Overview

Comprehensive settings for download folder, performance, behavior, appearance.

### Implementation

**Create settings struct:**

```go
type Settings struct {
    DownloadFolder      string `json:"downloadFolder"`
    MaxConcurrent       int    `json:"maxConcurrent"`
    BandwidthLimit      int    `json:"bandwidthLimit"`
    AutoStart           bool   `json:"autoStart"`
    ClearCompleted      int    `json:"clearCompleted"` // hours, 0 = manual
    Theme               string `json:"theme"`           // light, dark, auto
    NotificationsEnabled bool `json:"notificationsEnabled"`
    SoundEnabled        bool   `json:"soundEnabled"`
    KeepPartFiles       bool   `json:"keepPartFiles"`
}
```

**Persistent storage:**

```go
func (dm *DownloadManager) LoadSettings() error {
    data, _ := os.ReadFile(".settings.json")
    return json.Unmarshal(data, &dm.Settings)
}

func (dm *DownloadManager) SaveSettings() error {
    data, _ := json.MarshalIndent(dm.Settings, "", "  ")
    return os.WriteFile(".settings.json", data, 0644)
}
```

**API Endpoints:**

```
GET /api/settings - Get all settings
POST /api/settings - Save settings
POST /api/settings/reset - Reset to defaults
GET /api/settings/download-folder - Current folder
POST /api/settings/download-folder - Browse and change
```

**UI (`templates/index.html`):**

```html
<div id="settings-modal" class="modal hidden">
  <div class="modal-content">
    <h2>تنظیمات</h2>

    <!-- Tabs in settings -->
    <div class="settings-tabs">
      <button onclick="showSettingsTab('paths')">مسیرها</button>
      <button onclick="showSettingsTab('performance')">کارایی</button>
      <button onclick="showSettingsTab('behavior')">رفتار</button>
      <button onclick="showSettingsTab('appearance')">ظاهر</button>
      <button onclick="showSettingsTab('advanced')">پیشرفته</button>
    </div>

    <!-- Paths tab -->
    <div id="tab-paths" class="settings-tab">
      <label>پوشه دانلود:</label>
      <div class="folder-picker">
        <input type="text" id="download-folder" readonly />
        <button onclick="browseFolder()">مرور</button>
      </div>
    </div>

    <!-- Performance tab -->
    <div id="tab-performance" class="settings-tab">
      <label>دانلود‌های موازی حداکثر:</label>
      <input
        type="range"
        id="max-concurrent"
        min="1"
        max="32"
        value="4"
        oninput="updateConcurrent(this.value)"
      />
      <span id="concurrent-value">4</span>

      <label>محدودیت پهنای باند (KB/s):</label>
      <input
        type="number"
        id="bandwidth-limit"
        value="0"
        placeholder="0 = نامحدود"
      />
    </div>

    <!-- Behavior tab -->
    <div id="tab-behavior" class="settings-tab">
      <label>
        <input type="checkbox" id="auto-start" />
        شروع خودکار دانلود‌ها
      </label>

      <label>پاک‌کردن خودکار انجام‌شده:</label>
      <select id="clear-completed">
        <option value="0">دستی</option>
        <option value="1">بعد از 1 ساعت</option>
        <option value="24">بعد از 1 روز</option>
      </select>
    </div>

    <!-- Appearance tab -->
    <div id="tab-appearance" class="settings-tab">
      <label>تم:</label>
      <select id="theme">
        <option value="auto">خودکار</option>
        <option value="light">روشن</option>
        <option value="dark">تاریک</option>
      </select>
    </div>

    <!-- Advanced tab -->
    <div id="tab-advanced" class="settings-tab">
      <label>
        <input type="checkbox" id="keep-part-files" />
        نگاه‌داشتن فایل‌های .part بعد از دانلود
      </label>

      <label>سطح گزارش:</label>
      <select id="log-level">
        <option value="info">اطلاعات</option>
        <option value="debug">اشکال‌زدایی</option>
        <option value="error">خطا</option>
      </select>
    </div>

    <div class="modal-actions">
      <button onclick="closeSettings()">انصراف</button>
      <button onclick="saveSettings()" class="primary">ذخیره</button>
      <button onclick="resetSettings()">بازنشانی</button>
    </div>
  </div>
</div>
```

**Testing:**

- [ ] Open settings modal
- [ ] Change max concurrent from 4 to 8
- [ ] Restart app
- [ ] Verify 8 downloads now concurrent
- [ ] Change download folder
- [ ] New downloads go to new folder
- [ ] Change theme
- [ ] UI colors change

---

## Feature 6: Advanced Search & Filter

**Duration**: 2-3 hours  
**Priority**: Low  
**Complexity**: Medium

### Overview

Filter downloads by file type, date range, size range, status, domain.

### Implementation

**Filter UI (`templates/index.html`):**

```html
<div class="filter-panel" id="filter-panel">
  <h3>فیلتر‌ها:</h3>

  <label>وضعیت:</label>
  <div class="filter-chips">
    <button class="chip active" onclick="filterByStatus('all')">همه</button>
    <button class="chip" onclick="filterByStatus('completed')">
      انجام‌شده
    </button>
    <button class="chip" onclick="filterByStatus('error')">خطا</button>
  </div>

  <label>نوع فایل:</label>
  <div class="filter-checkboxes">
    <label><input type="checkbox" onchange="filterByType()" /> .zip</label>
    <label><input type="checkbox" onchange="filterByType()" /> .iso</label>
    <label><input type="checkbox" onchange="filterByType()" /> .tar</label>
    <label><input type="checkbox" onchange="filterByType()" /> .rar</label>
  </div>

  <label>اندازه:</label>
  <div class="size-range">
    <input
      type="range"
      id="size-min"
      min="0"
      max="10"
      onchange="filterBySize()"
    />
    <input
      type="range"
      id="size-max"
      min="0"
      max="10"
      onchange="filterBySize()"
    />
  </div>

  <label>دامنه:</label>
  <input
    type="text"
    placeholder="github.com"
    onkeyup="filterByDomain()"
    id="domain-filter"
  />

  <label>دامنه زمانی:</label>
  <div class="date-range">
    <input type="date" id="date-from" onchange="filterByDate()" />
    <input type="date" id="date-to" onchange="filterByDate()" />
  </div>

  <div class="filter-actions">
    <button onclick="saveFilter()">ذخیره فیلتر</button>
    <button onclick="resetFilter()">بازنشانی</button>
  </div>
</div>
```

**Filter logic:**

```js
let filters = {
  status: "all",
  types: [],
  sizeMin: 0,
  sizeMax: Infinity,
  domain: "",
  dateFrom: null,
  dateTo: null,
};

function applyFilters(downloads) {
  return downloads.filter((dl) => {
    // Status filter
    if (filters.status !== "all" && dl.status !== filters.status) {
      return false;
    }

    // Type filter
    if (filters.types.length > 0) {
      const ext = getExtension(dl.filename);
      if (!filters.types.includes(ext)) return false;
    }

    // Size filter
    if (dl.total < filters.sizeMin || dl.total > filters.sizeMax) {
      return false;
    }

    // Domain filter
    if (filters.domain && !dl.url.includes(filters.domain)) {
      return false;
    }

    // Date filter
    if (filters.dateFrom && dl.startTime < filters.dateFrom) {
      return false;
    }
    if (filters.dateTo && dl.startTime > filters.dateTo) {
      return false;
    }

    return true;
  });
}
```

**Testing:**

- [ ] Filter by .zip files only
- [ ] See only .zip in results
- [ ] Filter by github.com
- [ ] See only github downloads
- [ ] Combine filters
- [ ] Save filter preset
- [ ] Reset filter

---

## Feature 7: Keyboard Shortcuts

**Duration**: 1-2 hours  
**Priority**: Low  
**Complexity**: Easy

### Overview

Common keyboard shortcuts for power users.

### Implementation

```js
document.addEventListener("keydown", (e) => {
  // Ctrl/Cmd + N: New download
  if ((e.ctrlKey || e.metaKey) && e.key === "n") {
    e.preventDefault();
    focusUrlInput();
  }

  // Ctrl/Cmd + A: Select all
  if ((e.ctrlKey || e.metaKey) && e.key === "a") {
    e.preventDefault();
    selectAll();
  }

  // Ctrl/Cmd + D: Delete selected
  if ((e.ctrlKey || e.metaKey) && e.key === "d") {
    e.preventDefault();
    deleteSelected();
  }

  // Ctrl/Cmd + P: Pause/Resume selected
  if ((e.ctrlKey || e.metaKey) && e.key === "p") {
    e.preventDefault();
    togglePauseSelected();
  }

  // Ctrl/Cmd + Shift + P: Pause all
  if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === "P") {
    e.preventDefault();
    pauseAll();
  }

  // Ctrl/Cmd + Shift + R: Resume all
  if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === "R") {
    e.preventDefault();
    resumeAll();
  }

  // Ctrl/Cmd + F: Focus search
  if ((e.ctrlKey || e.metaKey) && e.key === "f") {
    e.preventDefault();
    focusSearch();
  }

  // Ctrl/Cmd + S: Settings
  if ((e.ctrlKey || e.metaKey) && e.key === "s") {
    e.preventDefault();
    openSettings();
  }

  // Del: Delete selected
  if (e.key === "Delete") {
    deleteSelected();
  }
});
```

**Display help:**

```html
<div class="help-modal" id="help-modal">
  <h2>میانبرهای صفحه‌کلید</h2>
  <table>
    <tr>
      <td>Ctrl/Cmd + N</td>
      <td>دانلود جدید</td>
    </tr>
    <tr>
      <td>Ctrl/Cmd + A</td>
      <td>انتخاب همه</td>
    </tr>
    <tr>
      <td>Ctrl/Cmd + D</td>
      <td>حذف انتخاب‌شده</td>
    </tr>
    <tr>
      <td>Ctrl/Cmd + P</td>
      <td>توقف/ادامه انتخاب‌شده</td>
    </tr>
    <tr>
      <td>Del</td>
      <td>حذف انتخاب‌شده</td>
    </tr>
  </table>
</div>

<button onclick="showHelp()">?</button>
```

**Testing:**

- [ ] Press Ctrl+N, URL input focused
- [ ] Press Ctrl+A, all downloads selected
- [ ] Press Ctrl+D, selected deleted
- [ ] Press Ctrl+P, selected paused

---

## Feature 8: Dark/Light Theme

**Duration**: 2-3 hours  
**Priority**: Low  
**Complexity**: Medium

### Overview

User preference for dark or light theme, with auto-detection.

### Implementation

**CSS Variables:**

```css
:root {
  --bg-primary: #ffffff;
  --bg-secondary: #f5f5f5;
  --text-primary: #000000;
  --text-secondary: #666666;
  --border-color: #dddddd;
  --accent-color: #2196f3;
}

[data-theme="dark"] {
  --bg-primary: #1e1e1e;
  --bg-secondary: #2d2d2d;
  --text-primary: #ffffff;
  --text-secondary: #cccccc;
  --border-color: #444444;
  --accent-color: #64b5f6;
}
```

**Theme toggle:**

```js
function setTheme(theme) {
  document.documentElement.setAttribute("data-theme", theme);
  localStorage.setItem("theme", theme);
  updateSettings({ theme });
}

function initTheme() {
  const saved = localStorage.getItem("theme");
  if (saved) {
    setTheme(saved);
  } else if (window.matchMedia("(prefers-color-scheme: dark)").matches) {
    setTheme("dark");
  } else {
    setTheme("light");
  }
}

// Listen for system theme changes
window
  .matchMedia("(prefers-color-scheme: dark)")
  .addEventListener("change", (e) => {
    if (localStorage.getItem("theme") === "auto") {
      setTheme(e.matches ? "dark" : "light");
    }
  });
```

**Testing:**

- [ ] Toggle theme
- [ ] All colors change
- [ ] Setting persists after restart
- [ ] System dark mode detected

---

## Feature 9: Statistics Dashboard

**Duration**: 3-4 hours  
**Priority**: Low  
**Complexity**: High

### Overview

Visual dashboard with charts and metrics.

### Implementation

**Requires Chart.js library:**

```html
<script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
```

**Dashboard UI:**

```html
<div id="statistics-dashboard">
  <div class="stats-grid">
    <!-- Summary cards -->
    <div class="stat-card">
      <h3>کل دانلود</h3>
      <p class="stat-value" id="stat-total-files">0</p>
      <p class="stat-label">فایل</p>
    </div>

    <div class="stat-card">
      <h3>کل داده</h3>
      <p class="stat-value" id="stat-total-bytes">0</p>
      <p class="stat-label">GB</p>
    </div>

    <div class="stat-card">
      <h3>سرعت متوسط</h3>
      <p class="stat-value" id="stat-avg-speed">0</p>
      <p class="stat-label">MB/s</p>
    </div>

    <!-- Charts -->
    <div class="chart-container">
      <h3>فعالیت این هفته</h3>
      <canvas id="activity-chart"></canvas>
    </div>

    <div class="chart-container">
      <h3>دامنه‌های برتر</h3>
      <canvas id="domains-chart"></canvas>
    </div>

    <div class="chart-container">
      <h3>انواع فایل</h3>
      <canvas id="types-chart"></canvas>
    </div>
  </div>
</div>
```

**Chart rendering:**

```js
function renderActivityChart(stats) {
  const ctx = document.getElementById("activity-chart").getContext("2d");
  new Chart(ctx, {
    type: "bar",
    data: {
      labels: ["ش", "ی", "د", "س", "چ", "پ", "ج"],
      datasets: [
        {
          label: "MB دانلود شده",
          data: stats.dailyActivity,
          backgroundColor: "#2196F3",
        },
      ],
    },
  });
}

function renderDomainsChart(topDomains) {
  const ctx = document.getElementById("domains-chart").getContext("2d");
  new Chart(ctx, {
    type: "pie",
    data: {
      labels: Object.keys(topDomains),
      datasets: [
        {
          data: Object.values(topDomains),
          backgroundColor: [
            "#FF6384",
            "#36A2EB",
            "#FFCE56",
            "#4BC0C0",
            "#9966FF",
          ],
        },
      ],
    },
  });
}
```

**Testing:**

- [ ] Complete several downloads
- [ ] Dashboard shows correct totals
- [ ] Charts display data
- [ ] Charts update in real-time

---

## Feature 10: Desktop Notifications

**Duration**: 1-2 hours  
**Priority**: Low  
**Complexity**: Easy

### Overview

Notify user when downloads complete (with browser permission).

### Implementation

```js
async function requestNotificationPermission() {
  if ("Notification" in window && Notification.permission === "default") {
    const permission = await Notification.requestPermission();
    return permission === "granted";
  }
  return Notification.permission === "granted";
}

function notifyDownloadComplete(filename) {
  if (Notification.permission === "granted") {
    new Notification("دانلود انجام شد", {
      body: `${filename} دانلود شد`,
      icon: "/icon.png",
      tag: "download-complete",
    });

    // Play sound if enabled
    if (settings.soundEnabled) {
      playSound("complete.mp3");
    }
  }
}

function notifyDownloadError(filename, error) {
  if (Notification.permission === "granted") {
    new Notification("خطا در دانلود", {
      body: `${filename}: ${error}`,
      icon: "/icon-error.png",
      tag: "download-error",
    });
  }
}
```

**UI Setting:**

```html
<label>
  <input
    type="checkbox"
    id="notifications-enabled"
    onchange="toggleNotifications()"
  />
  اعلان برای دانلود‌های انجام‌شده
</label>

<label>
  <input
    type="checkbox"
    id="sound-enabled"
    onchange="updateSetting('soundEnabled')"
  />
  صدا برای دانلود‌های انجام‌شده
</label>
```

**Testing:**

- [ ] Allow notifications in browser
- [ ] Complete a download
- [ ] Browser notification appears
- [ ] Check "sound enabled"
- [ ] Next completion plays sound

---

## Summary

### Quick Implementation Guide

| Feature              | Time | Difficulty | Priority |
| -------------------- | ---- | ---------- | -------- |
| Bandwidth Limiting   | 2-3h | Medium     | High     |
| Custom Headers       | 2-3h | Easy       | Medium   |
| Batch Import         | 2-3h | Easy       | High     |
| Drag & Drop          | 2-3h | Easy       | Low      |
| Settings Panel       | 3-4h | Medium     | High     |
| Advanced Search      | 2-3h | Medium     | Low      |
| Keyboard Shortcuts   | 1-2h | Easy       | Low      |
| Dark/Light Theme     | 2-3h | Medium     | Medium   |
| Statistics Dashboard | 3-4h | High       | Low      |
| Notifications        | 1-2h | Easy       | Low      |

### Quick Wins (do first)

1. Batch URL Import (2-3h, very useful)
2. Keyboard Shortcuts (1-2h, easy)
3. Settings Panel (3-4h, essential)
4. Bandwidth Limiting (2-3h, useful)

### Polish Features (if time)

5. Dark/Light Theme (2-3h, nice UX)
6. Custom Headers (2-3h, advanced users)
7. Advanced Search (2-3h, power users)
8. Statistics Dashboard (3-4h, impressive)
9. Drag & Drop (2-3h, modern UX)
10. Notifications (1-2h, completion feedback)

---

## Next Steps

Choose 3-4 features to implement from this list based on your priorities and available time.

See `PHASE4_POLISH.md` for final polish and documentation.

Good luck! 🚀
