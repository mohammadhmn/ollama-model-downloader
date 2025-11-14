package main

import (
	"archive/zip"
	"context"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

//go:embed templates/index.html
var templateFS embed.FS

const (
	defaultRegistry = "https://registry.ollama.ai"
	defaultWebPort  = 8080
)

var (
	currentZip        string
	currentProgress   *progress
	globalCancel      context.CancelFunc
	currentMessage    string
	pauseRequested    atomic.Bool
	currentSessionDir string
)

type PageData struct {
	Message         string
	ZipPath         string
	Downloads       []downloadEntry
	RunningSession  *partialSessionView
	PausedSessions  []partialSessionView
	ErroredSessions []partialSessionView
}

type downloadEntry struct {
	Name    string
	Model   string
	Path    string
	ModTime time.Time
}

type sessionMeta struct {
	Model       string    `json:"model"`
	SessionID   string    `json:"sessionId"`
	OutZip      string    `json:"outZip"`
	StagingRoot string    `json:"stagingRoot"`
	Registry    string    `json:"registry"`
	Platform    string    `json:"platform"`
	Concurrency int       `json:"concurrency"`
	Retries     int       `json:"retries"`
	StartedAt   time.Time `json:"startedAt"`
	LastUpdated time.Time `json:"lastUpdated"`
	State       string    `json:"state"`
	Message     string    `json:"message"`
}

const sessionMetaFileName = "session.json"

func sessionMetaPath(dir string) string {
	return filepath.Join(dir, sessionMetaFileName)
}

func loadSessionMeta(dir string) (sessionMeta, error) {
	var meta sessionMeta
	data, err := os.ReadFile(sessionMetaPath(dir))
	if err != nil {
		return meta, err
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return meta, err
	}
	return meta, nil
}

func saveSessionMeta(meta sessionMeta) error {
	meta.LastUpdated = time.Now()
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(sessionMetaPath(meta.StagingRoot), data, 0o644)
}

type partialSessionView struct {
	Model      string
	SessionID  string
	Started    string
	Updated    string
	StateLabel string
	Message    string
}

func discoverPartialSessions(outputDir string) ([]sessionMeta, error) {
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return nil, err
	}
	var sessions []sessionMeta
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasSuffix(entry.Name(), ".staging") {
			continue
		}
		meta, err := loadSessionMeta(filepath.Join(outputDir, entry.Name()))
		if err != nil {
			continue
		}
		sessions = append(sessions, meta)
	}
	return sessions, nil
}

func categorizeSessions(metas []sessionMeta) (running *partialSessionView, paused, errored []partialSessionView) {
	sort.Slice(metas, func(i, j int) bool {
		return metas[i].LastUpdated.After(metas[j].LastUpdated)
	})
	for _, meta := range metas {
		view := sessionViewFromMeta(meta)
		switch strings.ToLower(meta.State) {
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

func downloadsFromDir(dir string) []downloadEntry {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var downloads []downloadEntry
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".zip") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		downloads = append(downloads, downloadEntry{
			Name:    entry.Name(),
			Model:   strings.TrimSuffix(entry.Name(), ".zip"),
			Path:    filepath.Join(dir, entry.Name()),
			ModTime: info.ModTime(),
		})
	}
	sort.Slice(downloads, func(i, j int) bool {
		return downloads[i].ModTime.After(downloads[j].ModTime)
	})
	return downloads
}

func sessionViewFromMeta(meta sessionMeta) partialSessionView {
	return partialSessionView{
		Model:      meta.Model,
		SessionID:  meta.SessionID,
		Started:    formatSessionTime(meta.StartedAt),
		Updated:    formatSessionTime(meta.LastUpdated),
		StateLabel: stateLabel(meta.State),
		Message:    meta.Message,
	}
}

func formatSessionTime(t time.Time) string {
	if t.IsZero() {
		return "نامشخص"
	}
	return t.Format("2006-01-02 15:04:05")
}

func stateLabel(state string) string {
	switch strings.ToLower(state) {
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
		return state
	}
}

func beginDownloadSession(opt options, startMessage string) {
	pauseRequested.Store(false)
	currentZip = opt.outZip
	currentProgress = newProgress(0)
	currentMessage = startMessage
	currentSessionDir = opt.stagingDir

	// Create session metadata immediately so it appears in the UI
	_ = os.MkdirAll(opt.stagingDir, 0o755)
	meta := sessionMeta{
		Model:       opt.model,
		SessionID:   opt.sessionID,
		OutZip:      opt.outZip,
		StagingRoot: opt.stagingDir,
		Registry:    opt.registry,
		Platform:    opt.platform,
		Concurrency: opt.concurrency,
		Retries:     opt.retries,
		StartedAt:   time.Now(),
		LastUpdated: time.Now(),
		State:       "downloading",
		Message:     "در حال شروع دانلود...",
	}
	_ = saveSessionMeta(meta)

	ctx, cancel := context.WithCancel(context.Background())
	globalCancel = cancel

	go func() {
		err := run(ctx, opt)
		globalCancel = nil
		currentProgress = nil
		currentSessionDir = ""
		paused := pauseRequested.Load()
		pauseRequested.Store(false)
		if err != nil {
			if err == context.Canceled {
				if paused {
					currentMessage = "دانلود متوقف شد."
				} else {
					currentMessage = "دانلود لغو شد."
				}
			} else {
				setSessionStatus(opt.stagingDir, "error", err.Error())
				currentMessage = fmt.Sprintf("دانلود ناموفق: %s", err.Error())
			}
		} else {
			currentMessage = "دانلود کامل شد."
		}
	}()
}

func setSessionStatus(dir, state, message string) {
	if dir == "" {
		return
	}
	meta, err := loadSessionMeta(dir)
	if err != nil {
		return
	}
	meta.State = state
	meta.Message = message
	_ = saveSessionMeta(meta)
}

func main() {
	var opt options

	flag.StringVar(&opt.registry, "registry", defaultRegistry, "registry base URL")
	flag.IntVar(&opt.concurrency, "concurrency", 4, "number of concurrent blob downloads")
	flag.BoolVar(&opt.verbose, "v", false, "verbose logging")
	flag.BoolVar(&opt.keepStaging, "keep-staging", false, "keep staging directory (do not delete after zip)")
	flag.IntVar(&opt.retries, "retries", 3, "retry attempts for transient errors")
	var timeoutSec int
	flag.IntVar(&timeoutSec, "timeout", 0, "overall request timeout seconds (0 = no limit)")
	flag.BoolVar(&opt.insecureTLS, "insecure", false, "skip TLS verification (NOT recommended)")
	// Default platform from runtime
	defaultPlatform := fmt.Sprintf("linux/%s", archFromGo(runtime.GOARCH))
	flag.StringVar(&opt.platform, "platform", defaultPlatform, "target platform (linux/amd64 or linux/arm64)")
	flag.StringVar(&opt.outZip, "o", "", "output zip path (default: <model>.zip)")
	flag.StringVar(&opt.outputDir, "output-dir", "downloaded-models", "directory to save downloaded models")
	flag.IntVar(&opt.port, "port", 0, "port to listen on (0 for random)")
	flag.Parse()

	if flag.NArg() == 0 {
		startWebServer(opt.port)
	} else {
		opt.model = flag.Arg(0)
		opt.sessionID = sanitizeModelName(opt.model)
		if opt.outZip == "" {
			zipName := opt.sessionID
			if !strings.HasSuffix(strings.ToLower(zipName), ".zip") {
				zipName += ".zip"
			}
			opt.outZip = filepath.Join(opt.outputDir, zipName)
		}
		opt.stagingDir = filepath.Join(opt.outputDir, opt.sessionID+".staging")

		if timeoutSec > 0 {
			opt.timeout = time.Duration(timeoutSec) * time.Second
		} else {
			opt.timeout = 0
		}

		if err := run(context.Background(), opt); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	}
}

func archFromGo(goarch string) string {
	switch goarch {
	case "amd64":
		return "amd64"
	case "arm64":
		return "arm64"
	default:
		return goarch
	}
}

func sanitizeModelName(model string) string {
	s := strings.TrimSpace(model)
	if s == "" {
		return "model"
	}
	s = strings.Map(func(r rune) rune {
		switch {
		case r == '/' || r == ':' || r == '@' || r == '\\' || r == ' ':
			return '-'
		default:
			return r
		}
	}, s)
	s = strings.ToLower(strings.Trim(s, "-"))
	if s == "" {
		return "model"
	}
	return s
}

func startWebServer(port int) {
	// Create template with custom functions
	funcMap := template.FuncMap{
		"contains": strings.Contains,
		"add": func(a, b int) int {
			return a + b
		},
	}
	tmpl, err := template.New("index.html").Funcs(funcMap).ParseFS(templateFS, "templates/index.html")
	if err != nil {
		fmt.Println("Error parsing template:", err)
		return
	}

	downloadsDir := "downloaded-models"
	if err := os.MkdirAll(downloadsDir, 0o755); err != nil {
		fmt.Println("Error creating downloads directory:", err)
		return
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		data := PageData{Message: currentMessage}
		if currentZip != "" {
			if _, err := os.Stat(currentZip); err == nil {
				data.ZipPath = currentZip
			}
		}
		// List downloaded models
		data.Downloads = downloadsFromDir(downloadsDir)
		if sessions, err := discoverPartialSessions(downloadsDir); err == nil {
			running, paused, errored := categorizeSessions(sessions)
			data.RunningSession = running
			data.PausedSessions = paused
			data.ErroredSessions = errored
		}
		tmpl.Execute(w, data)
	})

	http.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		model := r.FormValue("model")
		outputDir := downloadsDir
		concurrencyStr := r.FormValue("concurrency")
		concurrency, _ := strconv.Atoi(concurrencyStr)
		if concurrency <= 0 {
			concurrency = 4
		}
		retriesStr := r.FormValue("retries")
		retries, _ := strconv.Atoi(retriesStr)
		if retries < 0 {
			retries = 3
		}

		opt := options{
			model:       model,
			registry:    defaultRegistry,
			platform:    fmt.Sprintf("linux/%s", archFromGo(runtime.GOARCH)),
			concurrency: concurrency,
			verbose:     false,
			keepStaging: false,
			retries:     retries,
			timeout:     0,
			insecureTLS: false,
			outputDir:   outputDir,
		}

		sessionID := sanitizeModelName(opt.model)
		opt.sessionID = sessionID
		zipName := sessionID
		if !strings.HasSuffix(strings.ToLower(zipName), ".zip") {
			zipName += ".zip"
		}
		opt.outZip = filepath.Join(opt.outputDir, zipName)
		opt.stagingDir = filepath.Join(opt.outputDir, sessionID+".staging")

		beginDownloadSession(opt, "در حال دانلود...")

		http.Redirect(w, r, "/", http.StatusFound)
	})

	http.HandleFunc("/model/action", modelActionHandler(downloadsDir))

	http.HandleFunc("/resume", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		sessionID := r.FormValue("session")
		if sessionID == "" {
			http.Error(w, "Missing session ID", http.StatusBadRequest)
			return
		}
		staging := filepath.Join(downloadsDir, sessionID+".staging")
		meta, err := loadSessionMeta(staging)
		if err != nil {
			http.Error(w, "Session not found", http.StatusNotFound)
			return
		}
		registry := meta.Registry
		if registry == "" {
			registry = defaultRegistry
		}
		platform := meta.Platform
		if platform == "" {
			platform = fmt.Sprintf("linux/%s", archFromGo(runtime.GOARCH))
		}
		concurrency := meta.Concurrency
		if concurrency <= 0 {
			concurrency = 4
		}
		retries := meta.Retries
		if retries < 0 {
			retries = 3
		}

		zipPath := meta.OutZip
		if zipPath == "" {
			name := sessionID
			if !strings.HasSuffix(strings.ToLower(name), ".zip") {
				name += ".zip"
			}
			zipPath = filepath.Join(downloadsDir, name)
		}

		opt := options{
			model:       meta.Model,
			registry:    registry,
			platform:    platform,
			concurrency: concurrency,
			verbose:     false,
			keepStaging: false,
			retries:     retries,
			timeout:     0,
			insecureTLS: false,
			outputDir:   downloadsDir,
			sessionID:   meta.SessionID,
			stagingDir:  staging,
			outZip:      zipPath,
		}
		setSessionStatus(staging, "downloading", "در حال ادامه دانلود...")
		beginDownloadSession(opt, "در حال ادامه دانلود...")
		http.Redirect(w, r, "/", http.StatusFound)
	})

	http.HandleFunc("/download/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		filename := strings.TrimPrefix(r.URL.Path, "/download/")
		if filename == "" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		http.ServeFile(w, r, filename)
	})

	http.HandleFunc("/progress", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		data := ProgressData{}
		if currentProgress != nil {
			data.Done = atomic.LoadInt64(&currentProgress.done)
			data.Total = currentProgress.total
			if data.Total > 0 {
				data.Percent = int((data.Done * 100) / data.Total)
			}
		}
		json.NewEncoder(w).Encode(data)
	})

	http.HandleFunc("/cancel", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		pauseRequested.Store(false)
		if globalCancel != nil {
			setSessionStatus(currentSessionDir, "paused", "لغو شد")
			globalCancel()
		}
		http.Redirect(w, r, "/", http.StatusFound)
	})

	http.HandleFunc("/pause", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if globalCancel != nil {
			pauseRequested.Store(true)
			setSessionStatus(currentSessionDir, "paused", "مکث شد")
			globalCancel()
		}
		http.Redirect(w, r, "/", http.StatusFound)
	})

	bindPort := port
	if bindPort == 0 {
		bindPort = defaultWebPort
	}
	addr := fmt.Sprintf(":%d", bindPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Printf("Port %d not available, using random port...\n", bindPort)
		listener, err = net.Listen("tcp", ":0")
		if err != nil {
			fmt.Println("Error starting server:", err)
			return
		}
	}
	actualPort := listener.Addr().(*net.TCPAddr).Port
	fmt.Printf("Running on http://localhost:%d\n", actualPort)
	go http.Serve(listener, nil)
	url := fmt.Sprintf("http://localhost:%d", actualPort)
	openBrowser(url)
	select {}
}

func modelActionHandler(downloadsDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		name := r.FormValue("name")
		action := r.FormValue("action")
		if name == "" || action == "" {
			http.Error(w, "Missing parameters", http.StatusBadRequest)
			return
		}
		target := filepath.Join(downloadsDir, name)
		var msg string
		var err error
		switch action {
		case "delete":
			err = os.Remove(target)
			if err == nil {
				staging := filepath.Join(downloadsDir, strings.TrimSuffix(name, ".zip")+".staging")
				_ = os.RemoveAll(staging)
				msg = fmt.Sprintf("%s حذف شد.", name)
			}
		case "open-folder":
			err = openExplorer(downloadsDir)
			if err == nil {
				msg = "پوشه دانلود باز شد."
			}
		case "unzip":
			dest, derr := ollamaModelsDir()
			if derr != nil {
				err = derr
				break
			}
			err = unzipToDir(target, dest)
			if err == nil {
				msg = fmt.Sprintf("%s به %s استخراج شد.", name, dest)
			}
		default:
			err = fmt.Errorf("عمل نامعتبر: %s", action)
		}
		if err != nil {
			currentMessage = fmt.Sprintf("خطا: %s", err)
		} else if msg != "" {
			currentMessage = msg
		}
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func openExplorer(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", path)
	default:
		return fmt.Errorf("unsupported OS")
	}
	return cmd.Start()
}

func ollamaModelsDir() (string, error) {
	if dir := os.Getenv("OLLAMA_MODELS_DIR"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "windows":
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			return filepath.Join(local, "Ollama", "models"), nil
		}
		return filepath.Join(home, "AppData", "Local", "Ollama", "models"), nil
	default:
		return filepath.Join(home, ".ollama", "models"), nil
	}
}

func unzipToDir(zipPath, dest string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	destClean := filepath.Clean(dest)
	if err := os.MkdirAll(destClean, 0o755); err != nil {
		return err
	}

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			targetDir := filepath.Join(destClean, filepath.FromSlash(f.Name))
			if err := os.MkdirAll(targetDir, f.Mode()); err != nil {
				return err
			}
			continue
		}
		targetPath := filepath.Join(destClean, filepath.FromSlash(f.Name))
		if !strings.HasPrefix(filepath.Clean(targetPath), destClean+string(os.PathSeparator)) && filepath.Clean(targetPath) != destClean {
			return fmt.Errorf("invalid file path: %s", f.Name)
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}
		out, err := os.OpenFile(targetPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			out.Close()
			return err
		}
		if _, err := io.Copy(out, rc); err != nil {
			rc.Close()
			out.Close()
			return err
		}
		rc.Close()
		out.Close()
	}
	return nil
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", "-a", "Google Chrome", url)
	case "linux":
		cmd = exec.Command("google-chrome", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "chrome.exe", url)
	default:
		fmt.Println("Unsupported OS for opening browser")
		return
	}
	cmd.Start()
}
