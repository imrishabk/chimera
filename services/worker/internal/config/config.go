package config

import (
	"os"
	"strconv"
)

// IngestConfig holds limits for the file-ingestion pipeline.
// All values have safe defaults matching docs/ingestion-pipeline-spec.md §8.
// Env overrides are optional and additive — existing env keys are untouched.
type IngestConfig struct {
	MaxUploadMBPerFile   int64
	MaxFilesPerRequest   int
	MaxTotalMBPerRequest int64
	MaxPagesPDF          int
	MaxCharsPerFile      int
	WorkerTimeoutMin     int
	QueueSize            int
	AllowURLFetch        bool
	URLFetchTimeoutSec   int
	URLFetchMaxMB        int64
	MaxURLsPerRequest    int
	// URLFetchAllowPrivate permits loopback/private fetch targets.
	// Test-only; never enable in production (SSRF).
	URLFetchAllowPrivate bool
}

func DefaultIngestConfig() IngestConfig {
	return IngestConfig{
		MaxUploadMBPerFile:   25,
		MaxFilesPerRequest:   10,
		MaxTotalMBPerRequest: 100,
		MaxPagesPDF:          500,
		MaxCharsPerFile:      1000000,
		WorkerTimeoutMin:     10,
		QueueSize:            100,
		AllowURLFetch:        false,
		URLFetchTimeoutSec:   10,
		URLFetchMaxMB:        5,
		MaxURLsPerRequest:    10,
		URLFetchAllowPrivate: false,
	}
}

// LoadIngestConfig reads env with fallback to defaults.
// Never fails — invalid values fall back to defaults.
func LoadIngestConfig() IngestConfig {
	cfg := DefaultIngestConfig()
	if v := os.Getenv("MAX_UPLOAD_MB_PER_FILE"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			cfg.MaxUploadMBPerFile = n
		}
	}
	if v := os.Getenv("MAX_FILES_PER_REQUEST"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MaxFilesPerRequest = n
		}
	}
	if v := os.Getenv("MAX_TOTAL_MB_PER_REQUEST"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			cfg.MaxTotalMBPerRequest = n
		}
	}
	if v := os.Getenv("MAX_PAGES_PDF"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MaxPagesPDF = n
		}
	}
	if v := os.Getenv("MAX_CHARS_PER_FILE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MaxCharsPerFile = n
		}
	}
	if v := os.Getenv("INGEST_WORKER_TIMEOUT_MIN"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.WorkerTimeoutMin = n
		}
	}
	if v := os.Getenv("INGEST_QUEUE_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.QueueSize = n
		}
	}
	if v := os.Getenv("ALLOW_URL_FETCH"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.AllowURLFetch = b
		}
	}
	if v := os.Getenv("URL_FETCH_TIMEOUT_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.URLFetchTimeoutSec = n
		}
	}
	if v := os.Getenv("URL_FETCH_MAX_MB"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			cfg.URLFetchMaxMB = n
		}
	}
	if v := os.Getenv("MAX_URLS_PER_REQUEST"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MaxURLsPerRequest = n
		}
	}
	if v := os.Getenv("URL_FETCH_ALLOW_PRIVATE"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.URLFetchAllowPrivate = b
		}
	}
	return cfg
}

// MaxBytesPerFile returns byte limit derived from MB setting.
func (c IngestConfig) MaxBytesPerFile() int64 {
	return c.MaxUploadMBPerFile * 1024 * 1024
}

// MaxTotalBytes returns total request byte limit.
func (c IngestConfig) MaxTotalBytes() int64 {
	return c.MaxTotalMBPerRequest * 1024 * 1024
}
