package main

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
)

var currentZip string
var currentProgress *progress
var globalCancel context.CancelFunc
var currentMessage string

type PageData struct {
	Message   string
	ZipPath   string
	Downloads []string
}

type ProgressData struct {
	Done    int64 `json:"done"`
	Total   int64 `json:"total"`
	Percent int   `json:"percent"`
}

// OCI / Docker media types we care about
const (
	mtOCIIndex    = "application/vnd.oci.image.index.v1+json"
	mtDockerIndex = "application/vnd.docker.distribution.manifest.list.v2+json"

	mtOCIManifest    = "application/vnd.oci.image.manifest.v1+json"
	mtDockerManifest = "application/vnd.docker.distribution.manifest.v2+json"
)

type imageIndex struct {
	Manifests []struct {
		MediaType string `json:"mediaType"`
		Digest    string `json:"digest"`
		Platform  struct {
			Architecture string `json:"architecture"`
			OS           string `json:"os"`
		} `json:"platform"`
	} `json:"manifests"`
}

type imageManifest struct {
	MediaType string `json:"mediaType"`
	Config    struct {
		MediaType string `json:"mediaType"`
		Digest    string `json:"digest"`
		Size      int64  `json:"size"`
	} `json:"config"`
	Layers []struct {
		MediaType string `json:"mediaType"`
		Digest    string `json:"digest"`
		Size      int64  `json:"size"`
	} `json:"layers"`
}

type bearerAuth struct {
	Realm   string
	Service string
	Scope   string
}

type options struct {
	model       string
	registry    string
	platform    string // linux/amd64 or linux/arm64
	outZip      string
	concurrency int
	verbose     bool
	keepStaging bool
	retries     int
	timeout     time.Duration
	insecureTLS bool
	port        int
	outputDir   string
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

		if opt.outZip == "" {
			// sanitize model name to be filesystem friendly
			sanitized := strings.ReplaceAll(opt.model, "/", "-")
			sanitized = strings.ReplaceAll(sanitized, ":", "-")
			sanitized = strings.ReplaceAll(sanitized, "@", "-")
			if !strings.HasSuffix(strings.ToLower(sanitized), ".zip") {
				sanitized += ".zip"
			}
			opt.outZip = filepath.Join(opt.outputDir, sanitized)
		}

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

type modelRef struct {
	Host         string // registry host, e.g. registry.ollama.ai
	Repository   string // e.g. library/llama3
	Reference    string // tag or digest
	ReferenceTag string // tag (if provided)
	IsDigest     bool
}

func parseModel(registryBase, model string) (modelRef, error) {
	// Accept forms:
	//   name[:tag]
	//   owner/name[:tag]
	//   name@sha256:...
	//   owner/name@sha256:...
	// Default tag is latest, default owner is library.

	u, err := url.Parse(registryBase)
	if err != nil {
		return modelRef{}, fmt.Errorf("invalid registry base: %w", err)
	}
	host := u.Host

	ref := model
	var repository string
	var reference string
	var tag string
	var isDigest bool

	if strings.Contains(ref, "@sha256:") {
		parts := strings.Split(ref, "@")
		name := parts[0]
		digest := parts[1]
		isDigest = true
		if !strings.Contains(name, "/") {
			repository = "library/" + name
		} else {
			repository = name
		}
		reference = digest
	} else {
		// tag or default latest
		var name string
		if strings.Contains(ref, ":") {
			p := strings.Split(ref, ":")
			name = p[0]
			tag = p[1]
		} else {
			name = ref
			tag = "latest"
		}
		if !strings.Contains(name, "/") {
			repository = "library/" + name
		} else {
			repository = name
		}
		reference = tag
	}

	return modelRef{Host: host, Repository: repository, Reference: reference, ReferenceTag: tag, IsDigest: isDigest}, nil
}

func run(ctx context.Context, opt options) error {
	// HTTP client with tuned transport
	client := newHTTPClient(opt)

	ref, err := parseModel(opt.registry, opt.model)
	if err != nil {
		return err
	}

	if opt.verbose {
		fmt.Printf("Resolved repository: %s, reference: %s, host: %s\n", ref.Repository, ref.Reference, ref.Host)
	}

	// 1) Get auth challenge and token
	token, err := getRegistryToken(ctx, client, opt, ref.Repository, ref.Reference)
	if err != nil {
		return fmt.Errorf("auth failed: %w", err)
	}

	// 2) Fetch manifest or index
	manifestJSON, manifestType, err := getManifestOrIndex(ctx, client, opt, ref.Repository, ref.Reference, token)
	if err != nil {
		return err
	}

	var manifest imageManifest
	switch manifestType {
	case mtOCIManifest, mtDockerManifest:
		if err := json.Unmarshal(manifestJSON, &manifest); err != nil {
			return fmt.Errorf("decode manifest: %w", err)
		}
	case mtOCIIndex, mtDockerIndex:
		// select platform
		var idx imageIndex
		if err := json.Unmarshal(manifestJSON, &idx); err != nil {
			return fmt.Errorf("decode index: %w", err)
		}
		arch := strings.Split(opt.platform, "/")
		targetOS, targetArch := "linux", arch[len(arch)-1]

		// Prefer exact match; if multiple, take first deterministic order
		var candidates []string
		for _, m := range idx.Manifests {
			if strings.EqualFold(m.Platform.OS, targetOS) && strings.EqualFold(m.Platform.Architecture, targetArch) {
				candidates = append(candidates, m.Digest)
			}
		}
		if len(candidates) == 0 {
			return fmt.Errorf("no manifest for platform %s found in index", opt.platform)
		}
		sort.Strings(candidates)
		chosen := candidates[0]
		if opt.verbose {
			fmt.Printf("Selected platform manifest: %s (%s)\n", chosen, opt.platform)
		}
		manifestJSON, manifestType, err = getManifestOrIndex(ctx, client, opt, ref.Repository, chosen, token)
		if err != nil {
			return err
		}
		if manifestType != mtOCIManifest && manifestType != mtDockerManifest {
			return fmt.Errorf("unexpected mediaType for chosen manifest: %s", manifestType)
		}
		if err := json.Unmarshal(manifestJSON, &manifest); err != nil {
			return fmt.Errorf("decode chosen manifest: %w", err)
		}
		// When pulling by digest, treat reference as digest for manifest storage
		if ref.ReferenceTag == "" {
			ref.IsDigest = true
		}
	default:
		if opt.verbose {
			fmt.Printf("Unexpected Content-Type: %s; attempting auto-detect...\n", manifestType)
		}
		// Try to decode as manifest first
		if err := json.Unmarshal(manifestJSON, &manifest); err == nil && (manifest.Config.Digest != "" || len(manifest.Layers) > 0) {
			// proceed as manifest
			break
		}
		// Try to decode as index and select platform
		var idx imageIndex
		if err := json.Unmarshal(manifestJSON, &idx); err == nil && len(idx.Manifests) > 0 {
			arch := strings.Split(opt.platform, "/")
			targetOS, targetArch := "linux", arch[len(arch)-1]
			var candidates []string
			for _, m := range idx.Manifests {
				if strings.EqualFold(m.Platform.OS, targetOS) && strings.EqualFold(m.Platform.Architecture, targetArch) {
					candidates = append(candidates, m.Digest)
				}
			}
			if len(candidates) == 0 {
				return fmt.Errorf("no manifest for platform %s found in index (fallback)", opt.platform)
			}
			sort.Strings(candidates)
			chosen := candidates[0]
			if opt.verbose {
				fmt.Printf("Selected platform manifest (fallback): %s (%s)\n", chosen, opt.platform)
			}
			manifestJSON, manifestType, err = getManifestOrIndex(ctx, client, opt, ref.Repository, chosen, token)
			if err != nil {
				return err
			}
			if err := json.Unmarshal(manifestJSON, &manifest); err != nil {
				return fmt.Errorf("decode chosen manifest (fallback): %w", err)
			}
			if ref.ReferenceTag == "" {
				ref.IsDigest = true
			}
			break
		}
		snippet := string(manifestJSON)
		if len(snippet) > 256 {
			snippet = snippet[:256] + "..."
		}
		return fmt.Errorf("unsupported manifest type: %s; body: %s", manifestType, snippet)
	}

	// 3) Stage files in temp dir
	stagingRoot, err := os.MkdirTemp(".", "ollama-staging-")
	if err != nil {
		return err
	}
	// create models/{manifests,blobs}
	modelsRoot := filepath.Join(stagingRoot, "models")
	blobsDir := filepath.Join(modelsRoot, "blobs")
	manifestsDir := filepath.Join(modelsRoot, "manifests", ref.Host, ref.Repository)
	if err := os.MkdirAll(blobsDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(manifestsDir, 0o755); err != nil {
		return err
	}

	// 4) Write manifest to path `manifests/<host>/<repo>/<tag or digest>`
	manifestTail := ref.Reference
	if ref.IsDigest {
		// store as sha256-<hex>
		if strings.HasPrefix(manifestTail, "sha256:") {
			manifestTail = "sha256-" + strings.TrimPrefix(manifestTail, "sha256:")
		}
	}
	manifestPath := filepath.Join(manifestsDir, manifestTail)
	if err := os.WriteFile(manifestPath, manifestJSON, 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	if opt.verbose {
		fmt.Printf("Wrote manifest: %s\n", manifestPath)
	}

	// 5) Download config + layers into blobs as sha256-<hex>
	var items []blobItem
	if manifest.Config.Digest != "" {
		items = append(items, blobItem{digest: manifest.Config.Digest, size: manifest.Config.Size})
	}
	for _, l := range manifest.Layers {
		items = append(items, blobItem{digest: l.Digest, size: l.Size})
	}
	items = dedupeBlobs(items)

	// Progress bar for total known bytes
	var total int64
	for _, it := range items {
		if it.size > 0 {
			total += it.size
		}
	}
	var p *progress
	if currentProgress != nil {
		p = currentProgress
		p.total = total
		// Don't start/stop for web UI, progress shown in browser
	} else {
		p = newProgress(total)
		if total > 0 {
			p.Start()
			defer func() {
				p.Stop()
				fmt.Fprintln(os.Stderr) // newline after progress
			}()
		}
	}

	sem := make(chan struct{}, max(1, opt.concurrency))
	errCh := make(chan error, len(items))
	for _, it := range items {
		it := it
		sem <- struct{}{}
		go func() {
			defer func() { <-sem }()
			if err := downloadBlob(ctx, client, opt.registry, ref.Repository, it.digest, token, blobsDir, opt.retries, p, it.size, opt.verbose); err != nil {
				errCh <- err
			}
		}()
	}
	// wait for all
	for i := 0; i < cap(sem); i++ {
		sem <- struct{}{}
	}
	close(errCh)
	for err := range errCh {
		if err != nil {
			return err
		}
	}

	// 6) Zip models/ content to output zip
	if err := os.MkdirAll(filepath.Dir(opt.outZip), 0755); err != nil {
		return err
	}
	if err := zipDir(modelsRoot, opt.outZip); err != nil {
		return fmt.Errorf("zip: %w", err)
	}
	if opt.verbose {
		fmt.Printf("Created zip: %s\n", opt.outZip)
	} else {
		fmt.Println("OK:", opt.outZip)
	}

	// 7) Cleanup staging
	if !opt.keepStaging {
		_ = os.RemoveAll(stagingRoot)
	} else {
		fmt.Println("staging kept at:", stagingRoot)
	}
	return nil
}

func uniqueStrings(in []string) []string {
	m := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := m[s]; !ok {
			m[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}

// dedupeBlobs removes duplicate digests keeping the first observed size.
type blobItem struct {
	digest string
	size   int64
}

func dedupeBlobs(items []blobItem) []blobItem {
	seen := make(map[string]int)
	out := make([]blobItem, 0, len(items))
	for _, it := range items {
		if _, ok := seen[it.digest]; ok {
			continue
		}
		seen[it.digest] = 1
		out = append(out, it)
	}
	return out
}

func getRegistryToken(ctx context.Context, client *http.Client, opt options, repository, reference string) (string, error) {
	// Probe without auth to get challenge (GET for broader compatibility)
	manifestURL := fmt.Sprintf("%s/v2/%s/manifests/%s", strings.TrimRight(opt.registry, "/"), repository, reference)
	headers := map[string]string{
		"Accept":     strings.Join([]string{mtOCIIndex, mtOCIManifest, mtDockerIndex, mtDockerManifest}, ", "),
		"User-Agent": "ollama-model-downloader/1.0",
	}
	resp, err := httpReqWithRetry(ctx, client, http.MethodGet, manifestURL, headers, opt.retries, opt.verbose)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK { // no auth required
		return "", nil
	}
	if resp.StatusCode != http.StatusUnauthorized {
		return "", fmt.Errorf("unexpected status probing auth: %s", resp.Status)
	}
	chal := resp.Header.Get("Www-Authenticate")
	if chal == "" {
		chal = resp.Header.Get("WWW-Authenticate")
	}
	if chal == "" {
		return "", errors.New("missing WWW-Authenticate header for bearer challenge")
	}
	b, err := parseBearerChallenge(chal)
	if err != nil {
		return "", err
	}
	if b.Scope == "" {
		// Standard scope for pull
		b.Scope = fmt.Sprintf("repository:%s:pull", repository)
	}
	// request token
	v := url.Values{}
	if b.Service != "" {
		v.Set("service", b.Service)
	}
	if b.Scope != "" {
		v.Set("scope", b.Scope)
	}
	realm, err := url.Parse(b.Realm)
	if err != nil {
		return "", fmt.Errorf("invalid realm: %w", err)
	}
	realm.RawQuery = v.Encode()
	trsp, err := httpReqWithRetry(ctx, client, http.MethodGet, realm.String(), map[string]string{"User-Agent": "ollama-model-downloader/1.0"}, opt.retries, opt.verbose)
	if err != nil {
		return "", err
	}
	defer trsp.Body.Close()
	if trsp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token fetch failed: %s", trsp.Status)
	}
	var tok struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		IssuedAt    string `json:"issued_at"`
	}
	if err := json.NewDecoder(trsp.Body).Decode(&tok); err != nil {
		return "", err
	}
	if tok.Token != "" {
		return tok.Token, nil
	}
	if tok.AccessToken != "" {
		return tok.AccessToken, nil
	}
	return "", errors.New("no token in auth response")
}

var bearerRe = regexp.MustCompile(`Bearer\s+realm="([^"]+)"(?:,\s*service="([^"]+)")?(?:,\s*scope="([^"]+)")?`)

func parseBearerChallenge(hdr string) (bearerAuth, error) {
	m := bearerRe.FindStringSubmatch(hdr)
	if m == nil {
		return bearerAuth{}, fmt.Errorf("unsupported auth challenge: %s", hdr)
	}
	return bearerAuth{Realm: m[1], Service: m[2], Scope: m[3]}, nil
}

func getManifestOrIndex(ctx context.Context, client *http.Client, opt options, repository, reference, token string) ([]byte, string, error) {
	u := fmt.Sprintf("%s/v2/%s/manifests/%s", strings.TrimRight(opt.registry, "/"), repository, reference)
	headers := map[string]string{
		"Accept":     strings.Join([]string{mtOCIIndex, mtOCIManifest, mtDockerIndex, mtDockerManifest}, ", "),
		"User-Agent": "ollama-model-downloader/1.0",
	}
	if token != "" {
		headers["Authorization"] = "Bearer " + token
	}
	resp, err := httpReqWithRetry(ctx, client, http.MethodGet, u, headers, opt.retries, opt.verbose)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("manifest fetch failed: %s", resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	ctype := resp.Header.Get("Content-Type")
	if ctype == "" {
		ctype = mtOCIManifest // be lenient
	}
	// trim parameters if any
	if i := strings.Index(ctype, ";"); i >= 0 {
		ctype = strings.TrimSpace(ctype[:i])
	}
	return data, ctype, nil
}

func downloadBlob(ctx context.Context, client *http.Client, registryBase, repository, digest, token, blobsDir string, retries int, p *progress, expectedSize int64, verbose bool) error {
	if !strings.HasPrefix(digest, "sha256:") {
		return fmt.Errorf("unsupported digest: %s", digest)
	}
	hexhash := strings.TrimPrefix(digest, "sha256:")
	outPath := filepath.Join(blobsDir, "sha256-"+hexhash)
	// Skip if file exists with nonzero size
	if st, err := os.Stat(outPath); err == nil && st.Size() > 0 {
		if verbose {
			fmt.Printf("blob exists, skipping: %s\n", outPath)
		}
		if p != nil && expectedSize > 0 {
			p.Add(expectedSize)
		}
		return nil
	}
	// download to temp then rename
	tmp := outPath + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	defer f.Close()

	u := fmt.Sprintf("%s/v2/%s/blobs/%s", strings.TrimRight(registryBase, "/"), repository, digest)
	headers := map[string]string{
		"Accept":     "application/octet-stream",
		"User-Agent": "ollama-model-downloader/1.0",
	}
	if token != "" {
		headers["Authorization"] = "Bearer " + token
	}
	resp, err := httpReqWithRetry(ctx, client, http.MethodGet, u, headers, retries, verbose)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("blob fetch failed (%s): %s", digest, resp.Status)
	}
	// Optional: verify sha256 while streaming and update progress
	hasher := sha256.New()
	var w io.Writer = io.MultiWriter(f, hasher)
	if p != nil {
		w = io.MultiWriter(f, hasher, p)
	}
	if _, err := io.Copy(w, resp.Body); err != nil {
		return err
	}
	sum := hex.EncodeToString(hasher.Sum(nil))
	if sum != hexhash {
		return fmt.Errorf("sha256 mismatch for %s: got %s", digest, sum)
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, outPath)
}

func zipDir(root, outZip string) error {
	// root folder will be included content-only; we want manifests/ and blobs/ at zip root
	out, err := os.Create(outZip)
	if err != nil {
		return err
	}
	defer out.Close()

	zw := zip.NewWriter(out)
	defer zw.Close()

	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		// zip needs forward slashes
		name := filepath.ToSlash(rel)
		if info.IsDir() {
			if !strings.HasSuffix(name, "/") {
				name += "/"
			}
			_, err := zw.CreateHeader(&zip.FileHeader{
				Name:     name,
				Method:   zip.Deflate,
				Modified: time.Now(),
			})
			return err
		}
		fh, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		fh.Name = name
		fh.Method = zip.Deflate
		fh.Modified = time.Now()
		w, err := zw.CreateHeader(fh)
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(w, f)
		return err
	})
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// progress is a simple concurrent progress tracker printing a single-line bar.
type progress struct {
	total int64
	done  int64
	tick  *time.Ticker
	quit  chan struct{}
}

func newProgress(total int64) *progress {
	return &progress{total: total, quit: make(chan struct{})}
}

// Write implements io.Writer so we can hook into io.Copy
func (p *progress) Write(b []byte) (int, error) {
	if p == nil {
		return len(b), nil
	}
	// atomic add
	p.Add(int64(len(b)))
	return len(b), nil
}

func (p *progress) Add(n int64) {
	if p == nil {
		return
	}
	atomic.AddInt64(&p.done, n)
}

func (p *progress) Start() {
	if p == nil || p.total <= 0 {
		return
	}
	p.tick = time.NewTicker(200 * time.Millisecond)
	go func() {
		for {
			select {
			case <-p.tick.C:
				p.render()
			case <-p.quit:
				if p.tick != nil {
					p.tick.Stop()
				}
				p.render()
				return
			}
		}
	}()
}

func (p *progress) Stop() {
	if p == nil || p.total <= 0 {
		return
	}
	select {
	case p.quit <- struct{}{}:
	default:
	}
}

func (p *progress) render() {
	done := atomic.LoadInt64(&p.done)
	if done > p.total {
		done = p.total
	}
	percent := 0
	if p.total > 0 {
		percent = int((done * 100) / p.total)
	}
	line := fmt.Sprintf("Downloading: %s / %s (%d%%)\r", humanBytes(done), humanBytes(p.total), percent)
	os.Stderr.WriteString(line)
}

func humanBytes(n int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)
	switch {
	case n >= GB:
		return fmt.Sprintf("%.2f GiB", float64(n)/float64(GB))
	case n >= MB:
		return fmt.Sprintf("%.2f MiB", float64(n)/float64(MB))
	case n >= KB:
		return fmt.Sprintf("%.2f KiB", float64(n)/float64(KB))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

// newHTTPClient builds an HTTP client with tuned timeouts suitable for large downloads
func newHTTPClient(opt options) *http.Client {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	tr := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: opt.insecureTLS},
		TLSHandshakeTimeout:   30 * time.Second,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 60 * time.Second,
	}
	return &http.Client{
		Transport: tr,
		Timeout:   opt.timeout, // 0 means no overall timeout
	}
}

// httpReqWithRetry performs the request with basic exponential backoff on
// timeouts, temporary network errors, and retryable status codes.
func httpReqWithRetry(ctx context.Context, client *http.Client, method, url string, headers map[string]string, retries int, verbose bool) (*http.Response, error) {
	var lastErr error
	attempts := max(1, retries+1)
	for i := 0; i < attempts; i++ {
		req, _ := http.NewRequestWithContext(ctx, method, url, nil)
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		resp, err := client.Do(req)
		if err == nil {
			if isRetryableStatus(resp.StatusCode) && i < attempts-1 {
				// drain body to reuse connection
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				backoff(i, verbose)
				continue
			}
			return resp, nil
		}
		lastErr = err
		if !isRetryableError(err) || i == attempts-1 {
			break
		}
		backoff(i, verbose)
	}
	return nil, lastErr
}

func isRetryableStatus(code int) bool {
	if code == http.StatusTooManyRequests || code == http.StatusRequestTimeout {
		return true
	}
	return code >= 500 && code <= 599
}

func isRetryableError(err error) bool {
	var nerr net.Error
	if errors.As(err, &nerr) {
		if nerr.Timeout() || nerr.Temporary() {
			return true
		}
	}
	// Fallback: string match common TLS/dial issues
	s := err.Error()
	if strings.Contains(s, "timeout") || strings.Contains(strings.ToLower(s), "tls") || strings.Contains(s, "connection reset") {
		return true
	}
	return false
}

func backoff(i int, verbose bool) {
	// Exponential with jitter: base 500ms
	base := 500 * time.Millisecond
	d := time.Duration(1<<i) * base
	// jitter +/- 20%
	jitter := time.Duration(rand.Intn(200)-100) * time.Millisecond
	sleep := d + jitter
	if sleep < 100*time.Millisecond {
		sleep = 100 * time.Millisecond
	}
	if verbose {
		fmt.Printf("retrying in %v...\n", sleep)
	}
	time.Sleep(sleep)
}

func startWebServer(port int) {
	// Create template with custom functions
	funcMap := template.FuncMap{
		"contains": strings.Contains,
	}
	tmpl, err := template.New("index.html").Funcs(funcMap).ParseFS(templateFS, "templates/index.html")
	if err != nil {
		fmt.Println("Error parsing template:", err)
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
		downloadsDir := "downloaded-models"
		if entries, err := os.ReadDir(downloadsDir); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".zip") {
					data.Downloads = append(data.Downloads, entry.Name())
				}
			}
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
		outputDir := "downloaded-models"
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

		// set outZip
		sanitized := strings.ReplaceAll(opt.model, "/", "-")
		sanitized = strings.ReplaceAll(sanitized, ":", "-")
		sanitized = strings.ReplaceAll(sanitized, "@", "-")
		if !strings.HasSuffix(strings.ToLower(sanitized), ".zip") {
			sanitized += ".zip"
		}
		opt.outZip = filepath.Join(opt.outputDir, sanitized)
		currentZip = opt.outZip
		currentProgress = newProgress(0) // total will be set later in run
		currentMessage = "در حال دانلود..."

		ctx, cancel := context.WithCancel(context.Background())
		globalCancel = cancel

		go func() {
			err := run(ctx, opt)
			globalCancel = nil
			currentProgress = nil
			if err != nil {
				if err == context.Canceled {
					currentMessage = "دانلود لغو شد."
				} else {
					currentMessage = fmt.Sprintf("دانلود ناموفق: %s", err.Error())
				}
			} else {
				currentMessage = "دانلود کامل شد."
			}
		}()

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
		if globalCancel != nil {
			globalCancel()
		}
		http.Redirect(w, r, "/", http.StatusFound)
	})

	addr := ":0"
	if port != 0 {
		addr = fmt.Sprintf(":%d", port)
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
	actualPort := listener.Addr().(*net.TCPAddr).Port
	fmt.Printf("Starting web server on :%d\n", actualPort)
	go http.Serve(listener, nil)
	url := fmt.Sprintf("http://localhost:%d", actualPort)
	openBrowser(url)
	select {}
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
