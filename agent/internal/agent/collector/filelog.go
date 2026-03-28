package collector

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hildanku/xemarify-agent/internal/agent/model"
)

// we need func for detect service like nginx or apache
// we need guess the severity
// we need func to get newLines
// we need runFileLog

func RunFileLog(
	ctx context.Context,
	paths []string,
	hostname string,
	output chan<- model.IngestEvent,
	pollInterval time.Duration,
) {
	if pollInterval <= 0 {
		pollInterval = 1 * time.Second
	}

	offsets := make(map[string]int64, len(paths))
	for _, path := range paths {
		size, err := fileSize(path)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				log.Printf("filelog init stat error path=%s: %v", path, err)
			}
			offsets[path] = 0
			continue
		}
		// start dari EOF biar tidak kirim backlog lama saat startup
		offsets[path] = size
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, path := range paths {
				lines, newOffset, err := readNewLines(path, offsets[path])
				if err != nil {
					if !errors.Is(err, os.ErrNotExist) {
						log.Printf("filelog read error path=%s: %v", path, err)
					}
					continue
				}

				offsets[path] = newOffset
				service := detectService(path)

				for _, line := range lines {
					msg := strings.TrimSpace(line)
					if msg == "" {
						continue
					}

					severity := guessSeverity(msg)
					sourceName := service + ":" + filepath.Base(path)
					attributes := map[string]interface{}{
						"source_type": "filelog",
						"source_name": sourceName,
						"file_path":   path,
						"service":     service,
						"severity":    severity,
					}
					event := model.IngestEvent{
						EventTime:  time.Now().UTC(),
						Hostname:   hostname,
						SourceIP:   "",
						InputType:  "filelog",
						SourceName: sourceName,
						Facility:   service,
						Severity:   severity,
						Category:   "web_log",
						Message:    msg,
						Raw:        msg,
						Attributes: attributes,
						Normalized: map[string]interface{}{
							"event_type":  "filelog",
							"source_type": "filelog",
							"source_name": sourceName,
							"file_path":   path,
							"service":     service,
							"severity":    severity,
						},
					}

					select {
					case output <- event:
					case <-ctx.Done():
						return
					default:
						log.Printf("dropping filelog event: forwarder queue is full path=%s", path)
					}
				}
			}
		}
	}
}

func readNewLines(path string, offset int64) ([]string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, offset, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, offset, err
	}

	// truncate file if it was rotated
	if info.Size() < offset {
		offset = 0
	}

	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return nil, offset, err
	}

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	lines := make([]string, 0, 32)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, offset, err
	}

	newOffset, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, offset, err
	}

	return lines, newOffset, nil
}

func fileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func detectService(path string) string {
	lower := strings.ToLower(path)
	switch {
	case strings.Contains(lower, "nginx"):
		return "nginx"
	case strings.Contains(lower, "apache"):
		return "apache"
	default:
		return "unknown"
	}
}

func guessSeverity(message string) string {
	m := strings.ToLower(message)
	switch {
	case strings.Contains(m, "error"), strings.Contains(m, "fatal"), strings.Contains(m, "panic"), strings.Contains(m, "crit"):
		return "HIGH"
	case strings.Contains(m, "warn"):
		return "MEDIUM"
	default:
		return "INFO"
	}
}
