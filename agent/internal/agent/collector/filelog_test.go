package collector

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hildanku/xemarify-agent/internal/agent/model"
)

func TestDetectService(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "nginx path", path: "/var/log/nginx/access.log", want: "nginx"},
		{name: "apache path", path: "/var/log/apache2/error.log", want: "apache"},
		{name: "auth log path", path: "/var/log/auth.log", want: "auth"},
		{name: "unknown path", path: "/tmp/custom.log", want: "unknown"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := detectService(testCase.path)
			if got != testCase.want {
				t.Fatalf("detectService(%q) = %q, want %q", testCase.path, got, testCase.want)
			}
		})
	}
}

func TestAuthLogSourceIP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		line string
		want string
	}{
		{name: "failed password", line: "sshd[1]: Failed password for root from 127.0.0.1 port 22 ssh2", want: "127.0.0.1"},
		{name: "pam auth failure", line: "sshd[1]: pam_unix(sshd:auth): authentication failure rhost=2001:db8::10 user=root", want: "2001:db8::10"},
		{name: "no source IP", line: "sshd[1]: Connection closed by user root", want: ""},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := authLogSourceIP(testCase.line); got != testCase.want {
				t.Fatalf("authLogSourceIP(%q) = %q, want %q", testCase.line, got, testCase.want)
			}
		})
	}
}

func TestGuessSeverity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		msg  string
		want string
	}{
		{msg: "request completed", want: "INFO"},
		{msg: "WARN upstream timeout", want: "MEDIUM"},
		{msg: "database ERROR connection refused", want: "HIGH"},
	}

	for _, testCase := range tests {
		got := guessSeverity(testCase.msg)
		if got != testCase.want {
			t.Fatalf("guessSeverity(%q) = %q, want %q", testCase.msg, got, testCase.want)
		}
	}
}

func TestReadNewLines_IncrementalRead(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "access.log")

	if err := os.WriteFile(logPath, []byte("old-line\n"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	baseSize, err := fileSize(logPath)
	if err != nil {
		t.Fatalf("fileSize: %v", err)
	}

	appendContent := "new-line-1\nnew-line-2\n"
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open append: %v", err)
	}
	if _, err := f.WriteString(appendContent); err != nil {
		_ = f.Close()
		t.Fatalf("append write: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close append file: %v", err)
	}

	lines, newOffset, err := readNewLines(logPath, baseSize)
	if err != nil {
		t.Fatalf("readNewLines: %v", err)
	}

	if len(lines) != 2 {
		t.Fatalf("expected 2 new lines, got %d", len(lines))
	}
	if lines[0] != "new-line-1" || lines[1] != "new-line-2" {
		t.Fatalf("unexpected lines: %#v", lines)
	}
	if newOffset <= baseSize {
		t.Fatalf("expected newOffset > baseSize, got newOffset=%d baseSize=%d", newOffset, baseSize)
	}
}

func TestReadNewLines_TruncateResetsOffset(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "error.log")

	if err := os.WriteFile(logPath, []byte("old-content\n"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	oldOffset, err := fileSize(logPath)
	if err != nil {
		t.Fatalf("fileSize: %v", err)
	}

	if err := os.WriteFile(logPath, []byte("new\n"), 0o600); err != nil {
		t.Fatalf("truncate rewrite file: %v", err)
	}

	lines, _, err := readNewLines(logPath, oldOffset)
	if err != nil {
		t.Fatalf("readNewLines: %v", err)
	}

	if len(lines) != 1 || lines[0] != "new" {
		t.Fatalf("truncate handling failed, got lines=%#v", lines)
	}
}

func TestExpandFileLogPaths(t *testing.T) {
	tempDir := t.TempDir()
	first := filepath.Join(tempDir, "log-2026-07-29.php")
	second := filepath.Join(tempDir, "log-2026-07-30.php")
	for _, path := range []string{first, second} {
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			t.Fatalf("create %s: %v", path, err)
		}
	}

	paths := expandFileLogPaths([]string{filepath.Join(tempDir, "log-*.php")})
	if len(paths) != 2 {
		t.Fatalf("expanded paths = %#v, want two matches", paths)
	}
}

func TestRunFileLog_EmitsOnlyNewLines(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "nginx-access.log")

	if err := os.WriteFile(logPath, []byte("existing-line\n"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	output := make(chan model.IngestEvent, 16)
	go RunFileLog(ctx, []string{logPath}, "test-host", output, 20*time.Millisecond, WebProbeConfig{})

	time.Sleep(40 * time.Millisecond)

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open append: %v", err)
	}
	if _, err := f.WriteString("WARN new-line\n"); err != nil {
		_ = f.Close()
		t.Fatalf("append write: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close append file: %v", err)
	}

	select {
	case event := <-output:
		if event.InputType != "filelog" {
			t.Fatalf("unexpected input type: %s", event.InputType)
		}
		if event.Hostname != "test-host" {
			t.Fatalf("unexpected hostname: %s", event.Hostname)
		}
		if event.Facility != "nginx" {
			t.Fatalf("unexpected facility: %s", event.Facility)
		}
		if event.Severity != "MEDIUM" {
			t.Fatalf("unexpected severity: %s", event.Severity)
		}
		if event.SourceName == "" {
			t.Fatal("source_name should not be empty")
		}
		if event.Attributes == nil {
			t.Fatal("attributes should not be nil")
		}
		if event.Attributes["source_type"] != "filelog" {
			t.Fatalf("unexpected attributes.source_type: %#v", event.Attributes["source_type"])
		}
		if event.Message != "WARN new-line" {
			t.Fatalf("unexpected message: %s", event.Message)
		}
		if _, ok := event.Normalized["event_type"]; ok {
			t.Fatalf("raw filelog must not set event_type: %#v", event.Normalized)
		}
	case <-time.After(800 * time.Millisecond):
		t.Fatal("timed out waiting for filelog event")
	}

	cancel()
}

func TestRunFileLog_ParsesAuthLogSourceIP(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "auth.log")
	if err := os.WriteFile(logPath, nil, 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	output := make(chan model.IngestEvent, 1)
	go RunFileLog(ctx, []string{logPath}, "test-host", output, 20*time.Millisecond, WebProbeConfig{})
	time.Sleep(40 * time.Millisecond)

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open append: %v", err)
	}
	if _, err := f.WriteString("sshd[1]: Failed password for root from 127.0.0.1 port 22 ssh2\n"); err != nil {
		_ = f.Close()
		t.Fatalf("append auth event: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	select {
	case event := <-output:
		if event.Facility != "auth" || event.Category != "auth_log" {
			t.Fatalf("facility/category = %q/%q, want auth/auth_log", event.Facility, event.Category)
		}
		if event.SourceIP != "127.0.0.1" {
			t.Fatalf("source_ip = %q, want 127.0.0.1", event.SourceIP)
		}
		if event.Normalized["source_ip"] != "127.0.0.1" {
			t.Fatalf("normalized.source_ip = %#v", event.Normalized["source_ip"])
		}
	case <-time.After(800 * time.Millisecond):
		t.Fatal("timed out waiting for auth log event")
	}
}
