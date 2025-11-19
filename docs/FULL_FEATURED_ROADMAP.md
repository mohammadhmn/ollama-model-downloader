# Full-Featured Download Manager - Implementation Roadmap

## Vision

Transform Ollama Model Downloader into a professional-grade file download manager with enterprise features.

## Feature Tiers

### 🟢 Tier 1: MVP (Essential Features)

Basic functionality that makes it work as a general downloader.

**Backend:**

- Generic HTTP/HTTPS download
- Resume support (Range headers)
- Simple progress tracking
- Error handling

**UI:**

- URL input field
- Progress bar
- Pause/Resume/Cancel buttons
- File management (delete, open folder)

**Result:** Works like curl/wget with a web UI

---

### 🟡 Tier 2: Download Manager Essentials

Multi-download support and basic queue management.

**Backend:**

- Queue data structure
- Multiple concurrent downloads
- Session persistence for each download
- Speed calculation

**UI:**

- Active/Queue/Completed tabs
- List view of all downloads
- Individual pause/resume controls
- Summary statistics (total files, total size)

**Result:** Functional download manager like FDM lite

---

### 🔴 Tier 3: Professional Features

Advanced features for power users.

**Backend:**

- Bandwidth limiting
- Download history database
- Statistics aggregation
- Advanced scheduling (future)

**UI:**

- Search and advanced filtering
- Bulk operations
- Settings panel
- Download history with metrics
- Statistics dashboard
- Keyboard shortcuts
- Dark/Light theme

**Result:** Competing with commercial download managers (IDM, FDM)

---

### ⚪ Tier 4: Premium Features (Nice-to-have)

Cutting-edge features.

**Planned for Future:**

- Torrent support
- FTP/SFTP protocol
- Authentication (cookies, OAuth)
- Scheduling & automation
- Browser integration
- Mobile app
- Cloud sync
- Browser extensions

---

## Detailed Feature Breakdown

### 2.5.1: Queue Management System

**Data Structures:**

```go
type DownloadQueue struct {
    Downloads    []Download
    Running      int
    MaxConcurrent int
}

type Download struct {
    ID            string
    URL           string
    Filename      string
    Status        string       // active, paused, queued, completed, error
    Priority      int          // 1-10, higher = earlier
    Progress      DownloadProgress
    CreatedAt     time.Time
    StartedAt     time.Time
    CompletedAt   time.Time
    ErrorMsg      string
}

type DownloadProgress struct {
    Downloaded    int64
    Total         int64
    Speed         int64        // bytes per second
    ETA           time.Duration
    StartTime     time.Time
}
```

**Backend APIs:**

- `POST /api/download/add` - Add to queue
- `POST /api/download/{id}/pause` - Pause specific download
- `POST /api/download/{id}/resume` - Resume specific download
- `POST /api/download/{id}/cancel` - Cancel download
- `POST /api/download/pause-all` - Pause all
- `POST /api/download/resume-all` - Resume all
- `DELETE /api/download/{id}` - Delete from queue
- `GET /api/downloads` - Get all downloads
- `GET /api/downloads/{id}/progress` - Get progress

---

### 2.5.2: Speed & ETA Calculation

**Algorithm:**

```
Speed = (CurrentDownloaded - PreviousDownloaded) / TimeDelta
MovingAverage = (Speed * 0.3) + (PreviousSpeed * 0.7)  // Smooth spikes
ETA = (Total - Downloaded) / MovingAverage
```

**Display Format:**

```
File: large-file.zip
↓ 524 MB / 1.2 GB (43%)
Speed: 2.5 MB/s ↓
ETA: 5m 30s remaining
```

---

### 2.5.3: Download History & Persistence

**Storage:**

```
downloaded-files/
├── .history/
│   └── downloads.json          // All downloads metadata
├── .cache/
│   └── statistics.json         // Stats aggregation
└── files/                       // Downloaded files
    ├── document.pdf
    ├── video.mp4
    └── archive.zip
```

**History Entry:**

```json
{
  "id": "dl-20241114-001",
  "url": "https://example.com/file.zip",
  "filename": "file.zip",
  "filesize": 1073741824,
  "downloadedAt": "2024-11-14T10:30:00Z",
  "duration": 120,
  "speed": 8945356,
  "status": "completed",
  "error": null
}
```

---

### 2.5.4: Batch Operations

**UI Elements:**

- Checkboxes on each download card
- "Select All" header checkbox
- Bulk action toolbar (when items selected):
  - Delete Selected
  - Pause Selected
  - Resume Selected
  - Move to Top/Bottom
  - Open Folders

**JavaScript Logic:**

```javascript
// Track selection
selectedDownloads = new Set();

function toggleDownload(id) {
  if (selectedDownloads.has(id)) {
    selectedDownloads.delete(id);
  } else {
    selectedDownloads.add(id);
  }
  updateBulkActionButtons();
}

function bulkPause() {
  selectedDownloads.forEach((id) => {
    fetch(`/api/download/${id}/pause`, { method: "POST" });
  });
}
```

---

### 2.5.5: Advanced Search & Filtering

**Search Fields:**

- Filename
- URL domain
- File type (by extension)
- Status
- Date range
- Size range

**Filter Presets:**

```
[ ] Active Downloads
[ ] Today's Downloads
[ ] Large Files (>100MB)
[ ] Images Only
[ ] Failed Downloads
[ ] By Domain (show dropdown)
```

**Saved Filters:**

- Users can save complex filter combinations
- "Show only downloads from github.com completed today"

---

### 2.5.6: Settings Panel

**Layout:** Side panel or modal dialog

**Sections:**

1. **Paths**

   - Default download folder
   - Browser for folder selection

2. **Performance**

   - Max concurrent downloads (1-32, slider)
   - Bandwidth limit (KB/s, checkbox + input)
   - Connection timeout

3. **Behavior**

   - Auto-start downloads
   - Pause on low disk space
   - Auto-rename on conflict
   - Clear completed after (time selector)

4. **Appearance**

   - Theme (Light/Dark/Auto)
   - Compact view
   - Font size

5. **Notifications**

   - Desktop notifications
   - Sound on completion
   - Show progress tray icon

6. **Advanced**
   - User-Agent
   - SSL verification
   - Keep .part files after download
   - Log level

---

### 2.5.7: Statistics Dashboard

**Widgets:**

```
┌─────────────────────────────────────────┐
│  📊 Download Statistics                 │
├─────────────────────────────────────────┤
│ Total Downloaded: 125 GB                │
│ Files Completed: 547                    │
│ This Session: 2.3 GB (12 files)         │
│ Average Speed: 5.2 MB/s                 │
│ Time Saved: 2h 15m (by pausing)         │
├─────────────────────────────────────────┤
│ 📈 Activity This Week                   │
│ Mon: 234 MB  Tue: 512 MB  Wed: 1.2 GB  │
│ Thu: 456 MB  Fri: 890 MB  Sat: 2.1 GB  │
│ Sun: 340 MB                             │
├─────────────────────────────────────────┤
│ 🔝 Top Domains                          │
│ github.com     →  45.3 GB               │
│ releases.ubuntu.com  →  23.1 GB         │
│ example.com    →  18.9 GB               │
├─────────────────────────────────────────┤
│ 📁 File Types                           │
│ .zip:  45 files (34.2 GB)               │
│ .iso:  12 files (56.7 GB)               │
│ .tar:  89 files (22.1 GB)               │
│ Others: 401 files (11.9 GB)             │
└─────────────────────────────────────────┘
```

---

### 2.5.8: Keyboard Shortcuts

```
Ctrl/Cmd + N    → New download
Ctrl/Cmd + A    → Select all
Ctrl/Cmd + D    → Delete selected
Ctrl/Cmd + P    → Pause/Resume selected
Ctrl/Cmd + Shift + P → Pause all
Ctrl/Cmd + Shift + R → Resume all
Ctrl/Cmd + F    → Focus search
Ctrl/Cmd + S    → Open settings
Ctrl/Cmd + H    → Show history
Del             → Delete selected
Space           → Toggle selection (when focused)
```

---

### 2.5.9: URL Input Enhancements

**Features:**

1. **URL Validation:**

   - Real-time validation as user types
   - Green checkmark for valid URLs
   - Red X for invalid URLs
   - Error message below input

2. **File Preview:**

   - After pasting URL, fetch headers
   - Show filename (from URL or Content-Disposition)
   - Show file size (from Content-Length)
   - Show MIME type

3. **Batch Input:**

   ```
   Allow pasting multiple URLs (one per line)
   - Validate all
   - Show preview of each
   - "Add All" button
   ```

4. **Drag & Drop:**
   ```
   Drop URLs or files into designated area
   - Parse dropped text as URLs
   - Show preview before adding
   ```

---

### 2.5.10: File Conflict Handling

**Scenarios:**

1. **File exists** → Ask user:

   - Overwrite
   - Rename (auto-suggest: file (1).zip)
   - Skip
   - Cancel

2. **Path invalid** → Show error with suggestions

3. **Low disk space** → Warn and ask to continue

**Implementation:**

```go
func getUniqueFilename(path string) string {
    if _, err := os.Stat(path); err != nil {
        return path  // Doesn't exist, use as-is
    }

    ext := filepath.Ext(path)
    base := strings.TrimSuffix(path, ext)

    for i := 1; i <= 999; i++ {
        newPath := fmt.Sprintf("%s (%d)%s", base, i, ext)
        if _, err := os.Stat(newPath); err != nil {
            return newPath
        }
    }
    return path
}
```

---

## Tab Structure

### Active Tab

- Currently downloading files
- Large progress bars
- Real-time speed/ETA
- Pause/Resume/Cancel buttons

### Queue Tab

- Files waiting to download
- Reorder by dragging
- Priority selector
- Bulk operations

### Completed Tab

- Successfully downloaded files
- File info (size, download time)
- Action buttons (delete, open, redownload)

### History Tab

- All completed downloads (archive)
- Search/filter
- Statistics
- Export option

---

## Backend Architecture for Multi-Download

**Download Manager Struct:**

```go
type DownloadManager struct {
    queue           []Download
    running         map[string]*Download
    maxConcurrent   int
    speedTracker    *SpeedTracker
    history         *History
    mutex           sync.RWMutex
    downloadDir     string
}

type SpeedTracker struct {
    samples []SpeedSample  // Last 10 samples
    mutex   sync.RWMutex
}

type SpeedSample struct {
    timestamp time.Time
    bytes     int64
}
```

**Key Methods:**

- `AddDownload(url, filename, priority)`
- `RemoveDownload(id)`
- `PauseDownload(id)`
- `ResumeDownload(id)`
- `ProcessQueue()` - Start next queued downloads up to maxConcurrent
- `GetAverageSpeed() int64`
- `GetStatistics() Statistics`

---

## Progressive Enhancement Strategy

### Phase 1: Build MVP

- Get basic download working
- Simple UI
- Don't worry about queue management yet

### Phase 2: Add Queue Support

- Extend backend for multiple downloads
- Update UI with tabs
- Keep features simple

### Phase 3: Add Speed/ETA

- Implement SpeedTracker
- Calculate ETA
- Display on UI

### Phase 4: Add Advanced Features

- History/Statistics
- Search/Filter
- Settings panel
- Keyboard shortcuts

### Phase 5: Polish

- Performance optimization
- Edge cases
- Documentation

---

## Testing Strategy

### Unit Tests

- URL validation
- File conflict resolution
- Speed calculation
- ETA calculation
- Queue management logic

### Integration Tests

- Download small files
- Resume downloads
- Queue processing
- Multiple concurrent downloads
- History persistence

### UI Tests

- Form validation
- Bulk operations
- Tab switching
- Search/filter
- Keyboard shortcuts

### Load Tests

- Many downloads in queue
- Large files
- Slow connections

---

## Success Metrics

1. **Functionality**: All features work as specified
2. **Performance**: Downloads don't lag UI, queuing is instant
3. **Reliability**: Resume works after app restart
4. **UX**: Intuitive without documentation
5. **Code Quality**: Well-organized, documented, tested

---

## Notes for Development

- Keep each feature in separate branch
- Test thoroughly before merge
- Consider Go's `database/sql` for history if needed
- Use WebSocket for real-time progress updates (optional)
- Consider rate limiting for API endpoints
- Implement proper error recovery
- Add request logging for debugging
