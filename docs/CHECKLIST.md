# Implementation Checklist

## Phase 1: MVP (Minimum Viable Product)

### Backend - Generic Download Logic

- [ ] Create `download_generic.go`
  - [ ] `downloadFile()` function with streaming
  - [ ] `extractFilenameFromURL()` function
  - [ ] `validateURL()` function
  - [ ] Range header support for resume
  - [ ] `.part` file handling
- [ ] Remove Ollama code from `download.go`
  - [ ] Delete `parseModel()`
  - [ ] Delete `getRegistryToken()`
  - [ ] Delete `getManifestOrIndex()`
  - [ ] Delete `downloadBlob()`
  - [ ] Delete `ollamaModelsDir()`
  - [ ] Delete `unzipToDir()`
  - [ ] Delete OCI/Docker constants
  - [ ] Delete bearer auth logic
- [ ] Update `sessionMeta` struct
  - [ ] Add `URL` field
  - [ ] Add `Filename` field
  - [ ] Add `ExpectedSize` field
  - [ ] Remove `Registry` field
  - [ ] Remove `Platform` field
  - [ ] Remove `Concurrency` field

### CLI Updates

- [ ] Update flag definitions
  - [ ] Remove `-registry` flag
  - [ ] Remove `-platform` flag
  - [ ] Keep `-o`, `-output-dir`, `-retries`, `-port`, `-v`
- [ ] Support URL as argument
  - [ ] Parse positional argument as URL
  - [ ] Support web UI mode (no args)

### Web UI - Form

- [ ] Update HTML form
  - [ ] Replace model input with URL input
  - [ ] Add optional filename input
  - [ ] Remove platform selector
  - [ ] Remove registry field
  - [ ] Remove concurrency field
- [ ] Update Persian labels
  - [ ] Title: "مدیریت دانلود فایل‌ها"
  - [ ] Form label: "آدرس دانلود"
  - [ ] Button: "دانلود فایل جدید"
  - [ ] All placeholders and help text
- [ ] Update `/download` handler
  - [ ] Parse `url` parameter
  - [ ] Parse `filename` parameter (optional)
  - [ ] Call `validateURL()`
  - [ ] Extract filename if needed

### Testing

- [ ] Small file download (100MB)
  - [ ] Progress updates correctly
  - [ ] Pause works
  - [ ] Resume works
- [ ] Large file download (1GB+)
  - [ ] Resume after interruption
  - [ ] Correct final file size
- [ ] Error cases
  - [ ] Invalid URL rejected
  - [ ] 404 handled gracefully
  - [ ] Timeout handled
- [ ] Session management
  - [ ] Restart and resume download
  - [ ] Progress preserved

### Finalize MVP

- [ ] All tests passing
- [ ] Code cleanup
- [ ] Commit with comprehensive message

---

## Phase 2: Manager Features

### Queue Manager

- [ ] Create `download_manager.go`
  - [ ] `DownloadManager` struct
  - [ ] `Download` struct
  - [ ] `AddDownload()` method
  - [ ] `RemoveDownload()` method
  - [ ] `PauseDownload()` method
  - [ ] `ResumeDownload()` method
  - [ ] `ProcessQueue()` method
  - [ ] Download worker goroutines
  - [ ] Mutex protection

### Speed Tracker

- [ ] Create `speed_tracker.go`
  - [ ] `SpeedTracker` struct
  - [ ] `SpeedSample` struct
  - [ ] `Record()` method
  - [ ] `GetSpeed()` method
  - [ ] `GetETA()` method
  - [ ] Moving average calculation

### History Manager

- [ ] Create `history.go`
  - [ ] `HistoryManager` struct
  - [ ] `HistoryEntry` struct
  - [ ] `Load()` method
  - [ ] `Save()` method
  - [ ] `AddEntry()` method
  - [ ] `GetStatistics()` method
  - [ ] Domain aggregation
  - [ ] File type aggregation

### API Endpoints

- [ ] Management endpoints
  - [ ] `GET /api/downloads`
  - [ ] `POST /api/download/add`
  - [ ] `POST /api/download/{id}/pause`
  - [ ] `POST /api/download/{id}/resume`
  - [ ] `POST /api/download/{id}/cancel`
  - [ ] `DELETE /api/download/{id}`
  - [ ] `POST /api/downloads/pause-all`
  - [ ] `POST /api/downloads/resume-all`
- [ ] Statistics endpoint
  - [ ] `GET /api/statistics`
  - [ ] `GET /api/history`

### Web UI - Tabs

- [ ] Tab navigation

  - [ ] Create tab structure
  - [ ] Active tab
  - [ ] Queue tab
  - [ ] Completed tab
  - [ ] History tab
  - [ ] Tab counts

- [ ] Active tab implementation

  - [ ] Display downloading files
  - [ ] Speed display
  - [ ] ETA display
  - [ ] Progress bar
  - [ ] Pause/Resume/Cancel buttons
  - [ ] Auto-refresh

- [ ] Queue tab implementation

  - [ ] List queued downloads
  - [ ] Priority selector
  - [ ] Drag-to-reorder
  - [ ] Move top/bottom buttons
  - [ ] Pause/Resume/Cancel buttons

- [ ] Completed tab implementation

  - [ ] Show finished downloads
  - [ ] File info (size, time)
  - [ ] Open file button
  - [ ] Open folder button
  - [ ] Delete button
  - [ ] Redownload button

- [ ] History tab implementation

  - [ ] Display all past downloads
  - [ ] Search functionality
  - [ ] Filter by date range
  - [ ] Statistics widget
  - [ ] Export option

- [ ] Bulk operations
  - [ ] Checkboxes on cards
  - [ ] Select all checkbox
  - [ ] Bulk toolbar
  - [ ] Delete selected
  - [ ] Pause selected
  - [ ] Resume selected

### Testing

- [ ] Queue management
  - [ ] Add multiple downloads
  - [ ] Verify order
  - [ ] Verify counts in tabs
- [ ] Concurrent downloads
  - [ ] Verify max concurrent limit
  - [ ] Verify worker pool
  - [ ] Verify auto-start
- [ ] Pause/Resume
  - [ ] Individual pause/resume
  - [ ] Pause/resume all
  - [ ] No data loss
- [ ] Speed and ETA
  - [ ] Speed calculation accurate
  - [ ] ETA updates
  - [ ] Moving average works
- [ ] History and stats
  - [ ] History persists
  - [ ] Statistics accurate
  - [ ] Search/filter works
- [ ] UI functionality
  - [ ] Tab switching
  - [ ] Bulk operations
  - [ ] Real-time updates
  - [ ] Drag-to-reorder

### Finalize Manager Phase

- [ ] All tests passing
- [ ] Performance acceptable
- [ ] Code cleanup
- [ ] Commit with comprehensive message

---

## Phase 3: Advanced Features (Optional)

### Features

- [ ] Bandwidth limiting
  - [ ] Per-download limit
  - [ ] Global limit
  - [ ] Speed throttling
- [ ] Custom headers
  - [ ] Header input UI
  - [ ] Authorization support
  - [ ] Preset storage
- [ ] Batch input
  - [ ] Textarea for URLs
  - [ ] Validation
  - [ ] Preview
  - [ ] Add all button
- [ ] Drag & drop
  - [ ] URL dropping
  - [ ] Visual feedback
- [ ] Settings panel
  - [ ] Download folder
  - [ ] Max concurrent (slider)
  - [ ] Bandwidth limit
  - [ ] Auto-start
  - [ ] Theme
  - [ ] Auto-clear option
- [ ] Advanced filtering
  - [ ] By extension
  - [ ] By size range
  - [ ] By date range
  - [ ] Save presets
- [ ] Keyboard shortcuts
  - [ ] Ctrl/Cmd + N
  - [ ] Ctrl/Cmd + A
  - [ ] Ctrl/Cmd + D
  - [ ] Ctrl/Cmd + P
  - [ ] Ctrl/Cmd + F
  - [ ] Ctrl/Cmd + S
- [ ] Theme support
  - [ ] Dark/Light toggle
  - [ ] Persistent choice
  - [ ] Responsive design
- [ ] Statistics dashboard
  - [ ] Total stats widget
  - [ ] Session stats widget
  - [ ] Speed graph
  - [ ] Activity chart
  - [ ] Top domains
  - [ ] File types
- [ ] Notifications
  - [ ] Toast notifications
  - [ ] Desktop notifications
  - [ ] Sound alerts
  - [ ] Tray icon

---

## Phase 4: Polish & Documentation

### Documentation

- [ ] Update README.md
  - [ ] New title
  - [ ] Feature list
  - [ ] Usage examples
  - [ ] Installation
  - [ ] Remove Ollama refs
- [ ] Update code comments
  - [ ] Remove Ollama docs
  - [ ] Add generic docs
  - [ ] Example comments
- [ ] Update CONTRIBUTING.md
  - [ ] Project structure
  - [ ] Development setup

### Code Quality

- [ ] Error handling
  - [ ] Review all errors
  - [ ] User-friendly messages
  - [ ] Helpful suggestions
  - [ ] Proper logging
- [ ] Performance
  - [ ] Profile code
  - [ ] Optimize hotspots
  - [ ] Reduce memory
  - [ ] Cache data
- [ ] Edge cases
  - [ ] File conflicts
  - [ ] Low disk space
  - [ ] Network issues
  - [ ] Large files
  - [ ] Special characters
- [ ] Code cleanup
  - [ ] Remove debug logs
  - [ ] Remove dead code
  - [ ] Format (gofmt)
  - [ ] Consistent naming
  - [ ] Add comments

### Final Steps

- [ ] Full test suite
- [ ] No regressions
- [ ] Performance acceptable
- [ ] All features working
- [ ] Final commit
- [ ] Merge to main

---

## Quick Stats

**Phase 1 (MVP)**: 16 tasks, 10-15 hours
**Phase 2 (Manager)**: 27 tasks, 20-25 hours  
**Phase 3 (Advanced)**: 10 tasks, 8-10 hours
**Phase 4 (Polish)**: 7 tasks, 3-4 hours

**Total**: 60 tasks, 41-54 hours
**Recommended**: 3-4 days for MVP, 2-3 additional days for full manager

---

## Status Tracking

Print this section and update as you progress:

```
PHASE 1: MVP
├─ Backend         [ ] 5/5 tasks
├─ CLI             [ ] 2/2 tasks
├─ UI              [ ] 5/5 tasks
├─ Testing         [ ] 5/5 tasks
└─ Finalize        [ ] 1/1 task

PHASE 2: Manager
├─ Queue Manager   [ ] 4/4 tasks
├─ Speed Tracker   [ ] 2/2 tasks
├─ History Manager [ ] 3/3 tasks
├─ API             [ ] 2/2 tasks
├─ UI Tabs         [ ] 6/6 tasks
├─ Testing         [ ] 6/6 tasks
└─ Finalize        [ ] 1/1 task

PHASE 3: Advanced
├─ Features        [ ] 10/10 tasks

PHASE 4: Polish
├─ Documentation   [ ] 3/3 tasks
├─ Code Quality    [ ] 4/4 tasks
└─ Final           [ ] 2/2 tasks
```

---

**Last Updated**: 2025-11-14
**Total Tasks**: 68
**Reference**: See docs/ for detailed plans
