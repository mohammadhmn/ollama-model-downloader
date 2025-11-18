package main

import (
	"fmt"
	"sync"
	"time"
)

// SpeedSample represents a speed measurement sample
type SpeedSample struct {
	timestamp time.Time
	bytes     int64
}

// SpeedTracker tracks download speed and calculates ETA
type SpeedTracker struct {
	samples []*SpeedSample
	mu      sync.RWMutex
}

// NewSpeedTracker creates a new speed tracker
func NewSpeedTracker() *SpeedTracker {
	return &SpeedTracker{
		samples: make([]*SpeedSample, 0, 10),
	}
}

// Record adds a new speed sample
func (st *SpeedTracker) Record(bytes int64) {
	st.mu.Lock()
	defer st.mu.Unlock()

	sample := &SpeedSample{
		timestamp: time.Now(),
		bytes:     bytes,
	}

	st.samples = append(st.samples, sample)

	// Keep only last 10 samples (rolling window)
	if len(st.samples) > 10 {
		st.samples = st.samples[1:]
	}
}

// GetSpeed returns current speed in bytes per second
func (st *SpeedTracker) GetSpeed() int64 {
	st.mu.RLock()
	defer st.mu.RUnlock()

	// Need at least 2 samples to calculate speed
	if len(st.samples) < 2 {
		return 0
	}

	first := st.samples[0]
	last := st.samples[len(st.samples)-1]

	// Calculate time delta in seconds
	timeDelta := last.timestamp.Sub(first.timestamp).Seconds()
	if timeDelta == 0 {
		return 0
	}

	// Calculate bytes transferred
	bytesDelta := last.bytes - first.bytes

	// Calculate speed (bytes per second)
	speed := int64(float64(bytesDelta) / timeDelta)

	return speed
}

// GetAverageSpeed returns moving average speed for smoother updates
func (st *SpeedTracker) GetAverageSpeed() int64 {
	st.mu.RLock()
	defer st.mu.RUnlock()

	if len(st.samples) < 2 {
		return 0
	}

	// Calculate speed for each consecutive pair
	var totalSpeed int64
	var count int

	for i := 1; i < len(st.samples); i++ {
		prev := st.samples[i-1]
		curr := st.samples[i]

		timeDelta := curr.timestamp.Sub(prev.timestamp).Seconds()
		if timeDelta == 0 {
			continue
		}

		bytesDelta := curr.bytes - prev.bytes
		speed := int64(float64(bytesDelta) / timeDelta)

		totalSpeed += speed
		count++
	}

	if count == 0 {
		return 0
	}

	return totalSpeed / int64(count)
}

// GetETA calculates estimated time to completion
func (st *SpeedTracker) GetETA(total, downloaded int64) time.Duration {
	speed := st.GetSpeed()

	if speed == 0 || total == 0 {
		return 0
	}

	remaining := total - downloaded
	if remaining <= 0 {
		return 0
	}

	seconds := float64(remaining) / float64(speed)
	return time.Duration(seconds) * time.Second
}

// FormatSpeed formats speed for display
func FormatSpeed(bytesPerSecond int64) string {
	if bytesPerSecond < 1024 {
		return fmt.Sprintf("%d B/s", bytesPerSecond)
	} else if bytesPerSecond < 1024*1024 {
		return fmt.Sprintf("%.1f KB/s", float64(bytesPerSecond)/1024)
	} else if bytesPerSecond < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB/s", float64(bytesPerSecond)/(1024*1024))
	} else {
		return fmt.Sprintf("%.2f GB/s", float64(bytesPerSecond)/(1024*1024*1024))
	}
}

// FormatDuration formats duration for display
func FormatDuration(d time.Duration) string {
	if d == 0 {
		return "calculating..."
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	} else {
		return fmt.Sprintf("%ds", seconds)
	}
}

// FormatSize formats bytes for display
func FormatSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	} else if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	} else {
		return fmt.Sprintf("%.2f GB", float64(bytes)/(1024*1024*1024))
	}
}
