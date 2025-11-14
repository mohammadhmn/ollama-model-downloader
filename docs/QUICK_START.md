# Quick Start Guide

Get started converting Ollama Model Downloader to a full-featured download manager in 3-4 days.

---

## Overview

**Total Effort**: 40-50 hours across 4 phases

| Phase                 | Duration  | Tasks    | Status         |
| --------------------- | --------- | -------- | -------------- |
| **Phase 1: MVP**      | 10-15 hrs | 16 tasks | ⬜ Not Started |
| **Phase 2: Manager**  | 20-25 hrs | 27 tasks | ⬜ Not Started |
| **Phase 3: Advanced** | 8-10 hrs  | 10 tasks | ⬜ Not Started |
| **Phase 4: Polish**   | 3-4 hrs   | 7 tasks  | ⬜ Not Started |

**Recommended Approach**: Complete Phase 1 MVP in days 1-2, then Phase 2 Manager in days 3-4. Phases 3-4 are optional.

---

## Day 1: Backend Foundation (Phase 1, Part 1)

**Goal**: Create generic download logic, remove Ollama code

### Morning (3-4 hours)

1. **Create `download_generic.go`** (2-3 hours)

   - [ ] Create new file
   - [ ] Implement `downloadFile()` with streaming
   - [ ] Implement `extractFilenameFromURL()`
   - [ ] Implement `validateURL()`
   - [ ] Support Range headers for resume
   - [ ] Handle `.part` files
   - [ ] Test compilation

2. **Remove Ollama code** (1-2 hours)
   - [ ] Delete from `download.go`:
     - `parseModel()`
     - `getRegistryToken()`
     - `getManifestOrIndex()`
     - `downloadBlob()`
     - OCI/Docker constants
     - Bearer auth logic
   - [ ] Search for "ollama", "registry", "OCI" - remove all
   - [ ] Test compilation

### Afternoon (2-3 hours)

3. **Update data structures** (30 mins)

   - [ ] Update `sessionMeta` struct in `main.go`
   - [ ] Add: `URL`, `Filename`, `ExpectedSize`
   - [ ] Remove: `Registry`, `Platform`, `Concurrency`

4. **CLI updates** (1-1.5 hours)

   - [ ] Update flag definitions - remove registry/platform
   - [ ] Accept URL as positional argument
   - [ ] Support both CLI and web modes
   - [ ] Test: `./downloader https://example.com/file.zip`

5. **Quick test** (30 mins)
   - [ ] Build and run web server
   - [ ] Test downloading small file
   - [ ] Verify progress tracking
   - [ ] Test pause/resume

**End of Day 1 Status**:

- ✅ Generic HTTP download working
- ✅ Ollama code removed
- ✅ CLI accepts URLs
- ⏳ UI not updated yet

---

## Day 2: UI Updates (Phase 1, Part 2)

**Goal**: Update web interface, finalize MVP

### Morning (2-3 hours)

1. **Update HTML form** (1 hour)

   - [ ] Replace "model name" with "URL" input
   - [ ] Add optional "filename" input
   - [ ] Remove platform/registry/concurrency fields
   - [ ] Update Persian labels
   - [ ] Test form renders correctly

2. **Update download handler** (1-1.5 hours)
   - [ ] Update `/download` HTTP handler
   - [ ] Parse `url` instead of `model`
   - [ ] Parse optional `filename`
   - [ ] Call `validateURL()`
   - [ ] Extract filename if needed
   - [ ] Test form submission works

### Afternoon (2-3 hours)

3. **Comprehensive testing** (2-3 hours)

   - [ ] Test 1: Download small file (100MB)
     - Progress updates ✅
     - Pause works ✅
     - Resume works ✅
   - [ ] Test 2: Download large file (1GB+)
     - Resume after interrupt ✅
     - File correct size ✅
   - [ ] Test 3: Error handling
     - Invalid URL rejected ✅
     - 404 handled ✅
     - Timeout handled ✅
   - [ ] Test 4: Session persistence
     - Restart app ✅
     - Resume download ✅
   - [ ] Test 5: CLI mode
     - Works with URLs ✅
     - Works with different file types ✅

4. **Cleanup & commit** (30 mins)
   - [ ] Remove debug logs
   - [ ] Code cleanup (gofmt)
   - [ ] Git commit MVP

**End of Day 2 Status**:

- ✅ MVP complete and tested
- ✅ Single file downloads work
- ✅ Pause/resume functional
- ✅ Session persistence works
- ⏳ Multi-download not supported yet

**Commit Message**:

```
feat: implement MVP generic file downloader

- Remove all Ollama-specific logic
- Add generic HTTP/HTTPS download with resume
- Update sessionMeta for generic downloads
- Update CLI and web UI for URLs
- Update Persian labels
- Comprehensive testing across file types
```

---

## Day 3: Queue Manager (Phase 2, Part 1)

**Goal**: Add multi-download support with queue management

### Morning (3-4 hours)

1. **Create `download_manager.go`** (2-3 hours)

   - [ ] Create `DownloadManager` struct
   - [ ] Create `Download` struct with full metadata
   - [ ] Implement `AddDownload()` method
   - [ ] Implement `RemoveDownload()` method
   - [ ] Implement `PauseDownload()` method
   - [ ] Implement `ResumeDownload()` method
   - [ ] Add thread-safety with RWMutex
   - [ ] Test compilation

2. **Implement worker pool** (1-2 hours)
   - [ ] Implement `ProcessQueue()` method
   - [ ] Create `downloadWorker()` goroutine
   - [ ] Implement concurrent limit (maxConcurrent)
   - [ ] Auto-start next when one completes
   - [ ] Test: Add 5 downloads, verify 4 concurrent max

### Afternoon (2-3 hours)

3. **Create speed and history managers** (2-3 hours)

   - [ ] Create `speed_tracker.go`

     - `SpeedTracker` struct
     - `Record()` method
     - `GetSpeed()` method
     - `GetETA()` method

   - [ ] Create `history.go`

     - `HistoryManager` struct
     - `HistoryEntry` struct
     - `Load()` and `Save()` methods
     - `AddEntry()` method
     - `GetStatistics()` method

   - [ ] Test: Save history, restart, load

**End of Day 3 Status**:

- ✅ Queue manager working
- ✅ Concurrent downloads limited properly
- ✅ Speed calculation working
- ✅ History persistence working
- ⏳ UI not updated yet

---

## Day 4: Manager UI & API (Phase 2, Part 2)

**Goal**: Add API endpoints and tab-based UI

### Morning (3-4 hours)

1. **Add API endpoints** (2-3 hours)

   - [ ] `GET /api/downloads` - list all
   - [ ] `POST /api/download/add` - add new
   - [ ] `POST /api/download/{id}/pause` - pause
   - [ ] `POST /api/download/{id}/resume` - resume
   - [ ] `POST /api/downloads/pause-all` - pause all
   - [ ] `POST /api/downloads/resume-all` - resume all
   - [ ] `GET /api/statistics` - get stats
   - [ ] Test with curl

2. **Create tab structure in UI** (1-1.5 hours)
   - [ ] Add tab navigation (Active, Queue, Completed, History)
   - [ ] Create tab content areas
   - [ ] Implement tab switching
   - [ ] Add badge counts
   - [ ] Test tabs work

### Afternoon (2-3 hours)

3. **Implement tab content** (2-3 hours)

   - [ ] Active tab: showing speed/ETA, pause/resume buttons
   - [ ] Queue tab: list, drag-to-reorder, priority selector
   - [ ] Completed tab: file info, open/delete buttons
   - [ ] History tab: search, sort, statistics widget
   - [ ] Test: add downloads, see them move through tabs

4. **Add bulk operations** (1 hour)

   - [ ] Checkboxes on each download
   - [ ] Select all checkbox
   - [ ] Bulk action buttons (pause all selected, delete, etc)
   - [ ] Test: select multiple, pause/delete them

5. **Final testing** (1 hour)
   - [ ] Add 5 downloads, verify queue
   - [ ] Verify max concurrent (4) respected
   - [ ] Complete downloads, see move to completed tab
   - [ ] Pause/resume all
   - [ ] Check history persists

**End of Day 4 Status**:

- ✅ Full download manager operational
- ✅ Multi-download with queue
- ✅ Speed and ETA tracking
- ✅ History and statistics
- ✅ Advanced UI with tabs
- ✅ Bulk operations

**Commit Message**:

```
feat: implement full-featured download manager

- Add DownloadManager for queue handling
- Implement SpeedTracker for real-time metrics
- Create HistoryManager for persistent records
- Support multiple concurrent downloads
- Add tabs: Active, Queue, Completed, History
- Implement bulk operations
- Add comprehensive statistics
- Add API endpoints for all operations
- Full testing and validation
```

---

## Optional: Day 5+ - Advanced Features (Phases 3-4)

If time allows, add optional features:

### Easy (1-2 hours each)

- [ ] Keyboard shortcuts (Ctrl+P for pause, Ctrl+N for new, etc)
- [ ] Custom headers support (for authentication)
- [ ] Bandwidth limiting
- [ ] Batch URL input (paste multiple URLs)

### Medium (2-3 hours each)

- [ ] Drag & drop URL input
- [ ] Dark/Light theme toggle
- [ ] Settings panel
- [ ] Advanced filtering (by type, date range, etc)

### Advanced (3+ hours each)

- [ ] Statistics dashboard with charts
- [ ] Desktop notifications
- [ ] Browser integration
- [ ] Mobile-responsive design

---

## Testing Checklist

### After MVP (Day 2)

- [ ] Download small file (100MB)
- [ ] Download large file (1GB+)
- [ ] Resume interrupted download
- [ ] Pause/resume works
- [ ] Invalid URL rejected
- [ ] 404 handled gracefully
- [ ] Session persists after restart
- [ ] CLI mode works
- [ ] Web UI responsive

### After Manager (Day 4)

- [ ] 5 downloads queued correctly
- [ ] Only 4 download concurrently
- [ ] Speed tracking accurate
- [ ] ETA updates in real-time
- [ ] History saves to file
- [ ] Can search history
- [ ] Bulk pause/resume works
- [ ] Bulk delete works
- [ ] Tab switching smooth
- [ ] Badge counts update
- [ ] Statistics calculated correctly

---

## File Modifications Summary

### Create New Files

```
download_generic.go      - Generic HTTP download
download_manager.go      - Queue and concurrency
speed_tracker.go         - Speed and ETA calculation
history.go               - Download history persistence
```

### Modify Existing Files

```
main.go                  - CLI flags, handlers, manager init
download.go              - Remove Ollama code, keep progress
templates/index.html     - Update form, add tabs, add API calls
```

### No Changes

```
README.md                - Update later in Phase 4
go.mod, go.sum          - No changes needed
Dockerfile              - No changes needed
```

---

## Build & Run Commands

```bash
# Build
go build -o downloader

# Run web server (port 8080)
./downloader -port 8080

# Run CLI mode
./downloader https://example.com/file.zip

# Run with output filename
./downloader https://example.com/file.zip -o myfile.zip

# Run with retries
./downloader https://example.com/file.zip -retries 5

# Run with verbose logging
./downloader -v -port 8080
```

---

## API Quick Reference

```bash
# Add download
curl -X POST http://localhost:8080/api/download/add \
  -d "url=https://example.com/file.zip"

# Get all downloads
curl http://localhost:8080/api/downloads | jq

# Pause specific
curl -X POST http://localhost:8080/api/download/{id}/pause

# Resume specific
curl -X POST http://localhost:8080/api/download/{id}/resume

# Get statistics
curl http://localhost:8080/api/statistics | jq

# Get history
curl http://localhost:8080/api/history | jq
```

---

## Git Workflow

```bash
# Create feature branch
git checkout -b feature/general-purpose-downloader

# After MVP (Day 2)
git add -A
git commit -m "feat: implement MVP generic file downloader"

# After Manager (Day 4)
git add -A
git commit -m "feat: implement full-featured download manager"

# When ready to merge
git checkout main
git merge feature/general-purpose-downloader
```

---

## Troubleshooting

**Build fails**

```bash
go mod tidy
go build -v
```

**No downloads showing**

- Check browser console for JS errors
- Check server logs for API errors
- Verify API endpoints return JSON

**Pause not working**

- Check if download is truly active (not yet started)
- Verify .part file is created
- Check error logs

**History not saving**

- Check if .history directory exists: `ls -la downloaded-files/.history/`
- Check history.json is valid: `cat downloaded-files/.history/history.json | jq`
- Check file permissions

**Speed shows 0**

- Need at least 2 speed samples (takes ~2 seconds)
- Very fast downloads may have limited precision

---

## Performance Tips

**For large queues (100+ downloads)**:

- Consider SQLite for history instead of JSON
- Add pagination to UI
- Implement lazy loading for history tab

**For large files (10GB+)**:

- Ensure .part files created in correct location
- Monitor disk space
- Consider streaming progress updates instead of polling

**For slow connections**:

- Pause other apps to free bandwidth
- Increase retry count
- Use bandwidth limiting feature (Phase 3)

---

## Next Steps After MVP

1. **Get MVP working** (Days 1-2)
2. **Add queue manager** (Days 3-4)
3. **Choose optional features** to add from Phase 3-4
4. **Polish and document** (Phase 4)
5. **Release or share** with users

---

## Useful References

- See `PHASE1_MVP.md` for detailed Phase 1 tasks
- See `PHASE2_MANAGER.md` for detailed Phase 2 tasks
- See `PHASE3_ADVANCED.md` for optional features
- See `PHASE4_POLISH.md` for final polish
- See `CHECKLIST.md` for quick checkboxes
- See `CONVERSION_PLAN.md` for architecture
- See `FULL_FEATURED_ROADMAP.md` for complete vision

---

## Success Indicators

**Day 2 (MVP Done)**:

- You can download any HTTP/HTTPS file ✅
- Progress bar updates in real-time ✅
- Pause and resume works ✅
- Sessions persist after restart ✅
- No Ollama code remains ✅

**Day 4 (Manager Done)**:

- Multiple downloads queue correctly ✅
- Speed and ETA display accurately ✅
- History saves and loads ✅
- Statistics calculate correctly ✅
- Bulk operations work smoothly ✅
- Tabs switch without lag ✅

---

## Time Estimates by Component

| Component            | Time       | Difficulty |
| -------------------- | ---------- | ---------- |
| Remove Ollama code   | 1-2h       | Easy       |
| Add generic download | 2-3h       | Easy       |
| CLI updates          | 1h         | Easy       |
| UI form updates      | 1-2h       | Easy       |
| MVP testing          | 2-3h       | Medium     |
| Queue manager        | 2-3h       | Medium     |
| Speed tracker        | 1h         | Easy       |
| History manager      | 2h         | Easy       |
| API endpoints        | 2h         | Medium     |
| Tab UI               | 3-4h       | Medium     |
| Manager testing      | 2-3h       | Medium     |
| **Total MVP**        | **10-15h** |            |
| **Total Manager**    | **20-25h** |            |
| **Grand Total**      | **30-40h** |            |

---

## Questions?

Refer to detailed docs:

- Architecture questions → `CONVERSION_PLAN.md`
- Implementation details → `IMPLEMENTATION_STEPS.md`
- Feature roadmap → `FULL_FEATURED_ROADMAP.md`
- Task checklists → `CHECKLIST.md`

---

**Ready to start?** Begin with Phase 1, Day 1, Task 1: Create `download_generic.go`

Good luck! 🚀
