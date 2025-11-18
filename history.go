package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// HistoryEntry represents a completed download record
type HistoryEntry struct {
	ID           string    `json:"id"`
	URL          string    `json:"url"`
	Filename     string    `json:"filename"`
	FileSize     int64     `json:"fileSize"`
	DownloadedAt time.Time `json:"downloadedAt"`
	Duration     int64     `json:"duration"` // seconds
	Speed        int64     `json:"speed"`    // bytes/sec
	Status       string    `json:"status"`   // completed, error
	Error        string    `json:"error,omitempty"`
}

// HistoryManager manages download history
type HistoryManager struct {
	entries []*HistoryEntry
	file    string
	mu      sync.RWMutex
}

// NewHistoryManager creates a new history manager
func NewHistoryManager(dir string) *HistoryManager {
	// Create directory if not exists
	historyDir := filepath.Join(dir, ".history")
	os.MkdirAll(historyDir, 0755)

	hm := &HistoryManager{
		entries: make([]*HistoryEntry, 0),
		file:    filepath.Join(historyDir, "history.json"),
	}

	// Load existing history
	hm.Load()

	return hm
}

// AddEntry adds a new history entry
func (hm *HistoryManager) AddEntry(entry *HistoryEntry) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	hm.entries = append(hm.entries, entry)

	// Save to disk
	return hm.save()
}

// Save persists history to disk
func (hm *HistoryManager) Save() error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	return hm.save()
}

// save is the internal save method (must be called with lock held)
func (hm *HistoryManager) save() error {
	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(hm.entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	// Write to file
	err = os.WriteFile(hm.file, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write history file: %w", err)
	}

	return nil
}

// Load reads history from disk
func (hm *HistoryManager) Load() error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	// Read file
	data, err := os.ReadFile(hm.file)
	if err != nil {
		// Ignore if file doesn't exist
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read history file: %w", err)
	}

	// Unmarshal JSON
	err = json.Unmarshal(data, &hm.entries)
	if err != nil {
		return fmt.Errorf("failed to unmarshal history: %w", err)
	}

	return nil
}

// GetEntries returns all history entries
func (hm *HistoryManager) GetEntries() []*HistoryEntry {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	// Return a copy to avoid data races
	entries := make([]*HistoryEntry, len(hm.entries))
	copy(entries, hm.entries)

	return entries
}

// DeleteEntry removes an entry by ID
func (hm *HistoryManager) DeleteEntry(id string) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	// Find and remove entry
	for i, entry := range hm.entries {
		if entry.ID == id {
			hm.entries = append(hm.entries[:i], hm.entries[i+1:]...)
			return hm.save()
		}
	}

	return fmt.Errorf("entry not found: %s", id)
}

// GetStatistics calculates download statistics
func (hm *HistoryManager) GetStatistics() Statistics {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	stats := Statistics{
		TopDomains:    make(map[string]int64),
		FileTypeStats: make(map[string]int64),
	}

	for _, entry := range hm.entries {
		if entry.Status == "completed" {
			stats.TotalFiles++
			stats.TotalBytes += entry.FileSize
			stats.TotalTime += entry.Duration

			// Check if today
			if isToday(entry.DownloadedAt) {
				stats.TodayFiles++
				stats.TodayBytes += entry.FileSize
			}

			// Extract domain
			domain := extractDomain(entry.URL)
			stats.TopDomains[domain] += entry.FileSize

			// Extract file type
			fileType := extractFileType(entry.Filename)
			stats.FileTypeStats[fileType]++
		}
	}

	// Calculate average speed
	if stats.TotalTime > 0 {
		stats.AverageSpeed = stats.TotalBytes / stats.TotalTime
	}

	return stats
}

// Helper functions

// extractDomain extracts the domain from a URL
func extractDomain(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "unknown"
	}
	return parsed.Host
}

// extractFileType extracts the file extension
func extractFileType(filename string) string {
	ext := filepath.Ext(filename)
	if ext == "" {
		return ".unknown"
	}

	// Handle double extensions like .tar.gz
	if strings.HasSuffix(filename, ".tar.gz") {
		return ".tar.gz"
	}
	if strings.HasSuffix(filename, ".tar.bz2") {
		return ".tar.bz2"
	}
	if strings.HasSuffix(filename, ".tar.xz") {
		return ".tar.xz"
	}

	return ext
}
