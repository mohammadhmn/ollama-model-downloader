package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// downloadFile downloads a file from a URL with resume support via Range headers
func downloadFile(ctx context.Context, downloadURL, outputPath string, p *progress) error {
	// Validate URL first
	if err := validateURL(downloadURL); err != nil {
		return err
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}

	// Check for existing .part file
	partPath := outputPath + ".part"
	offset := int64(0)
	if info, err := os.Stat(partPath); err == nil && info.Size() > 0 {
		offset = info.Size()
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return err
	}

	// Add Range header if resuming
	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Handle response codes
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("download failed: %s (HTTP %d)", string(body), resp.StatusCode)
	}

	// Update progress total if we have Content-Length
	if p != nil {
		contentLength := resp.ContentLength
		if contentLength > 0 {
			if offset > 0 && resp.StatusCode == http.StatusPartialContent {
				// Resuming: adjust total to remaining bytes
				p.total = offset + contentLength
			} else {
				p.total = contentLength
				offset = 0
			}
		}
	}

	// Determine file open flags
	flags := os.O_CREATE | os.O_WRONLY
	if offset > 0 && resp.StatusCode == http.StatusPartialContent {
		flags = os.O_APPEND | os.O_WRONLY
	}

	// Open file for writing
	f, err := os.OpenFile(partPath, flags, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	// If full download (not resuming), truncate
	if offset == 0 && resp.StatusCode == http.StatusOK {
		if err := f.Truncate(0); err != nil {
			return err
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return err
		}
	}

	// Stream download with progress tracking
	var writers []io.Writer
	writers = append(writers, f)
	if p != nil {
		writers = append(writers, p)
	}

	if _, err := io.Copy(io.MultiWriter(writers...), resp.Body); err != nil {
		return err
	}

	// Close file before renaming
	f.Close()

	// Rename .part to final filename on success
	if err := os.Rename(partPath, outputPath); err != nil {
		return err
	}

	return nil
}

// extractFilenameFromURL extracts filename from URL path
func extractFilenameFromURL(urlStr string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return "download"
	}

	path := u.Path
	if path == "" || path == "/" {
		// No path or root path
		if u.Host != "" {
			return u.Host
		}
		return "download"
	}

	// Get last path component
	filename := filepath.Base(path)
	if filename == "" || filename == "." {
		return "download"
	}

	// Remove query parameters if any made it into filename
	if idx := strings.Index(filename, "?"); idx >= 0 {
		filename = filename[:idx]
	}

	// Sanitize filename
	filename = sanitizeFilename(filename)
	if filename == "" {
		return "download"
	}

	return filename
}

// sanitizeFilename removes invalid characters from filename
func sanitizeFilename(filename string) string {
	filename = strings.Map(func(r rune) rune {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'):
			return r
		case r == '.' || r == '-' || r == '_':
			return r
		default:
			return -1 // remove character
		}
	}, filename)
	return strings.TrimSpace(filename)
}

// validateURL validates URL format and scheme
func validateURL(urlStr string) error {
	u, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	// Verify HTTP/HTTPS scheme only
	if u.Scheme == "" {
		return errors.New("URL must include scheme (http:// or https://)")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme: %s (only http and https allowed)", u.Scheme)
	}

	// Check host is present
	if u.Host == "" {
		return errors.New("URL must include host")
	}

	return nil
}
