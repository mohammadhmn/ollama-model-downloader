package web

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strconv"
	"strings"

	"ollama-model-downloader/internal/errors"
)

type Server struct {
	template     *template.Template
	downloadsDir string
}

func NewServer(templateFS fs.FS, downloadsDir string) (*Server, error) {
	funcMap := template.FuncMap{
		"contains": strings.Contains,
		"js":       js,
	}

	tmpl, err := template.New("index.html").Funcs(funcMap).ParseFS(templateFS, "templates/index.html")
	if err != nil {
		return nil, fmt.Errorf("error parsing template: %w", err)
	}

	return &Server{
		template:     tmpl,
		downloadsDir: downloadsDir,
	}, nil
}

func (s *Server) SetupRoutes() {
	http.HandleFunc("/", s.handleIndex)
	http.HandleFunc("/download", s.handleDownload)
	http.HandleFunc("/model/action", s.handleModelAction)
	http.HandleFunc("/resume", s.handleResume)
	http.HandleFunc("/download/", s.handleFileDownload)
	http.HandleFunc("/progress", s.handleProgress)
	http.HandleFunc("/cancel", s.handleCancel)
	http.HandleFunc("/pause", s.handlePause)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errors.BadRequest("Method not allowed", nil).WriteHTTPResponse(w)
		return
	}

	data := s.getPageData()
	if err := s.template.Execute(w, data); err != nil {
		errors.InternalServerError("Template execution error", err).WriteHTTPResponse(w)
	}
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errors.BadRequest("Method not allowed", nil).WriteHTTPResponse(w)
		return
	}

	if err := r.ParseForm(); err != nil {
		errors.BadRequest("Bad request", err).WriteHTTPResponse(w)
		return
	}

	model := r.FormValue("model")
	if model == "" {
		errors.BadRequest("Model name is required", nil).WriteHTTPResponse(w)
		return
	}

	concurrency, _ := strconv.Atoi(r.FormValue("concurrency"))
	if concurrency <= 0 {
		concurrency = 4
	}

	retries, _ := strconv.Atoi(r.FormValue("retries"))
	if retries < 0 {
		retries = 3
	}

	// Start download logic here
	// This would call download function with parsed parameters

	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Server) handleModelAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errors.BadRequest("Method not allowed", nil).WriteHTTPResponse(w)
		return
	}

	if err := r.ParseForm(); err != nil {
		errors.BadRequest("Bad request", err).WriteHTTPResponse(w)
		return
	}

	name := r.FormValue("name")
	action := r.FormValue("action")

	if name == "" || action == "" {
		errors.BadRequest("Missing parameters", nil).WriteHTTPResponse(w)
		return
	}

	// Handle model actions (unzip, open-folder, delete)
	// This would call the appropriate action functions

	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Server) handleProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errors.BadRequest("Method not allowed", nil).WriteHTTPResponse(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Return progress data
	data := map[string]interface{}{
		"done":    0,
		"total":   0,
		"percent": 0,
	}

	json.NewEncoder(w).Encode(data)
}

func (s *Server) handleResume(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errors.BadRequest("Method not allowed", nil).WriteHTTPResponse(w)
		return
	}
	// Implementation here
	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Server) handleFileDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errors.BadRequest("Method not allowed", nil).WriteHTTPResponse(w)
		return
	}
	// Implementation here
}

func (s *Server) handleCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errors.BadRequest("Method not allowed", nil).WriteHTTPResponse(w)
		return
	}
	// Implementation here
	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Server) handlePause(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errors.BadRequest("Method not allowed", nil).WriteHTTPResponse(w)
		return
	}
	// Implementation here
	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Server) getPageData() interface{} {
	// Return page data for template
	return struct{}{}
}

func js(value string) string {
	// Escape JavaScript string
	return strings.ReplaceAll(strings.ReplaceAll(value, "\\", "\\\\"), "'", "\\'")
}
