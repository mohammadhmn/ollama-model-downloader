package config

import (
	"flag"
	"fmt"
	"runtime"
	"strings"
	"time"
)

type Config struct {
	Registry    string
	Platform    string
	Concurrency int
	Verbose     bool
	KeepStaging bool
	Retries     int
	Timeout     time.Duration
	InsecureTLS bool
	Port        int
	OutputDir   string
}

func Parse() (*Config, error) {
	cfg := &Config{}

	flag.StringVar(&cfg.Registry, "registry", "https://registry.ollama.ai", "registry base URL")
	flag.IntVar(&cfg.Concurrency, "concurrency", 4, "number of concurrent blob downloads")
	flag.BoolVar(&cfg.Verbose, "v", false, "verbose logging")
	flag.BoolVar(&cfg.KeepStaging, "keep-staging", false, "keep staging directory (do not delete after zip)")
	flag.IntVar(&cfg.Retries, "retries", 3, "retry attempts for transient errors")

	var timeoutSec int
	flag.IntVar(&timeoutSec, "timeout", 0, "overall request timeout seconds (0 = no limit)")
	flag.BoolVar(&cfg.InsecureTLS, "insecure", false, "skip TLS verification (NOT recommended)")

	defaultPlatform := fmt.Sprintf("linux/%s", archFromGo(runtime.GOARCH))
	flag.StringVar(&cfg.Platform, "platform", defaultPlatform, "target platform (linux/amd64 or linux/arm64)")
	flag.StringVar(&cfg.OutputDir, "output-dir", "downloaded-models", "directory to save downloaded models")
	flag.IntVar(&cfg.Port, "port", 5050, "port to listen on (5050 by default, 0 for random)")

	flag.Parse()

	if timeoutSec > 0 {
		cfg.Timeout = time.Duration(timeoutSec) * time.Second
	}

	return cfg, nil
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

func SanitizeModelName(model string) string {
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
