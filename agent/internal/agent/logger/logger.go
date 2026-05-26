package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
)

const (
	defaultLogPath    = "/var/log/xemarify-agent/agent.log"
	defaultMaxSizeMB  = 100
	defaultMaxBackups = 5
	megabyte          = 1024 * 1024
)

var (
	mu      sync.Mutex
	file    *os.File
	current *log.Logger
)

type Config struct {
	Path       string
	MaxSizeMB  int64
	MaxBackups int
}

// Init sets up the global logger to write to a dedicated file
// If the file cannot be opened, it falls back to stderr so the agent
// never silently loses diagnostic output
func Init(cfg Config) {
	if cfg.Path == "" {
		cfg.Path = defaultLogPath
	}
	if cfg.MaxSizeMB <= 0 {
		cfg.MaxSizeMB = defaultMaxSizeMB
	}
	if cfg.MaxBackups <= 0 {
		cfg.MaxBackups = defaultMaxBackups
	}

	mu.Lock()
	defer mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(cfg.Path), 0o750); err != nil {
		log.SetOutput(os.Stderr)
		log.Printf("logger: failed to create log directory %s, falling back to stderr: %v", filepath.Dir(cfg.Path), err)
		current = log.Default()
		return
	}

	f, err := os.OpenFile(cfg.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
	if err != nil {
		log.SetOutput(os.Stderr)
		log.Printf("logger: failed to open log file %s, falling back to stderr: %v", cfg.Path, err)
		current = log.Default()
		return
	}

	file = f
	log.SetOutput(f)
	log.SetFlags(log.LstdFlags)
	current = log.Default()

	rotateLocked(cfg)
}

// rotateLocked checks the current log file size and rotates if needed
// Must be called with mu held
func rotateLocked(cfg Config) {
	if file == nil {
		return
	}

	info, err := file.Stat()
	if err != nil {
		return
	}

	if info.Size() < cfg.MaxSizeMB*megabyte {
		return
	}

	_ = file.Close()

	// Shift existing backups: agent.log.4 -> drop, agent.log.3 -> .4, etc.
	for i := cfg.MaxBackups - 1; i >= 1; i-- {
		old := fmt.Sprintf("%s.%d", cfg.Path, i)
		newer := fmt.Sprintf("%s.%d", cfg.Path, i+1)
		_ = os.Rename(old, newer)
	}
	_ = os.Rename(cfg.Path, cfg.Path+".1")

	f, err := os.OpenFile(cfg.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
	if err != nil {
		log.SetOutput(os.Stderr)
		file = nil
		return
	}

	file = f
	log.SetOutput(f)
}

func Close() {
	mu.Lock()
	defer mu.Unlock()

	if file != nil {
		_ = file.Sync()
		_ = file.Close()
		file = nil
	}
}

func Writer() io.Writer {
	mu.Lock()
	defer mu.Unlock()

	if file != nil {
		return file
	}
	return os.Stderr
}
