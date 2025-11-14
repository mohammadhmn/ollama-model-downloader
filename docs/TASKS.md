# Task List: Convert to Full-Featured Download Manager

## Overview

Complete conversion of Ollama Model Downloader into a professional file download manager with three implementation phases:

- **Phase 1 (MVP)**: 16 tasks - Basic generic file downloading (3-4 days)
- **Phase 2 (Manager)**: 27 tasks - Queue management and advanced features (2-3 days)
- **Phase 3 (Advanced)**: 10 tasks - Polish features (ongoing)
- **Phase 4 (Polish)**: 7 tasks - Documentation and cleanup

**Total estimated effort**: 24-33 hours (3-4 working days for MVP, 2-3 additional for full manager)

---

## Phase 1: MVP (Minimum Viable Product) - HIGH PRIORITY

**Goal**: Convert to generic HTTP/HTTPS file downloader with pause/resume support

### Backend Refactoring (6 hours)

- [ ] **1.1** Create `download_generic.go` with generic HTTP download functions

  - `downloadFile()` - main download with streaming
  - `extractFilenameFromURL()` - parse filename from URL
  - `validateURL()` - validate HTTP/HTTPS URLs
  - Support Range headers for resume capability
  - Handle `.part` files for incomplete downloads

- [ ] **1.2** Remove all Ollama-specific code from `download.go`

  - Remove `parseModel()`
  - Remove `getRegistryToken()` and bearer auth logic
  - Remove `getManifestOrIndex()` (OCI/Docker manifest)
  - Remove `downloadBlob()` and blob verification
  - Remove `ollamaModelsDir()` and unzip functionality
  - Remove OCI/Docker constants (mtOCIIndex, mtDockerIndex, etc.)
  - Remove staging/blob directory structure

- [ ] **1.3** Simplify `sessionMeta` struct in `main.go`

  - Add: `URL`, `Filename`, `ExpectedSize`
  - Remove: `Registry`, `Platform`, `Concurrency`
  - Keep: `SessionID`, `OutPath`, `StagingRoot`, `Retries`, `State`, `Message`
  - Update: Start/End timestamps, progress tracking

- [ ] **1.4** Implement resume support

  - Check for existing `.part` files
  - Add Range header: `bytes=<offset>-`
  - Handle 206 Partial Content responses
  - Rename `.part` to final filename on completion

- [ ] **1.5** Add URL validation and parsing
  - Validate URL format and scheme (HTTP/HTTPS only)
  - Extract filename from URL path or use fallback
  - Handle URLs with query parameters
  - Parse Content-Length header when available

### CLI Updates (1-2 hours)

- [ ] **2.1** Update flag definitions

  - Remove: `-registry`, `-platform`, `-concurrency`
  - Keep: `-o` (output), `-output-dir`, `-retries`, `-port`, `-v`
  - Add: `-url` flag for web UI pre-fill

- [ ] **2.2** Accept URL as positional argument
  - Example: `./downloader https://example.com/file.zip`
  - Extract filename automatically if not provided
  - Support web UI mode when no URL provided

### Web UI - Form & Display (3-4 hours)

- [ ] **3.1** Update form inputs

  - Replace "نام مدل" (model name) input with "آدرس دانلود" (download URL)
  - Add optional "نام فایل" (filename) input
  - Keep "تعداد تلاش" (retries) field

- [ ] **3.2** Remove unnecessary form fields

  - Remove platform selector
  - Remove registry field
  - Remove concurrency input (for MVP)

- [ ] **3.3** Update all Persian UI labels

  - "مدیریت دانلود مدل‌های Ollama" → "مدیریت دانلود فایل‌ها"
  - "دانلود مدل جدید" → "دانلود فایل جدید"
  - "نام مدل" → "آدرس دانلود (URL)"
  - Update placeholders and help text

- [ ] **3.4** Update download card display

  - Show URL instead of model name
  - Display filename separately
  - Show file size when available
  - Simplify information layout

- [ ] **3.5** Update `/download` handler
  - Parse `url` parameter instead of `model`
  - Parse optional `filename` parameter
  - Call `validateURL()` before processing
  - Extract filename if not provided
  - Create download session with generic options

### Testing (3-4 hours)

- [ ] **4.1** Test small file download

  - Download 50-100MB file
  - Verify progress updates
  - Test pause functionality
  - Test resume functionality
  - Verify file integrity

- [ ] **4.2** Test large file download and resume

  - Download 1GB+ file
  - Interrupt mid-download
  - Verify `.part` file exists
  - Resume and complete download
  - Verify speed and progress accuracy

- [ ] **4.3** Test error handling

  - Invalid URL format → show error message
  - 404 Not Found → display error
  - Network timeout → handle gracefully
  - Missing Content-Length → allow unknown size

- [ ] **4.4** Test CLI mode

  - `./downloader https://example.com/file.zip`
  - Download completes and saves to correct location
  - Test with various file types (zip, iso, tar, bin)

- [ ] **4.5** Test session persistence
  - Start download
  - Close application
  - Restart application
  - Verify session is restored
  - Verify can resume download

### Completion

- [ ] **MVP Commit** Push all changes with comprehensive message

  ```
  feat: implement MVP generic file downloader

  - Remove all Ollama-specific logic (OCI, registry, platform)
  - Add generic HTTP/HTTPS download with resume support
  - Update CLI to accept URLs directly
  - Simplify sessionMeta for generic downloads
  - Update web UI form for URL input
  - Update Persian labels for generic downloader
  - Implement comprehensive testing
  ```

---

## Phase 2: Manager Features - MEDIUM PRIORITY

**Goal**: Multi-download support with queue management, speed tracking, and history

### Queue Manager (4-5 hours)

- [ ] **5.1** Create `download_manager.go`

  - `DownloadManager` struct with thread-safe queue
  - `Download` struct with full metadata
  - Context support for cancellation
  - Sync.RWMutex for thread safety

- [ ] **5.2** Define Download struct

  - ID (unique identifier)
  - URL, Filename, OutputPath
  - Status (queued, active, paused, completed, error)
  - Priority (1-10, higher = earlier)
  - Progress (bytes downloaded, total bytes)
  - Timestamps (created, started, completed)
  - ErrorMsg for failed downloads

- [ ] **5.3** Implement DownloadManager methods

  - `AddDownload(url, filename, path)` → ID
  - `RemoveDownload(id)` → error
  - `PauseDownload(id)` → error
  - `ResumeDownload(id)` → error
  - `CancelDownload(id)` → error
  - `GetDownload(id)` → Download
  - `GetAll()` → []Download

- [ ] **5.4** Implement multi-goroutine worker pool
  - `ProcessQueue()` - main queue processor
  - `downloadWorker()` - goroutine per active download
  - Respect `maxConcurrent` limit
  - Auto-start next download when one completes
  - Handle pause/resume transitions
  - Track running downloads

### Speed Tracker (2-3 hours)

- [ ] **6.1** Create `speed_tracker.go`

  - `SpeedTracker` struct with sample history
  - `SpeedSample` with timestamp and bytes
  - Keep rolling window of last 10 samples

- [ ] **6.2** Implement speed and ETA calculations
  - `Record(bytes)` - record sample at current time
  - `GetSpeed()` - calculate bytes/second
  - Use moving average to smooth spikes
  - `GetETA(total, downloaded)` - estimate time remaining
  - Format output: "2.5 MB/s", "5m 30s remaining"

### History Manager (3-4 hours)

- [ ] **7.1** Create `history.go`

  - `HistoryManager` struct for persistence
  - `HistoryEntry` with download metadata
  - JSON-based storage (`.history/history.json`)
  - Thread-safe access with RWMutex

- [ ] **7.2** Implement persistence

  - `Load()` - read history from JSON file
  - `Save()` - write history to JSON file
  - `AddEntry(entry)` - add completed download
  - Auto-save after each entry
  - Handle file not found gracefully

- [ ] **7.3** Implement statistics calculation
  - `GetStatistics()` → Statistics struct
  - Total files and bytes downloaded
  - Average download speed
  - Today's downloads vs all-time
  - Top domains by bandwidth
  - File type distribution by count and size

### API Endpoints (3 hours)

- [ ] **8.1** Implement management endpoints

  - `GET /api/downloads` - get all downloads (JSON)
  - `POST /api/download/add` - add new download
  - `POST /api/download/{id}/pause` - pause specific
  - `POST /api/download/{id}/resume` - resume specific
  - `POST /api/download/{id}/cancel` - cancel specific
  - `DELETE /api/download/{id}` - delete from queue
  - `POST /api/downloads/pause-all` - pause all
  - `POST /api/downloads/resume-all` - resume all

- [ ] **8.2** Implement statistics endpoint
  - `GET /api/statistics` - return aggregated stats
  - `GET /api/history` - get completed downloads
  - Response includes counts, speeds, aggregations

### Web UI - Tab Structure (4-5 hours)

- [ ] **9.1** Create tab navigation

  - Active tab (currently downloading)
  - Queue tab (waiting to download)
  - Completed tab (finished downloads)
  - History tab (all past downloads)
  - Tab indicators showing counts: "Active (1)"

- [ ] **9.2** Implement Active tab

  - Display only currently downloading files
  - Large progress bar with percentage
  - Real-time speed display: "2.5 MB/s ↓"
  - ETA display: "5m 30s remaining"
  - Downloaded/Total: "524 MB / 1.2 GB"
  - Pause/Resume/Cancel buttons per download
  - Auto-refresh progress every 1 second

- [ ] **9.3** Implement Queue tab

  - List all queued downloads
  - Show priority for each (1-10)
  - Drag-to-reorder queue
  - Priority selector dropdown
  - Show file size and URL
  - Move to Top/Bottom buttons
  - Pause/Resume/Cancel buttons

- [ ] **9.4** Implement Completed tab

  - List successfully downloaded files
  - Show filename, size, download time
  - Show average speed for that download
  - Open file button
  - Open folder button
  - Delete button
  - Redownload button

- [ ] **9.5** Implement History tab

  - Display all past downloads (persisted)
  - Search/filter by filename or URL
  - Filter by date range
  - Filter by status (completed, error)
  - Statistics widget:
    - Total downloaded: XXX GB
    - Files completed: XXX
    - Average speed: X.X MB/s
  - Delete history entries
  - Export history to CSV

- [ ] **9.6** Add bulk operations toolbar
  - Checkbox on each download card
  - "Select All" / "Deselect All" in header
  - Show when items selected:
    - Delete Selected button
    - Pause Selected button
    - Resume Selected button
    - Move to Top/Bottom buttons
    - Open Folders button
  - JavaScript to track selections

### Testing (4-5 hours)

- [ ] **10.1** Test queue management

  - Add 5 URLs to queue
  - Verify all queued
  - Verify order correct
  - Verify counts show in tabs

- [ ] **10.2** Test concurrent downloads

  - Start 4 downloads (max concurrent)
  - Verify only 4 active at once
  - Add 5th - verify queued
  - Pause one - verify 5th starts
  - Check worker pool efficiency

- [ ] **10.3** Test pause/resume operations

  - Pause individual download
  - Verify shows as paused in UI
  - Resume and verify continues
  - Pause all downloads
  - Resume all
  - Verify no data loss during pause

- [ ] **10.4** Test speed and ETA accuracy

  - Download and monitor speed readings
  - Verify moving average smoothing
  - Check ETA updates every second
  - Download variable-speed file
  - Verify final times logged correctly

- [ ] **10.5** Test history and persistence

  - Complete multiple downloads
  - Close application
  - Restart and verify history loaded
  - Check statistics calculated correctly
  - Verify history file written
  - Test search/filter functionality

- [ ] **10.6** Test UI functionality
  - Tab switching works smoothly
  - Bulk operations select correctly
  - Progress updates in real-time
  - Drag-to-reorder in queue
  - Statistics widget updates
  - Search filters results

### Completion

- [ ] **Manager Commit** Push all Phase 2 changes

  ```
  feat: implement full-featured download manager

  - Add DownloadManager for queue handling
  - Implement SpeedTracker for real-time speed/ETA
  - Create HistoryManager for persistent records
  - Support multiple concurrent downloads
  - Add tabs: Active, Queue, Completed, History
  - Implement bulk operations
  - Add statistics dashboard
  - Full API endpoints for all operations
  - Comprehensive testing and validation
  ```

---

## Phase 3: Advanced Features - LOW PRIORITY

**Goal**: Power user features and polish

- [ ] **11.1** Implement bandwidth limiting

  - Per-download KB/s cap
  - Global bandwidth limit option
  - Throttle write speed to limit

- [ ] **11.2** Add custom headers support

  - Text input for custom headers
  - Support Authorization headers
  - Store common header presets

- [ ] **11.3** Implement batch URL input

  - Textarea for paste multiple URLs
  - One URL per line
  - Validate all before adding
  - Show preview of each
  - "Add All" button

- [ ] **11.4** Add drag & drop support

  - Drop URLs on input area
  - Drag files to queue
  - Visual feedback during drag

- [ ] **11.5** Create settings panel

  - Default download folder (with browser)
  - Max concurrent downloads (slider 1-32)
  - Bandwidth limit (checkbox + input)
  - Auto-start downloads (toggle)
  - Clear completed after X hours (selector)
  - Theme preference (Light/Dark/Auto)

- [ ] **11.6** Implement advanced search/filter

  - Filter by file type/extension
  - Filter by size range
  - Filter by date range
  - Save filter presets
  - Combination filters

- [ ] **11.7** Add keyboard shortcuts

  - Ctrl/Cmd + N → New download
  - Ctrl/Cmd + A → Select all
  - Ctrl/Cmd + D → Delete selected
  - Ctrl/Cmd + P → Pause/Resume selected
  - Ctrl/Cmd + Shift + P → Pause all
  - Ctrl/Cmd + Shift + R → Resume all
  - Ctrl/Cmd + F → Focus search
  - Ctrl/Cmd + S → Settings
  - Del → Delete selected

- [ ] **11.8** Implement theme support

  - Dark/Light theme toggle
  - Persistent preference in localStorage
  - Responsive design for mobile/tablet
  - Smooth theme transitions

- [ ] **11.9** Create statistics dashboard

  - Widget: Total downloaded (size and count)
  - Widget: This session stats
  - Widget: Average speed graph
  - Widget: Activity by day (week view)
  - Widget: Top domains by bandwidth
  - Widget: File type distribution
  - Export statistics to JSON/CSV

- [ ] **11.10** Add notifications
  - Toast notifications for actions
  - Desktop notifications on completion
  - Sound alert option
  - System tray updates
  - Unread count badge

---

## Phase 4: Polish & Documentation - LOW PRIORITY

**Goal**: Final cleanup, documentation, and optimization

- [ ] **12.1** Update README.md

  - Change title and description
  - Update feature list
  - Remove Ollama references
  - Update usage examples
  - Update screenshot/GIF (if any)
  - Update installation instructions

- [ ] **12.2** Update code comments

  - Remove Ollama documentation
  - Add generic download comments
  - Document URL format requirements
  - Add examples in comments
  - Update function documentation

- [ ] **12.3** Improve error handling

  - Review all error messages
  - Make messages user-friendly
  - Add helpful suggestions
  - Log errors for debugging
  - Handle all edge cases

- [ ] **12.4** Performance optimization

  - Profile download speed
  - Optimize progress updates
  - Reduce memory footprint for large queues
  - Cache frequently accessed data
  - Lazy-load history entries

- [ ] **12.5** Handle edge cases

  - File conflict (existing file) → auto-rename
  - Low disk space → warn user
  - Network interruption → retry with backoff
  - Very large files → handle streaming properly
  - Special characters in filenames → sanitize

- [ ] **12.6** Code cleanup
  - Remove debug logs
  - Remove dead code
  - Consistent code style (gofmt)
  - Consistent naming conventions
  - Add/update comments

### Final Steps

- [ ] **Final Testing** - End-to-end testing across all features

  - MVP features still work
  - Manager features stable
  - Advanced features functional
  - No regressions
  - Performance acceptable

- [ ] **Final Commit** - Merge to main

  ```
  feat: release full-featured download manager

  - Complete feature parity with download managers
  - Polished UI and error handling
  - Comprehensive documentation
  - Full test coverage
  - Ready for production use
  ```

---

## Quick Reference

### Implementation Order

**Week 1 (MVP)**

1. Backend refactoring (remove Ollama, add generic downloads)
2. CLI updates
3. UI form updates
4. Handler updates
5. Testing and fixes
6. Commit

**Week 2 (Manager)** 7. Queue manager implementation 8. Speed tracker 9. History manager 10. API endpoints 11. Tab UI components 12. Testing and fixes 13. Commit

**Week 3+ (Optional)** 14. Advanced features (bandwidth, batch input, etc.) 15. Polish and documentation 16. Final testing and release

### Testing Checklist

**MVP Verification**

- [ ] Single file downloads work
- [ ] Pause/Resume functionality
- [ ] Progress display accurate
- [ ] Session persistence
- [ ] Error handling clear
- [ ] No Ollama code remains
- [ ] CLI works with URLs
- [ ] Web UI clean and functional

**Manager Verification**

- [ ] Multiple downloads in queue
- [ ] Speed tracking works
- [ ] ETA accurate
- [ ] History persists
- [ ] Statistics correct
- [ ] Bulk operations smooth
- [ ] Search/filter functional
- [ ] Keyboard shortcuts work
- [ ] Performance acceptable
- [ ] Edge cases handled

---

## Time Estimates

| Phase        | Component       | Hours     | Effort     |
| ------------ | --------------- | --------- | ---------- |
| **MVP**      | Backend         | 3-5       | Medium     |
|              | CLI             | 1-2       | Low        |
|              | UI              | 3-4       | Medium     |
|              | Testing         | 3-4       | High       |
|              | **Subtotal**    | **10-15** |            |
| **Manager**  | Queue Manager   | 4-5       | High       |
|              | Speed Tracker   | 2-3       | Medium     |
|              | History Manager | 3-4       | Medium     |
|              | API             | 3         | Medium     |
|              | UI Tabs         | 4-5       | Medium     |
|              | Testing         | 4-5       | High       |
|              | **Subtotal**    | **20-25** |            |
| **Advanced** | Features        | 8-10      | Low-Medium |
| **Polish**   | Documentation   | 3-4       | Low        |
|              | **Total**       | **41-54** |            |

**Recommended**: 3-4 working days for MVP, 2-3 additional days for Manager phase.

---

## Key Files to Modify/Create

### Create New Files

- `download_generic.go` - Generic HTTP download logic
- `download_manager.go` - Queue management
- `speed_tracker.go` - Speed and ETA calculation
- `history.go` - Download history persistence

### Modify Existing Files

- `main.go` - Remove Ollama logic, add manager integration
- `download.go` - Keep progress, remove OCI/blob code
- `templates/index.html` - Update form, add tabs, add API calls
- `README.md` - Update documentation

### Remove/Deprecate

- OCI/Docker registry constants
- Ollama-specific configuration
- Unzip/blob extraction logic

---

## Success Metrics

### MVP Phase

✅ Download any HTTP/HTTPS file successfully
✅ Show accurate progress percentage
✅ Resume interrupted downloads
✅ Pause/Resume functionality works
✅ Web UI clean and functional
✅ CLI works with URLs
✅ Session persistence across restarts
✅ Error messages are helpful

### Manager Phase

✅ Multiple concurrent downloads with queue
✅ Speed tracking and ETA calculation
✅ Download history with statistics
✅ Bulk operations work smoothly
✅ Advanced search and filtering
✅ Settings panel for customization
✅ Responsive UI for mobile/tablet
✅ Keyboard shortcuts responsive
✅ No performance degradation
✅ Edge cases handled gracefully

---

## Notes

- Keep each feature in separate branch during development
- Test thoroughly before merging
- Use WebSocket for real-time updates (optional enhancement)
- Consider SQLite for history if JSON becomes too large
- Add request logging for debugging
- Implement proper error recovery and retry logic
- Reference: CONVERSION_PLAN.md, IMPLEMENTATION_STEPS.md, FULL_FEATURED_ROADMAP.md
