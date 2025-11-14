# Quick Reference: Conversion Plan Summary

## Branch

- **Name:** `feature/general-purpose-downloader`
- **Status:** ✅ Created and active

## High-Level Transformation

```
Ollama Model Downloader
    ↓
General Purpose File Downloader
    ↓
Full-Featured Download Manager
```

## What Stays

- Web UI framework (HTTP handlers, templating)
- Session persistence mechanism (.staging folders)
- Progress tracking core logic
- Pause/Resume capability (Range headers)
- File operations (delete, open folder)
- Error handling patterns

## What Gets Removed

```
❌ parseModel()           - Ollama model parsing
❌ getRegistryToken()     - Bearer auth for registries
❌ getManifestOrIndex()   - OCI manifest fetching
❌ downloadBlob()         - Digest-based downloads
❌ ollamaModelsDir()      - Ollama-specific paths
❌ unzipToDir()           - Zip extraction (make optional)
❌ Models/blobs/staging   - OCI blob structure
❌ Registry constants     - OCI/Docker types
```

## What Gets Added (Iteration 1: MVP)

```
✅ downloadFile(url)          - Generic HTTP download
✅ validateURL()              - URL validation
✅ extractFilenameFromURL()   - Parse filename
✅ URL form input             - Replace model name
✅ Simplified sessionMeta     - Remove Ollama fields
```

## What Gets Added (Iteration 2: Full Manager)

```
✅ DownloadQueue              - Multiple downloads
✅ SpeedTracker               - Speed/ETA calculation
✅ Download History           - Persistent database
✅ Queue Management UI        - Active/Queue/Completed tabs
✅ Bulk Operations            - Pause all, delete selected, etc
✅ Statistics Dashboard       - Metrics and charts
✅ Advanced Search/Filter     - By domain, date, size
✅ Settings Panel             - User preferences
✅ Keyboard Shortcuts         - Power user features
```

## What Gets Added (Iteration 3: Polish)

```
✅ Documentation              - README, FEATURES, Contributing guide
✅ Error Handling             - User-friendly messages with suggestions
✅ Performance Optimization   - CPU/memory efficiency, large files
✅ Edge Case Handling         - File conflicts, low disk space
✅ Security Review            - URL validation, path traversal prevention
✅ Comprehensive Testing      - Manual checklist and automated tests
✅ Changelog                  - Document all changes
```

## What Gets Added (Iteration 4: Automation - Optional)

```
✅ Download Scheduling        - Cron-based scheduled downloads
✅ Mirror Fallback            - Automatic fallback to alternate sources
✅ Batch Import               - Import multiple URLs at once
✅ CLI Improvements           - Advanced flags, output formats
✅ Webhook Notifications      - External system integration
✅ Docker Support             - Container deployment
✅ Metrics & Monitoring       - Statistics and health API
✅ Example Scripts            - Real-world usage examples
```

## File Changes Summary

### Backend Files to Modify

| File          | Changes                                             | Priority |
| ------------- | --------------------------------------------------- | -------- |
| `main.go`     | Remove Ollama flags, update handlers, simplify form | HIGH     |
| `download.go` | Remove OCI logic, add generic HTTP download         | HIGH     |
| Go structs    | `options`, `sessionMeta`, `downloadEntry`           | HIGH     |
| CLI flags     | Change to accept URLs                               | HIGH     |

### Frontend Files to Modify

| File                   | Changes                                    | Priority |
| ---------------------- | ------------------------------------------ | -------- |
| `templates/index.html` | URL input, update labels, add new sections | MEDIUM   |
| Form logic             | Remove model/registry/platform fields      | MEDIUM   |
| Tab structure          | Keep Active/Queue/Completed + add History  | MEDIUM   |
| JavaScript             | Update form handling and API calls         | MEDIUM   |

### New Files to Create

| File                      | Purpose                        | Phase     |
| ------------------------- | ------------------------------ | --------- |
| `download_manager.go`     | Queue and concurrent downloads | Phase 2.5 |
| `history.go`              | History and statistics         | Phase 2.5 |
| `speed_tracker.go`        | Speed and ETA calculation      | Phase 2.5 |
| `.history/downloads.json` | Persistent history             | Phase 2.5 |

## Key Data Structure Changes

### Before (Ollama)

```go
type sessionMeta struct {
    Model       string      // "llama3.2"
    Registry    string      // "registry.ollama.ai"
    Platform    string      // "linux/amd64"
    ...
}
```

### After (Generic)

```go
type sessionMeta struct {
    URL           string    // "https://example.com/file.zip"
    Filename      string    // "file.zip"
    ExpectedSize  int64     // From Content-Length
    ...
}
```

## Implementation Roadmap

### MVP (Iteration 1) - Get It Working

1. Backend generic download ✓
2. Basic UI updates ✓
3. Session management ✓
4. Error handling ✓
   **Target:** Basic file downloader with pause/resume

### Manager (Iteration 2) - Make It Pro

5. Queue management
6. Speed tracking + ETA
7. History + Statistics
8. Search, filter, bulk ops
9. Settings panel
10. Basic optimization
    **Target:** Compete with IDM/FDM

### Polish (Iteration 3) - Release Quality

11. Documentation (README, FEATURES, CONTRIBUTING)
12. Error messages with suggestions
13. Performance optimization
14. Edge case handling
15. Security hardening
16. Comprehensive testing
17. CHANGELOG
    **Target:** Professional, ready for public release

### Automation (Iteration 4) - Enterprise Optional

18. Download scheduling with cron
19. Mirror fallback support
20. Batch import
21. CLI improvements
22. Webhook notifications
23. Docker containerization
24. Metrics & health API
25. Example scripts
    **Target:** Enterprise-grade automation features

## Key Technical Decisions

| Decision                       | Why                                   |
| ------------------------------ | ------------------------------------- |
| Keep Range header for resume   | Simple, HTTP standard, reliable       |
| Use .part files for incomplete | No DB needed, easy cleanup            |
| Direct file save (no zipping)  | Simpler, faster, users want raw files |
| SQLite for history (later)     | Better than JSON for queries          |
| Goroutines for concurrency     | Go's strength, lightweight            |
| Simple file naming             | No complex hash/digest logic          |

## Testing Checklist

### MVP Testing

- [ ] Download small file (< 10MB)
- [ ] Download large file (> 1GB)
- [ ] Pause and resume
- [ ] Delete during download
- [ ] Invalid URL handling
- [ ] Low disk space handling
- [ ] Browser reload persistence

### Manager Testing

- [ ] Queue 5+ downloads
- [ ] Pause/resume individual
- [ ] Pause/resume all
- [ ] Reorder queue
- [ ] Search functionality
- [ ] Filter by status
- [ ] Bulk delete
- [ ] History persistence

## Success Criteria Checklist

### MVP ✓ When Achieved

- [ ] Download any file from URL
- [ ] Pause/Resume works
- [ ] Progress bar accurate
- [ ] Survives app restart
- [ ] Error messages helpful
- [ ] Web UI functional

### Manager ✓ When Achieved

- [ ] Multiple downloads simultaneously
- [ ] Speed/ETA display accurate
- [ ] History persists correctly
- [ ] Search/filter works
- [ ] Bulk ops work smoothly
- [ ] Settings save properly
- [ ] Dashboard shows real metrics

## Phase Files

- **Phase 1:** `PHASE1_MVP.md` - Basic file downloader with pause/resume
- **Phase 2:** `PHASE2_MANAGER.md` - Queue, history, speed tracking
- **Phase 3:** `PHASE3_ADVANCED.md` - Optional power-user features
- **Phase 4:** `PHASE4_POLISH.md` - Documentation, testing, optimization
- **Phase 5:** `PHASE5_AUTOMATION.md` - Scheduling, webhooks, Docker (optional)

## Quick Links

- **Conversion Plan:** `CONVERSION_PLAN.md`
- **Full Roadmap:** `FULL_FEATURED_ROADMAP.md`
- **Checklist:** `CHECKLIST.md`
- **Current Branch:** `feature/general-purpose-downloader`
- **Original Branch:** `main`

## Next Steps

1. Read CONVERSION_PLAN.md for detailed breakdown
2. Start Phase 1: Backend conversion
3. Test MVP before moving to Phase 2
4. Get feedback on UI/UX before Phase 2
5. Complete Phase 4 (Polish) for public release
6. Phase 5 (Automation) is optional for enterprise users
