package collector

import (
	"net"
	"net/url"
	pathpkg "path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// WebProbeConfig controls semantic classification of Nginx access-log lines.
// SensitivePaths are URL path prefixes, not regular expressions.
type WebProbeConfig struct {
	Enabled        bool
	SensitivePaths []string
}

var nginxAccessLinePattern = regexp.MustCompile(`^(\S+) \S+ \S+ \[([^\]]+)\] "([^"]*)" ([0-9]{3}) (?:\S+)(?: |$)`)

// parseWebProbe converts a standard Nginx combined access-log entry into a
// semantic security event when its request path is configured as sensitive.
// It returns false for non-access-log lines, malformed requests, and ordinary
// paths so those lines retain the normal filelog representation.
func parseWebProbe(line, hostname string, sensitivePaths []string) (map[string]interface{}, bool) {
	matches := nginxAccessLinePattern.FindStringSubmatch(strings.TrimSpace(line))
	if len(matches) == 0 {
		return nil, false
	}

	method, requestTarget, ok := parseRequestLine(matches[3])
	if !ok {
		return nil, false
	}

	requestPath, ok := normalizeRequestPath(requestTarget)
	if !ok {
		return nil, false
	}
	matchRule, ok := matchSensitivePath(requestPath, sensitivePaths)
	if !ok {
		return nil, false
	}

	status, err := strconv.Atoi(matches[4])
	if err != nil {
		return nil, false
	}

	normalized := map[string]interface{}{
		"schema":        "xemarify.web_probe.v1",
		"event_type":    "web_probe_admin_path",
		"source_ip":     normalizeSourceIP(matches[1]),
		"target_host":   hostname,
		"http_method":   method,
		"request_path":  requestPath,
		"http_status":   status,
		"path_category": "admin_panel",
		"match_rule":    "prefix:" + matchRule,
		"web_server":    "nginx",
	}
	if occurredAt, err := time.Parse("02/Jan/2006:15:04:05 -0700", matches[2]); err == nil {
		normalized["occurred_at"] = occurredAt.UTC().Format(time.RFC3339)
	}
	return normalized, true
}

func parseRequestLine(request string) (method, target string, ok bool) {
	fields := strings.Fields(request)
	if len(fields) != 3 || !strings.HasPrefix(fields[2], "HTTP/") {
		return "", "", false
	}
	return strings.ToUpper(fields[0]), fields[1], true
}

// normalizeRequestPath removes the query string and canonicalizes path
// separators and percent-encoding before a configured prefix is compared.
func normalizeRequestPath(requestTarget string) (string, bool) {
	parsed, err := url.ParseRequestURI(requestTarget)
	if err != nil || parsed.Path == "" {
		return "", false
	}
	decoded, err := url.PathUnescape(parsed.EscapedPath())
	if err != nil {
		return "", false
	}
	decoded = strings.ReplaceAll(decoded, "\\", "/")
	cleaned := pathpkg.Clean("/" + strings.TrimPrefix(decoded, "/"))
	if cleaned == "." || !strings.HasPrefix(cleaned, "/") {
		return "", false
	}
	return strings.ToLower(cleaned), true
}

func matchSensitivePath(requestPath string, sensitivePaths []string) (string, bool) {
	for _, configuredPath := range sensitivePaths {
		normalizedPath, ok := normalizeConfiguredPath(configuredPath)
		if !ok {
			continue
		}
		if requestPath == normalizedPath || strings.HasPrefix(requestPath, normalizedPath+"/") {
			return normalizedPath, true
		}
	}
	return "", false
}

func normalizeConfiguredPath(configuredPath string) (string, bool) {
	configuredPath = strings.TrimSpace(configuredPath)
	if configuredPath == "" || strings.ContainsAny(configuredPath, "?#") {
		return "", false
	}
	return normalizeRequestPath(configuredPath)
}

func normalizeSourceIP(value string) string {
	value = strings.TrimSpace(value)
	if parsed := net.ParseIP(value); parsed != nil {
		return parsed.String()
	}
	return value
}
