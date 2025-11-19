package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Download represents a single download task
type Download struct {
	ID            string     `json:"id"`
	URL           string     `json:"url"`
	Filename      string     `json:"filename"`
	OutputPath    string     `json:"outputPath"`
	Status        string     `json:"status"`
	Priority      int        `json:"priority"`
	Progress      int64      `json:"progress"`
	Total         int64      `json:"total"`
	StartTime     time.Time  `json:"startTime"`
	ResumedAt     *time.Time `json:"resumedAt,omitempty"`
	CompletedTime *time.Time `json:"completedTime,omitempty"`
	Error         string     `json:"error,omitempty"`
	Speed         int64      `json:"speed"`
	ETA           int64      `json:"eta"` // seconds
	Retries       int        `json:"retries"`
	MaxRetries    int        `json:"maxRetries"`

	// Internal fields
	speedTracker  *SpeedTracker
	ctx           context.Context
	cancel        context.CancelFunc
}

// DownloadManager manages download queue and concurrent downloads
type DownloadManager struct {
	downloads     map[string]*Download
	queue         []string
	running       []string
	maxConcurrent int
	mu            sync.RWMutex
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
}

// Statistics represents download statistics
type Statistics struct {
	TotalFiles    int              `json:"totalFiles"`
	TotalBytes    int64            `json:"totalBytes"`
	TotalTime     int64            `json:"totalTime"` // seconds
	AverageSpeed  int64            `json:"averageSpeed"`
	TodayFiles    int              `json:"todayFiles"`
	TodayBytes    int64            `json:"todayBytes"`
	TopDomains    map[string]int64 `json:"topDomains"`
	FileTypeStats map[string]int64 `json:"fileTypeStats"`
}

// Download status constants
const (
	StatusQueued    = "queued"
	StatusActive    = "active"
	StatusPaused    = "paused"
	StatusCompleted = "completed"
	StatusError     = "error"
)

// NewDownloadManager creates a new download manager
func NewDownloadManager(maxConcurrent int) *DownloadManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &DownloadManager{
		downloads:     make(map[string]*Download),
		queue:         make([]string, 0),
		running:       make([]string, 0),
		maxConcurrent: maxConcurrent,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// generateID creates a unique download ID
func generateID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return "dl-" + hex.EncodeToString(bytes)
}

// AddDownload adds a new download to the queue
func (dm *DownloadManager) AddDownload(url, filename, outputPath string, retries int) (string, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	// Validate inputs
	if url == "" {
		return "", fmt.Errorf("URL cannot be empty")
	}
	if filename == "" {
		return "", fmt.Errorf("filename cannot be empty")
	}

	// Generate unique ID
	id := generateID()

	// Create download context
	ctx, cancel := context.WithCancel(dm.ctx)

	// Create download struct
	download := &Download{
		ID:           id,
		URL:          url,
		Filename:     filename,
		OutputPath:   outputPath,
		Status:       StatusQueued,
		Priority:     5, // Default priority
		Progress:     0,
		Total:        0,
		StartTime:    time.Now(),
		MaxRetries:   retries,
		Retries:      0,
		speedTracker: NewSpeedTracker(),
		ctx:          ctx,
		cancel:       cancel,
	}

	// Add to downloads map and queue
	dm.downloads[id] = download
	dm.queue = append(dm.queue, id)

	// Trigger queue processing
	go dm.processQueue()

	return id, nil
}

// RemoveDownload removes a download completely
func (dm *DownloadManager) RemoveDownload(id string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	download, exists := dm.downloads[id]
	if !exists {
		return fmt.Errorf("download not found: %s", id)
	}

	// Cancel if active
	if download.Status == StatusActive {
		download.cancel()
	}

	// Remove from queue
	dm.queue = removeFromSlice(dm.queue, id)
	dm.running = removeFromSlice(dm.running, id)

	// Remove from map
	delete(dm.downloads, id)

	return nil
}

// PauseDownload pauses a specific download
func (dm *DownloadManager) PauseDownload(id string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	download, exists := dm.downloads[id]
	if !exists {
		return fmt.Errorf("download not found: %s", id)
	}

	// Only pause active or queued downloads
	if download.Status != StatusActive && download.Status != StatusQueued {
		return fmt.Errorf("cannot pause download in status: %s", download.Status)
	}

	// Cancel context to stop download
	if download.Status == StatusActive {
		download.cancel()
	}

	download.Status = StatusPaused

	return nil
}

// ResumeDownload resumes a paused download
func (dm *DownloadManager) ResumeDownload(id string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	download, exists := dm.downloads[id]
	if !exists {
		return fmt.Errorf("download not found: %s", id)
	}

	if download.Status != StatusPaused {
		return fmt.Errorf("cannot resume download in status: %s", download.Status)
	}

	// Create new context
	ctx, cancel := context.WithCancel(dm.ctx)
	download.ctx = ctx
	download.cancel = cancel

	// Set status back to queued
	download.Status = StatusQueued
	now := time.Now()
	download.ResumedAt = &now

	// Add back to queue if not already there
	if !containsString(dm.queue, id) {
		dm.queue = append(dm.queue, id)
	}

	// Trigger queue processing
	go dm.processQueue()

	return nil
}

// GetDownload returns a copy of a download
func (dm *DownloadManager) GetDownload(id string) *Download {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	download, exists := dm.downloads[id]
	if !exists {
		return nil
	}

	// Return a copy to avoid data races
	dlCopy := *download
	return &dlCopy
}

// GetAll returns all downloads
func (dm *DownloadManager) GetAll() []*Download {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	downloads := make([]*Download, 0, len(dm.downloads))
	for _, dl := range dm.downloads {
		dlCopy := *dl
		downloads = append(downloads, &dlCopy)
	}

	return downloads
}

// GetByStatus returns downloads matching a specific status
func (dm *DownloadManager) GetByStatus(status string) []*Download {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	downloads := make([]*Download, 0)
	for _, dl := range dm.downloads {
		if dl.Status == status {
			dlCopy := *dl
			downloads = append(downloads, &dlCopy)
		}
	}

	return downloads
}

// PauseAll pauses all active and queued downloads
func (dm *DownloadManager) PauseAll() error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	for _, dl := range dm.downloads {
		if dl.Status == StatusActive || dl.Status == StatusQueued {
			if dl.Status == StatusActive {
				dl.cancel()
			}
			dl.Status = StatusPaused
		}
	}

	return nil
}

// ResumeAll resumes all paused downloads
func (dm *DownloadManager) ResumeAll() error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	for id, dl := range dm.downloads {
		if dl.Status == StatusPaused {
			// Create new context
			ctx, cancel := context.WithCancel(dm.ctx)
			dl.ctx = ctx
			dl.cancel = cancel

			dl.Status = StatusQueued
			now := time.Now()
			dl.ResumedAt = &now

			// Add back to queue if not already there
			if !containsString(dm.queue, id) {
				dm.queue = append(dm.queue, id)
			}
		}
	}

	// Trigger queue processing
	go dm.processQueue()

	return nil
}

// GetStatistics calculates and returns statistics
func (dm *DownloadManager) GetStatistics() Statistics {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	stats := Statistics{
		TopDomains:    make(map[string]int64),
		FileTypeStats: make(map[string]int64),
	}

	for _, dl := range dm.downloads {
		if dl.Status == StatusCompleted {
			stats.TotalFiles++
			stats.TotalBytes += dl.Total

			if dl.CompletedTime != nil {
				duration := dl.CompletedTime.Sub(dl.StartTime)
				stats.TotalTime += int64(duration.Seconds())
			}

			// Check if today
			if isToday(dl.StartTime) {
				stats.TodayFiles++
				stats.TodayBytes += dl.Total
			}
		}
	}

	// Calculate average speed
	if stats.TotalTime > 0 {
		stats.AverageSpeed = stats.TotalBytes / stats.TotalTime
	}

	return stats
}

// Shutdown gracefully shuts down the download manager
func (dm *DownloadManager) Shutdown() {
	dm.cancel()
	dm.wg.Wait()
}

// processQueue processes the download queue
func (dm *DownloadManager) processQueue() {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	// Count currently running
	runningCount := len(dm.running)

	// Process queue
	for len(dm.queue) > 0 && runningCount < dm.maxConcurrent {
		// Get next queued download
		id := dm.queue[0]
		dm.queue = dm.queue[1:]

		download, exists := dm.downloads[id]
		if !exists {
			continue
		}

		// Skip if not queued
		if download.Status != StatusQueued {
			continue
		}

		// Change status to active
		download.Status = StatusActive
		dm.running = append(dm.running, id)
		runningCount++

		// Start worker goroutine
		dm.wg.Add(1)
		go dm.downloadWorker(id)
	}
}

// downloadWorker is the worker goroutine for downloading
func (dm *DownloadManager) downloadWorker(id string) {
	defer dm.wg.Done()

	dm.mu.RLock()
	download, exists := dm.downloads[id]
	dm.mu.RUnlock()

	if !exists {
		return
	}

	// Perform the download
	err := dm.executeDownload(download)

	dm.mu.Lock()

	// Remove from running list
	dm.running = removeFromSlice(dm.running, id)

	if err != nil {
		// Handle error
		download.Retries++
		if download.Retries >= download.MaxRetries {
			download.Status = StatusError
			download.Error = err.Error()
		} else {
			// Retry: add back to queue
			download.Status = StatusQueued
			dm.queue = append(dm.queue, id)
		}
	} else {
		// Success
		download.Status = StatusCompleted
		now := time.Now()
		download.CompletedTime = &now
	}

	dm.mu.Unlock()

	// Process next in queue
	go dm.processQueue()
}

// executeDownload performs the actual download
func (dm *DownloadManager) executeDownload(download *Download) error {
	// Create a progress tracker for this download
	p := &progress{
		total: download.Total,
		done:  download.Progress,
	}

	// Call the downloadFile function from download_generic.go
	// Note: downloadFile expects outputPath to include the filename
	err := downloadFile(download.ctx, download.URL, download.OutputPath, p)

	// Update download progress and total from the progress tracker
	download.Progress = atomic.LoadInt64(&p.done)
	if p.total > 0 {
		download.Total = p.total
	}

	return err
}

// Helper functions

func removeFromSlice(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}

func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func isToday(t time.Time) bool {
	now := time.Now()
	year, month, day := now.Date()
	tyear, tmonth, tday := t.Date()
	return year == tyear && month == tmonth && day == tday
}
