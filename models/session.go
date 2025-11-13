package models

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const sessionMetaFileName = "session.json"

type SessionState string

const (
	StateDownloading SessionState = "downloading"
	StatePaused      SessionState = "paused"
	StateError       SessionState = "error"
	StateReady       SessionState = ""
)

type SessionMeta struct {
	Model       string       `json:"model"`
	SessionID   string       `json:"sessionId"`
	OutZip      string       `json:"outZip"`
	StagingRoot string       `json:"stagingRoot"`
	Registry    string       `json:"registry"`
	Platform    string       `json:"platform"`
	Concurrency int          `json:"concurrency"`
	Retries     int          `json:"retries"`
	StartedAt   time.Time    `json:"startedAt"`
	LastUpdated time.Time    `json:"lastUpdated"`
	State       SessionState `json:"state"`
	Message     string       `json:"message"`
}

type SessionView struct {
	Model      string
	SessionID  string
	Started    string
	Updated    string
	StateLabel string
	Message    string
}

type DownloadEntry struct {
	Name    string
	Model   string
	Path    string
	ModTime time.Time
}

func SessionMetaPath(dir string) string {
	return filepath.Join(dir, sessionMetaFileName)
}

func LoadSessionMeta(dir string) (SessionMeta, error) {
	var meta SessionMeta
	data, err := os.ReadFile(SessionMetaPath(dir))
	if err != nil {
		return meta, err
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return meta, err
	}
	return meta, nil
}

func SaveSessionMeta(meta SessionMeta) error {
	meta.LastUpdated = time.Now()
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(SessionMetaPath(meta.StagingRoot), data, 0o644)
}

func SessionViewFromMeta(meta SessionMeta) SessionView {
	return SessionView{
		Model:      meta.Model,
		SessionID:  meta.SessionID,
		Started:    formatSessionTime(meta.StartedAt),
		Updated:    formatSessionTime(meta.LastUpdated),
		StateLabel: StateLabel(meta.State),
		Message:    meta.Message,
	}
}

func formatSessionTime(t time.Time) string {
	if t.IsZero() {
		return "نامشخص"
	}
	return t.Format("2006-01-02 15:04:05")
}

func StateLabel(state SessionState) string {
	switch strings.ToLower(string(state)) {
	case "downloading":
		return "در حال دانلود"
	case "paused":
		return "مکث شده"
	case "error":
		return "خطا"
	default:
		if state == "" {
			return "در انتظار"
		}
		return string(state)
	}
}

func SetSessionStatus(dir, state string, message string) error {
	if dir == "" {
		return nil
	}
	meta, err := LoadSessionMeta(dir)
	if err != nil {
		return err
	}
	meta.State = SessionState(state)
	meta.Message = message
	return SaveSessionMeta(meta)
}

func DiscoverPartialSessions(outputDir string) ([]SessionMeta, error) {
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return nil, err
	}
	var sessions []SessionMeta
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasSuffix(entry.Name(), ".staging") {
			continue
		}
		meta, err := LoadSessionMeta(filepath.Join(outputDir, entry.Name()))
		if err != nil {
			continue
		}
		sessions = append(sessions, meta)
	}
	return sessions, nil
}

func CategorizeSessions(metas []SessionMeta) (running *SessionView, paused, errored []SessionView) {
	// Sort by last updated time (newest first)
	for i := 0; i < len(metas)-1; i++ {
		for j := i + 1; j < len(metas); j++ {
			if metas[i].LastUpdated.Before(metas[j].LastUpdated) {
				metas[i], metas[j] = metas[j], metas[i]
			}
		}
	}

	for _, meta := range metas {
		view := SessionViewFromMeta(meta)
		switch strings.ToLower(string(meta.State)) {
		case "downloading":
			if running == nil {
				tmp := view
				running = &tmp
			}
		case "paused":
			paused = append(paused, view)
		case "error":
			errored = append(errored, view)
		default:
			paused = append(paused, view)
		}
	}
	return
}

func DownloadsFromDir(dir string) []DownloadEntry {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var downloads []DownloadEntry
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".zip") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		downloads = append(downloads, DownloadEntry{
			Name:    entry.Name(),
			Model:   strings.TrimSuffix(entry.Name(), ".zip"),
			Path:    filepath.Join(dir, entry.Name()),
			ModTime: info.ModTime(),
		})
	}

	// Sort by modification time (newest first)
	for i := 0; i < len(downloads)-1; i++ {
		for j := i + 1; j < len(downloads); j++ {
			if downloads[i].ModTime.Before(downloads[j].ModTime) {
				downloads[i], downloads[j] = downloads[j], downloads[i]
			}
		}
	}
	return downloads
}
