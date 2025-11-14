# Conversion Plan: Ollama Model Downloader → General Purpose File Downloader

## Overview

Convert the Ollama-specific downloader into a flexible file downloader that accepts any HTTP/HTTPS URL.

## Phase 1: Backend Architecture Changes

### 1.1 Remove Ollama-Specific Logic

- [ ] Remove `parseModel()` function (Ollama-specific)
- [ ] Remove `getRegistryToken()` - bearer auth for OCI registry
- [ ] Remove `getManifestOrIndex()` - OCI/Docker manifest fetching
- [ ] Remove `downloadBlob()` - OCI blob download with digest verification
- [ ] Remove Ollama registry constants (`mtOCIIndex`, `mtDockerIndex`, etc.)
- [ ] Remove `ollamaModelsDir()` and unzip functionality
- [ ] Remove staging/blob structure (models/manifests/blobs)
- [ ] Remove session metadata fields related to Ollama (registry, platform, etc.)

### 1.2 Create Generic Download Logic

- [ ] Create new `download.go` download function for direct URL downloads
- [ ] Implement simple HTTP GET with streaming
- [ ] Support resume capability (Range headers)
- [ ] Add file hash validation (optional, can skip)
- [ ] Track download progress with size information
- [ ] Handle redirects and follow location headers

### 1.3 Update Data Structures

- [ ] Simplify `options` struct:
  - Remove: registry, platform, concurrency, retries
  - Keep: model/name, outZip, outputDir, sessionID, stagingDir
  - Add: url, filename (optional)
- [ ] Simplify `sessionMeta` struct to remove Ollama fields
- [ ] Update `downloadEntry` for generic files
- [ ] Rename concepts: "model" → "download", "registry" → "url"

### 1.4 Simplify Download Flow

- [ ] Create generic `downloadFile()` function
- [ ] Handle URL validation
- [ ] Extract filename from URL if not provided
- [ ] Direct file download (no manifest/blob structure)
- [ ] Save directly to output path (no staging/zipping unless user wants it)
- [ ] Update progress tracking for single file downloads

## Phase 2: UI/Frontend Changes

### 2.1 Update Form Inputs

- [ ] Change "model name" input to "Download URL"
- [ ] Add optional "Filename" input (auto-fill from URL)
- [ ] Remove "Platform" selector
- [ ] Remove "Registry" field
- [ ] Keep "Concurrency" (for multiple files) or remove for simplicity
- [ ] Keep "Retries" for reliability

### 2.2 Update Labels & Text (Change from Persian Ollama-specific)

- [ ] "مدیریت دانلود مدل‌های Ollama" → "مدیریت دانلود فایل‌ها" (File Download Manager)
- [ ] "دانلود مدل جدید" → "دانلود فایل جدید"
- [ ] "نام مدل" → "آدرس دانلود (URL)"
- [ ] "کتابخانه مدل‌ها" → "فایل‌های دانلود شده"
- [ ] Remove "استخراج" (unzip) action or make it optional
- [ ] Update all placeholders and help text

### 2.3 Update Download Card Display

- [ ] Show URL instead of model name
- [ ] Show file size when available
- [ ] Show filename separately
- [ ] Simplify displayed information

### 2.4 Actions

- [ ] Keep: Delete, Open folder
- [ ] Remove: Unzip (convert to optional download option)
- [ ] Add: Copy download link button

## Phase 2.5: Full-Fledged Download Manager Features

### 2.5.1 Multiple Concurrent Downloads
- [ ] Queue management system (add, remove, reorder)
- [ ] Support for multiple simultaneous downloads (configurable)
- [ ] Priority system (high, normal, low)
- [ ] Pause individual or all downloads
- [ ] Resume individual or all downloads
- [ ] Cancel individual or all downloads
- [ ] Visual queue display with status for each

### 2.5.2 Advanced Progress Tracking
- [ ] Show download speed (KB/s, MB/s)
- [ ] Calculate and display ETA (time remaining)
- [ ] Show time elapsed
- [ ] Show current/total file size in real-time
- [ ] Percentage progress bar with smooth animation
- [ ] Aggregate statistics for multiple downloads

### 2.5.3 Download History & Statistics
- [ ] Track all completed downloads with timestamps
- [ ] Show completion time for each download
- [ ] Display total downloaded data (session and all-time)
- [ ] Show average download speed
- [ ] Persistent history across sessions
- [ ] Filter/search download history
- [ ] Delete history entries

### 2.5.4 Enhanced Form Controls
- [ ] Batch URL input (paste multiple URLs at once)
- [ ] URL validation with preview of filename/size
- [ ] Drag & drop for URL input or file list
- [ ] Save download presets/templates
- [ ] Import/export download lists
- [ ] Concurrent download limit selector (1-32)
- [ ] Bandwidth limiting option (KB/s cap)
- [ ] Auto-retry configuration
- [ ] Notification settings

### 2.5.5 Download Management UI
- [ ] Tabs/sections: Active, Queue, Completed, History
- [ ] Search/filter downloads by name or URL
- [ ] Sort by: name, size, speed, date, status
- [ ] Bulk actions (select multiple downloads):
  - [ ] Delete selected
  - [ ] Pause/Resume selected
  - [ ] Move to top/bottom of queue
  - [ ] Open folder for selected

### 2.5.6 File Management
- [ ] Show file size after header fetch
- [ ] Confirm overwrite if file exists
- [ ] Auto-rename on conflict (file.zip → file (1).zip)
- [ ] Move/copy downloaded files
- [ ] View file properties (size, type, date)
- [ ] Open file location or file directly
- [ ] Calculate and display total disk space used

### 2.5.7 Advanced Download Options
- [ ] Custom headers input (for protected URLs)
- [ ] User-Agent selector (browser, custom)
- [ ] Follow redirects (automatic, configurable)
- [ ] Cookie support (future)
- [ ] Proxy configuration (future)
- [ ] SSL/TLS verification toggle
- [ ] Filename pattern templates

### 2.5.8 UI/UX Enhancements
- [ ] Dark/Light theme toggle
- [ ] Responsive design for mobile/tablet
- [ ] Keyboard shortcuts for common actions
- [ ] Right-click context menu for downloads
- [ ] Drag to reorder queue
- [ ] Floating progress widget (minimize main window)
- [ ] Notifications for completion/errors
- [ ] Auto-refresh settings (interval selector)
- [ ] Smooth animations and transitions
- [ ] Accessibility improvements (ARIA labels, keyboard nav)

### 2.5.9 Status Indicators & Notifications
- [ ] Color-coded status badges:
  - [ ] Green: Downloading/Active
  - [ ] Yellow: Paused
  - [ ] Blue: Queued
  - [ ] Red: Error
  - [ ] Gray: Completed
- [ ] Toast notifications for actions
- [ ] Desktop notifications for completion (future)
- [ ] Error detail messages with suggestions
- [ ] Warning for slow connections

### 2.5.10 Settings Panel
- [ ] Download folder selection
- [ ] Default save location
- [ ] Max concurrent downloads
- [ ] Bandwidth limit
- [ ] Auto-start downloads
- [ ] Clear completed automatically
- [ ] Theme preference
- [ ] Language selection
- [ ] Advanced logging options
- [ ] Cache/database cleanup

### 2.5.11 Search & Filtering
- [ ] Search by filename or URL
- [ ] Filter by status (active, paused, completed, error)
- [ ] Filter by date range
- [ ] Filter by file type/extension
- [ ] Filter by size range
- [ ] Save filters as presets

### 2.5.12 Download Statistics Dashboard
- [ ] Total files downloaded (count & size)
- [ ] Average download speed
- [ ] Today's downloads vs this week vs all-time
- [ ] Top downloaded file types
- [ ] Most frequent domains
- [ ] Total time saved by pausing/resuming
- [ ] Performance metrics chart

## Phase 3: Configuration & Settings

### 3.1 Command Line Flags

- [ ] Change to accept URLs directly
- [ ] Example: `./downloader https://example.com/file.zip`
- [ ] Remove: `-registry`, `-platform`
- [ ] Keep: `-o` (output path), `-output-dir`, `-port`, `-v`, `-retries`
- [ ] Add: `-url` (for web UI)

### 3.2 Web UI Form Parameters

- [ ] Change POST parameter from "model" to "url"
- [ ] Add optional "filename" parameter
- [ ] Remove "registry", "platform" parameters

## Phase 4: File Organization

### 4.1 Naming & Structure

- [ ] Rename project (optional): `ollama-model-downloader` → `file-downloader` or `universal-downloader`
- [ ] Update main.go comments
- [ ] Update README.md
- [ ] Update Dockerfile (if applicable)
- [ ] Update go.mod (if renaming package)

### 4.2 Cleanup

- [ ] Remove config/ directory if Ollama-specific
- [ ] Keep internal/ if it contains generic utilities
- [ ] Remove/Update templates/ README if exists

## Phase 5: Session Management Updates

### 5.1 Update Session Metadata

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

### 5.2 Remove File Staging

- [ ] Simplify to direct download (no blob structure)
- [ ] Keep staging for incomplete downloads (.part files)
- [ ] Remove zip packaging (save raw files)

## Phase 6: Error Handling

### 6.1 HTTP-Specific Errors

- [ ] Handle 404 (file not found)
- [ ] Handle 403 (permission denied)
- [ ] Handle rate limiting (429)
- [ ] Handle timeouts
- [ ] Handle redirects (follow automatically)
- [ ] Handle content-length mismatch

### 6.2 User Feedback

- [ ] Clear error messages for invalid URLs
- [ ] Show expected file size when available
- [ ] Handle cases where size is unknown
- [ ] Better progress indication for large files

## Phase 7: Testing & Validation

### 7.1 Test Cases

- [ ] Download small text file
- [ ] Download large binary file
- [ ] Resume interrupted download
- [ ] Handle missing Content-Length header
- [ ] Invalid URL rejection
- [ ] Concurrent downloads (if keeping concurrency)
- [ ] Pause/Resume functionality

### 7.2 Browser Testing

- [ ] Test with multiple browsers
- [ ] Test progress bar accuracy
- [ ] Test session resume after restart
- [ ] Test file deletion and folder operations

## Phase 8: Documentation

### 8.1 Update README

- [ ] Change title and description
- [ ] Update feature list
- [ ] Update usage examples
- [ ] Update installation instructions
- [ ] Remove Ollama-specific sections

### 8.2 Update Code Comments

- [ ] Remove Ollama references in comments
- [ ] Add generic download documentation
- [ ] Document URL format requirements

## Implementation Order

### Iteration 1: MVP (Minimum Viable Product)
1. **Backend** (Phases 1, 3):
   - Create generic download logic
   - Update data structures
   - Get CLI working with URLs

2. **Basic UI** (Phase 2.1-2.4):
   - Update form and labels
   - Basic progress display
   - Test web interface

3. **Integration** (Phase 5):
   - Update session management
   - Test download flow

4. **Polish** (Phases 4, 6, 7, 8):
   - Cleanup and documentation
   - Error handling refinement

### Iteration 2: Full-Featured Download Manager
5. **Advanced Backend**:
   - Multi-download queue support
   - Download speed tracking
   - Bandwidth limiting
   - History/statistics persistence

6. **Full-Featured UI** (Phase 2.5):
   - Queue management (Active, Queue, Completed, History tabs)
   - Advanced progress tracking (speed, ETA)
   - Bulk operations
   - Settings panel
   - Search/filter
   - Statistics dashboard

7. **Polish & Testing** (Phase 7):
   - Comprehensive testing
   - Performance optimization
   - Edge case handling

## Key Considerations

- **Backward Compatibility**: Not needed (fresh start)
- **Concurrency**: Can simplify to single-file downloads initially
- **Authentication**: Support basic auth in URL? (https://user:pass@host/file)
- **File Size**: Determine size from Content-Length header or user input
- **Resume Support**: Keep using Range header and .part files
- **Validation**: Validate URL format before attempting download
- **Security**: Validate URLs to prevent path traversal attacks

## Success Criteria

### MVP Phase (Iteration 1)
- ✅ Download any HTTP/HTTPS file successfully
- ✅ Show accurate progress percentage
- ✅ Resume interrupted downloads
- ✅ Pause/Resume functionality works
- ✅ Web UI is clean and functional
- ✅ CLI works with URLs
- ✅ Session management persists across restarts
- ✅ Error messages are helpful
- ✅ Basic file management (delete, open folder)

### Full-Featured Phase (Iteration 2)
- ✅ Multiple concurrent downloads with queue management
- ✅ Download speed tracking (KB/s, MB/s)
- ✅ ETA calculation and display
- ✅ Download history with statistics
- ✅ Bulk operations (pause all, resume all, delete multiple)
- ✅ Advanced search and filtering
- ✅ Settings panel for customization
- ✅ Bandwidth limiting support
- ✅ Statistics dashboard with metrics
- ✅ Responsive UI for mobile/tablet
- ✅ Keyboard shortcuts for power users
- ✅ Drag & drop file/URL support
- ✅ URL validation with preview
- ✅ Auto-rename on file conflicts
- ✅ Toast notifications for user feedback
