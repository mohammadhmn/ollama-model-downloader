package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"ollama-model-downloader/config"
	"ollama-model-downloader/models"
)

//go:embed templates/index.html
var templateFS embed.FS

//go:embed Vazirmatn/static/Vazirmatn-Regular.ttf
var fontFS embed.FS

var (
	currentProgressMu sync.RWMutex
	currentProgress   *progress
)

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

type pageData struct {
	Downloads []models.DownloadEntry
	Queue     []taskView
	Message   string
}

type queueState string

const (
	stateQueued      queueState = "queued"
	stateDownloading queueState = "downloading"
	statePaused      queueState = "paused"
	stateCanceled    queueState = "canceled"
	stateError       queueState = "error"
	stateDone        queueState = "done"
)

func queueStateLabel(state queueState) string {
	switch state {
	case stateDownloading:
		return "Downloading"
	case statePaused:
		return "Paused"
	case stateCanceled:
		return "Canceled"
	case stateError:
		return "Error"
	case stateDone:
		return "Completed"
	default:
		return "Queued"
	}
}

func queueStateClass(state queueState) string {
	switch state {
	case stateDownloading:
		return "state-running"
	case statePaused:
		return "state-paused"
	case stateCanceled:
		return "state-canceled"
	case stateError:
		return "state-error"
	case stateDone:
		return "state-done"
	default:
		return "state-queued"
	}
}

type taskView struct {
	ID         string
	Model      string
	State      queueState
	StateLabel string
	StateClass string
	Message    string
	Percent    int
	Done       int64
	Total      int64
	CreatedAt  string
	UpdatedAt  string
	ZipName    string
}

type downloadTask struct {
	ID        string
	Model     string
	Sanitized string
	State     queueState
	Message   string
	ZipName   string
	ZipPath   string
	CreatedAt time.Time
	UpdatedAt time.Time
	Progress  *progress
	cancel    context.CancelFunc
}

func (t *downloadTask) view() taskView {
	var done, total int64
	if t.Progress != nil {
		done = atomic.LoadInt64(&t.Progress.done)
		total = t.Progress.total
	}
	percent := 0
	if total > 0 {
		percent = int((done * 100) / total)
	}
	return taskView{
		ID:         t.ID,
		Model:      t.Model,
		State:      t.State,
		StateLabel: queueStateLabel(t.State),
		StateClass: queueStateClass(t.State),
		Message:    t.Message,
		Percent:    percent,
		Done:       done,
		Total:      total,
		CreatedAt:  formatTime(t.CreatedAt),
		UpdatedAt:  formatTime(t.UpdatedAt),
		ZipName:    t.ZipName,
	}
}

type downloadManager struct {
	cfg          *config.Config
	downloadsDir string
	tasks        []*downloadTask
	mu           sync.Mutex
	cond         *sync.Cond
}

func newDownloadManager(cfg *config.Config) *downloadManager {
	m := &downloadManager{
		cfg:          cfg,
		downloadsDir: cfg.OutputDir,
	}
	m.cond = sync.NewCond(&m.mu)
	return m
}

func (m *downloadManager) Snapshot() []taskView {
	m.mu.Lock()
	defer m.mu.Unlock()
	views := make([]taskView, 0, len(m.tasks))
	for _, task := range m.tasks {
		views = append(views, task.view())
	}
	return views
}

func (m *downloadManager) Enqueue(model string) {
	sanitized := config.SanitizeModelName(model)
	id := newTaskID()
	zipName := fmt.Sprintf("%s-%s.zip", sanitized, id)
	task := &downloadTask{
		ID:        id,
		Model:     model,
		Sanitized: sanitized,
		State:     stateQueued,
		Message:   "Queued",
		ZipName:   zipName,
		ZipPath:   filepath.Join(m.downloadsDir, zipName),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.mu.Lock()
	m.tasks = append(m.tasks, task)
	m.cond.Signal()
	m.mu.Unlock()
}

func (m *downloadManager) runQueue() {
	for {
		m.mu.Lock()
		task := m.nextQueuedTaskLocked()
		for task == nil {
			m.cond.Wait()
			task = m.nextQueuedTaskLocked()
		}
		task.State = stateDownloading
		task.Message = "Downloading..."
		task.UpdatedAt = time.Now()
		m.mu.Unlock()

		ctx, cancel := context.WithCancel(context.Background())
		task.cancel = cancel
		progress := &progress{}
		setCurrentProgress(progress)
		task.Progress = progress
		err := m.downloadModel(ctx, task)
		setCurrentProgress(nil)
		cancel()

		m.mu.Lock()
		if task.State == statePaused || task.State == stateCanceled {
			task.cancel = nil
			task.UpdatedAt = time.Now()
			m.mu.Unlock()
			continue
		}
		if err != nil {
			if errors.Is(err, context.Canceled) && (task.State == statePaused || task.State == stateCanceled) {
			} else {
				task.State = stateError
				task.Message = err.Error()
			}
		} else {
			task.State = stateDone
			task.Message = "Completed"
		}
		task.cancel = nil
		task.UpdatedAt = time.Now()
		m.mu.Unlock()
	}
}

func (m *downloadManager) nextQueuedTaskLocked() *downloadTask {
	for _, task := range m.tasks {
		if task.State == stateQueued {
			return task
		}
	}
	return nil
}

func (m *downloadManager) downloadModel(ctx context.Context, task *downloadTask) error {
	staging := filepath.Join(m.downloadsDir, fmt.Sprintf("%s-%s.staging", task.Sanitized, task.ID))
	opt := options{
		model:       task.Model,
		registry:    m.cfg.Registry,
		platform:    m.cfg.Platform,
		outZip:      task.ZipPath,
		concurrency: m.cfg.Concurrency,
		retries:     m.cfg.Retries,
		timeout:     m.cfg.Timeout,
		insecureTLS: m.cfg.InsecureTLS,
		stagingDir:  staging,
		sessionID:   task.ID,
	}
	return run(ctx, opt)
}

func (m *downloadManager) performAction(id, action string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	task := m.findTaskLocked(id)
	if task == nil {
		return "Task not found", false
	}
	switch action {
	case "pause":
		if task.State == stateDownloading {
			task.State = statePaused
			task.Message = "Paused"
			if task.cancel != nil {
				task.cancel()
			}
			task.UpdatedAt = time.Now()
			return "Paused", true
		}
		if task.State == stateQueued {
			task.State = statePaused
			task.Message = "Paused"
			task.UpdatedAt = time.Now()
			return "Paused queued task", true
		}
		return "Cannot pause task in this state", false
	case "resume":
		if task.State == statePaused || task.State == stateError || task.State == stateCanceled {
			task.State = stateQueued
			task.Message = "Queued"
			task.UpdatedAt = time.Now()
			task.Progress = nil
			m.cond.Signal()
			return "Resumed", true
		}
		return "Cannot resume task", false
	case "cancel":
		if task.State == stateDownloading {
			task.State = stateCanceled
			task.Message = "Canceled"
			if task.cancel != nil {
				task.cancel()
			}
			task.UpdatedAt = time.Now()
			return "Canceled", true
		}
		if task.State == stateQueued || task.State == statePaused {
			task.State = stateCanceled
			task.Message = "Canceled"
			task.UpdatedAt = time.Now()
			return "Canceled", true
		}
		return "Cannot cancel task", false
	default:
		return "Unknown action", false
	}
}

func (m *downloadManager) findTaskLocked(id string) *downloadTask {
	for _, task := range m.tasks {
		if task.ID == id {
			return task
		}
	}
	return nil
}

type server struct {
	tmpl    *template.Template
	manager *downloadManager
}

func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	data := pageData{
		Downloads: models.DownloadsFromDir(s.manager.downloadsDir),
		Queue:     s.manager.Snapshot(),
		Message:   r.URL.Query().Get("message"),
	}
	if err := s.tmpl.Execute(w, data); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}

func (s *server) handleQueueAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		redirectWithMessage(w, r, "Invalid form")
		return
	}
	model := strings.TrimSpace(r.FormValue("model"))
	if model == "" {
		redirectWithMessage(w, r, "Model name required")
		return
	}
	s.manager.Enqueue(model)
	redirectWithMessage(w, r, "Model queued")
}

func (s *server) handleQueueAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		redirectWithMessage(w, r, "Invalid action")
		return
	}
	id := strings.TrimSpace(r.FormValue("id"))
	action := strings.TrimSpace(r.FormValue("action"))
	if id == "" || action == "" {
		redirectWithMessage(w, r, "Missing action parameters")
		return
	}
	msg, ok := s.manager.performAction(id, action)
	if ok {
		redirectWithMessage(w, r, msg)
	} else {
		redirectWithMessage(w, r, msg)
	}
}

func (s *server) handleZip(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/download/")
	if name == "" {
		http.NotFound(w, r)
		return
	}
	path := filepath.Join(s.manager.downloadsDir, name)
	baseDir := filepath.Clean(s.manager.downloadsDir)
	if !strings.HasPrefix(filepath.Clean(path), baseDir+string(os.PathSeparator)) {
		http.NotFound(w, r)
		return
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, path)
}

func (s *server) handleFont(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/font/")
	if name == "" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "font/ttf")
	w.Header().Set("Cache-Control", "public, max-age=31536000")
	http.FileServer(http.FS(fontFS)).ServeHTTP(w, r)
}

func redirectWithMessage(w http.ResponseWriter, r *http.Request, message string) {
	target := "/"
	if message != "" {
		target = "/?message=" + url.QueryEscape(message)
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("2006-01-02 15:04:05")
}

func setCurrentProgress(p *progress) {
	currentProgressMu.Lock()
	currentProgress = p
	currentProgressMu.Unlock()
}

func newTaskID() string {
	return fmt.Sprintf("%d%04d", time.Now().UnixNano(), rand.Intn(10000))
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", url)
	default:
		return
	}
	_ = cmd.Start()
}

func main() {
	rand.Seed(time.Now().UnixNano())
	cfg, err := config.Parse()
	if err != nil {
		log.Fatalf("failed to parse config: %v", err)
	}
	if cfg.OutputDir == "" {
		cfg.OutputDir = "downloaded-models"
	}
	if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
		log.Fatalf("failed to create downloads dir: %v", err)
	}

	manager := newDownloadManager(cfg)
	go manager.runQueue()

	tmpl, err := template.New("index.html").Funcs(template.FuncMap{
		"formatTime": formatTime,
		"humanBytes": humanBytes,
		"stateLabel": queueStateLabel,
		"stateClass": queueStateClass,
	}).ParseFS(templateFS, "templates/index.html")
	if err != nil {
		log.Fatalf("failed to parse template: %v", err)
	}

	server := &server{
		tmpl:    tmpl,
		manager: manager,
	}

	addr := fmt.Sprintf(":%d", cfg.Port)
	if cfg.Port == 0 {
		addr = ":0"
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		listener, err = net.Listen("tcp", ":0")
		if err != nil {
			log.Fatalf("failed to start server: %v", err)
		}
	}

	actualPort := listener.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://localhost:%d", actualPort)
	fmt.Println("Opening web UI at", url)
	go openBrowser(url)

	mux := http.NewServeMux()
	mux.HandleFunc("/", server.handleIndex)
	mux.HandleFunc("/queue/add", server.handleQueueAdd)
	mux.HandleFunc("/queue/action", server.handleQueueAction)
	mux.HandleFunc("/download/", server.handleZip)
	mux.HandleFunc("/font/", server.handleFont)

	if err := http.Serve(listener, mux); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server terminated: %v", err)
	}
}
