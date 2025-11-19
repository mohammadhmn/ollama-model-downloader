# Architecture Overview: Multi-Phase Conversion

## Phase Progression Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Ollama Model Downloader                          │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  Backend: OCI Registry Integration                          │   │
│  │  - parseModel() → Ollama model parsing                       │   │
│  │  - getRegistryToken() → Bearer authentication               │   │
│  │  - getManifestOrIndex() → OCI manifest fetching             │   │
│  │  - downloadBlob() → Digest-based blob downloads            │   │
│  │  - Staging: models/blobs/manifests structure               │   │
│  ├──────────────────────────────────────────────────────────────┤   │
│  │  UI: Ollama-specific                                         │   │
│  │  - Input: Model name (llama3.2, mistral, etc)               │   │
│  │  - Options: Platform, Registry, Concurrency                │   │
│  │  - Actions: Download, Pause, Resume, Unzip                │   │
│  └──────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
                                  ↓
                           ┌─ Phase 1 ─┐
                           │ MVP: Basic │
                           └────────────┘
                                  ↓
┌─────────────────────────────────────────────────────────────────────┐
│              General Purpose File Downloader (MVP)                   │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  Backend: Generic HTTP Downloads                            │   │
│  │  - downloadFile(url) → Simple HTTP GET with streaming       │   │
│  │  - validateURL() → Basic URL validation                     │   │
│  │  - extractFilename() → Parse from URL/headers               │   │
│  │  - Resume with Range headers                                │   │
│  │  - Simple progress tracking (bytes/total)                   │   │
│  │  - .part files for incomplete downloads                     │   │
│  ├──────────────────────────────────────────────────────────────┤   │
│  │  UI: Generic                                                 │   │
│  │  - Input: Download URL                                       │   │
│  │  - Options: Retries, Output directory                       │   │
│  │  - Actions: Download, Pause, Resume, Cancel                │   │
│  │  - Single file download at a time                           │   │
│  └──────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
                                  ↓
                        ┌─ Phase 2.5 ─┐
                        │ Manager:     │
                        │ Multi-file   │
                        └──────────────┘
                                  ↓
┌─────────────────────────────────────────────────────────────────────┐
│          Full-Featured Download Manager                             │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  Backend: Professional Download Management                  │   │
│  │  - DownloadQueue → Manage multiple downloads               │   │
│  │  - SpeedTracker → Real-time speed/ETA calculation          │   │
│  │  - History DB → Persistent download history & stats        │   │
│  │  - BandwidthLimiter → Speed cap support                    │   │
│  │  - Concurrent downloads (configurable 1-32)                │   │
│  │  - Priority queue system                                    │   │
│  ├──────────────────────────────────────────────────────────────┤   │
│  │  UI: Professional                                            │   │
│  │  - 4 tabs: Active | Queue | Completed | History             │   │
│  │  - Speed/ETA display                                         │   │
│  │  - Bulk operations (pause all, delete selected, etc)        │   │
│  │  - Advanced search & filtering                              │   │
│  │  - Statistics dashboard                                     │   │
│  │  - Settings panel                                           │   │
│  │  - Keyboard shortcuts                                       │   │
│  │  - Dark/Light theme                                         │   │
│  │  - Drag & drop support                                      │   │
│  └──────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Data Flow Architecture

### Phase 1: MVP

```
                        User
                         ↓
                  Web Form / CLI
                         ↓
                    Validate URL
                         ↓
                  Extract Filename
                         ↓
                  HTTP GET Request
                    (with Range)
                         ↓
        ┌────────────────┼────────────────┐
        ↓                ↓                ↓
    Streaming       Progress         Session
    to .part        Tracking         Metadata
    file            (bytes)          (.json)
        ↓                ↓                ↓
        └────────────────┼────────────────┘
                         ↓
                  Verify Checksum
                         ↓
                  Rename to Final
                         ↓
                  Update Session
                         ↓
                   UI Update
                         ↓
                   User Views Result
```

### Phase 2.5: Manager

```
                        User
                         ↓
              Web Form / Batch Input
                         ↓
            ┌─────────────┴─────────────┐
            ↓                           ↓
       Validate URL              Add to Queue
            ↓                           ↓
    Extract Filename         Save Session Meta
            ↓                           ↓
         Queue Db              Show in "Queue" Tab
            ↓                           ↓
       Process Queue ←──────────────────┘
            ↓
     Download Manager
            ↓
    ┌─────────────────────────┐
    ↓         ↓         ↓     ↓
  DL-1      DL-2      DL-3  ...
  Active   Queued    Queued  ...
    ↓         ↓         ↓     ↓
 SpeedTracker (all downloads)
    ↓
 History DB
    ↓
Real-time Progress Updates → UI
    ↓
Tab Management (Active/Queue/Completed/History)
    ↓
User Actions (Pause/Resume/Delete/Bulk Ops)
```

---

## Component Interaction Diagram

### MVP Components

```
┌──────────────────────────────────────────────────────────┐
│                     HTTP Server                           │
│  (main.go)                                                │
│  ┌────────────────────────────────────────────────────┐  │
│  │ Routes:                                            │  │
│  │  /          → Show UI                             │  │
│  │  /download  → Start new download (POST)           │  │
│  │  /progress  → Get progress JSON                   │  │
│  │  /pause     → Pause current download              │  │
│  │  /resume    → Resume paused download              │  │
│  │  /cancel    → Cancel download                     │  │
│  └────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────┘
                         ↓
        ┌────────────────────────────────┐
        ↓                                 ↓
    ┌─────────────┐          ┌────────────────────┐
    │  Downloader │          │  Session Manager   │
    │ (download.go)          │  (main.go)         │
    │             │          │                    │
    │ Functions:  │          │ Functions:         │
    │ • Download  │          │ • Load Meta        │
    │   File      │          │ • Save Meta        │
    │ • Validate  │          │ • Status Updates   │
    │   URL       │          │ • Persistence      │
    │ • Track     │          │                    │
    │   Progress  │          │                    │
    │ • Resume    │          │                    │
    │   Support   │          │                    │
    └─────────────┘          └────────────────────┘
        ↓                                 ↓
    ┌─────────────────────────────────────────────┐
    │  File System                                │
    │  ├── downloaded-files/                      │
    │  │   ├── file.zip                          │
    │  │   ├── file.zip.part (incomplete)        │
    │  │   └── .staging/                         │
    │  │       └── session.json                  │
    │  └── .history/ (Phase 2.5)                 │
    └─────────────────────────────────────────────┘
```

### Manager Components (Phase 2.5)

```
┌──────────────────────────────────────────────────────────┐
│                     HTTP Server                           │
│  (main.go)                                                │
│  ┌────────────────────────────────────────────────────┐  │
│  │ API Routes:                                        │  │
│  │  /api/download/add         → Add to queue         │  │
│  │  /api/downloads            → Get all downloads    │  │
│  │  /api/download/{id}/pause  → Pause specific      │  │
│  │  /api/download/{id}/resume → Resume specific     │  │
│  │  /api/download/{id}/cancel → Cancel specific     │  │
│  │  /api/statistics           → Get stats            │  │
│  │  /api/history              → Get history         │  │
│  └────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────┘
         ↓              ↓              ↓              ↓
    ┌─────────┐  ┌──────────────┐  ┌────────────┐  ┌──────────┐
    │Download │  │    Speed     │  │  Download │  │  History │
    │ Manager │  │   Tracker    │  │  Database │  │  Manager │
    │         │  │              │  │           │  │          │
    │ Queue:  │  │ Real-time:   │  │ SQLite:   │  │ Tracks:  │
    │ • Add   │  │ • Speed      │  │ • All DLs │  │ • All    │
    │ • Order │  │   (KB/s)     │  │ • Stats   │  │   DLs    │
    │ • Pause │  │ • ETA        │  │ • Dates   │  │ • Stats  │
    │ • Resume│  │ • Samples    │  │ • Sizes   │  │ • Metrics│
    │ • Delete│  │   (10-sec)   │  │ • Sources │  │          │
    │         │  │              │  │           │  │          │
    │Concur:  │  │Smoothing:    │  │Queries:   │  │Export:   │
    │ • Max   │  │ • 30% weight │  │ • Filter  │  │ • JSON   │
    │   (1-32)   │   new sample │  │ • Search  │  │ • CSV    │
    │ • Start │  │ • 70% weight │  │ • Sort    │  │          │
    │   next  │  │   old avg    │  │ • Agg     │  │          │
    └─────────┘  └──────────────┘  └────────────┘  └──────────┘
         ↓              ↓              ↓              ↓
         └──────────────┴──────────────┴──────────────┘
                         ↓
          ┌──────────────────────────────┐
          │  Session Manager             │
          │  Per-Download Metadata       │
          └──────────────────────────────┘
                         ↓
         ┌───────────────────────────────┐
         │  File System                  │
         │  ├── downloaded-files/        │
         │  │   ├── file1.zip            │
         │  │   ├── file2.mp4            │
         │  │   ├── partial.iso.part    │
         │  │   └── .staging/            │
         │  │       ├── dl-1/            │
         │  │       ├── dl-2/            │
         │  │       └── session.json     │
         │  └── .history/                │
         │      ├── downloads.db         │
         │      └── statistics.json      │
         └───────────────────────────────┘
```

---

## State Machine Diagrams

### Single Download State Machine (MVP)

```
    ┌─────────────┐
    │   Created   │
    └──────┬──────┘
           ↓
    ┌─────────────┐
    │ Validating  │
    └──┬──────┬───┘
       │      └──────→ [Invalid] → Error
       ↓
    ┌──────────────┐
    │ Downloading  │
    └──┬──────┬────┘
       │      │
    Pause   Cancel
       │      │
       ↓      └─→ Cancelled
    ┌──────────┐
    │  Paused  │
    └──┬──────┬┘
       │      │
    Resume Cancel
       │      │
       ↓      └─→ Cancelled
       [back to Downloading]

    Downloading ──→ [Complete] → Completed
              ──→ [Error] → Error → [Retry?] → Downloading
```

### Queue Processing State Machine (Manager)

```
User Input
    ↓
┌──────────────┐
│   Validate   │
└──┬────────┬──┘
   │        └─→ Invalid → Error
   ↓
┌──────────────┐
│   Queued     │ ← Waiting in queue
└──┬────────┬──┘
   │        └─→ [Too many active] → Wait
   ↓
┌──────────────────┐
│   Active/Active  │ ← Actually downloading
└──┬────────┬──────┘
   │        │
Pause    Complete/Error
   │        │
   ↓        ├─→ Completed
┌──────────┐
│  Paused  │ ← Can resume
└──┬───┬───┘
   │   └─→ Delete → Removed
   │
Resume or User Action
   │
   ↓
Back to Queued
   ↓
Back to Active
   ↓
Complete/Error
```

---

## File Organization After Conversion

### MVP Phase

```
ollama-model-downloader/
├── main.go                 (simplified, generic handlers)
├── download.go             (generic HTTP download logic)
├── templates/
│   └── index.html          (updated UI)
├── downloaded-files/       (output directory)
│   └── .staging/           (session metadata)
├── README.md               (updated documentation)
└── CONVERSION_PLAN.md      (this plan)
```

### Manager Phase

```
ollama-model-downloader/
├── main.go                 (updated with API routes)
├── download.go             (core download logic)
├── download_manager.go     (queue & concurrency)
├── speed_tracker.go        (speed & ETA)
├── history.go              (history & statistics)
├── templates/
│   └── index.html          (full-featured UI)
├── downloaded-files/       (output directory)
│   ├── .staging/           (per-download sessions)
│   └── .history/           (persistent history)
├── internal/               (if needed)
│   ├── models/             (data structures)
│   └── utils/              (helpers)
└── README.md               (comprehensive docs)
```

---

## Performance Targets

### MVP

- Single download: Minimal overhead
- Progress updates: 1 per second
- Memory: < 50 MB
- CPU: < 5% idle

### Manager

- Multiple downloads: Negligible per-download overhead
- Progress updates: 1 per second per download
- Memory: < 200 MB (with 10+ downloads)
- CPU: < 10% with concurrent downloads
- Speed calculation latency: < 100ms
- Database queries: < 50ms for typical filters

---

## Security Considerations

```
├── Input Validation
│   ├── URL format validation
│   ├── Path traversal prevention
│   └── Filename sanitization
│
├── File Operations
│   ├── Permissions check
│   ├── Disk space verification
│   └── Overwrite confirmation
│
├── Network
│   ├── HTTPS by default
│   ├── SSL/TLS verification (configurable)
│   └── Timeout protection
│
└── Storage
    ├── Session file encryption (future)
    ├── History database access control
    └── Temporary file cleanup
```

---

## Dependencies (Minimal for MVP)

```
stdlib only:
- net/http
- context
- sync/atomic
- io
- os
- time
- html/template

Optional for Manager:
- database/sql + sqlite3 (history)
- encoding/json (already used)
```

This keeps the application lightweight and deployment-simple.
