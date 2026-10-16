package collector

import "testing"

func TestParseWebProbe_ClassifiesConfiguredNginxPath(t *testing.T) {
	line := `203.0.113.10 - - [31/Jul/2026:10:14:32 +0700] "GET /%61dmin/login?next=%2Fdashboard HTTP/1.1" 404 153 "-" "Mozilla/5.0"`

	normalized, ok := parseWebProbe(line, "ppdb-host", []string{"/admin", "/wp-admin"})
	if !ok {
		t.Fatal("expected sensitive Nginx request to be classified")
	}
	assertWebProbeValue(t, normalized, "event_type", "web_probe_admin_path")
	assertWebProbeValue(t, normalized, "source_ip", "203.0.113.10")
	assertWebProbeValue(t, normalized, "request_path", "/admin/login")
	assertWebProbeValue(t, normalized, "http_method", "GET")
	assertWebProbeValue(t, normalized, "http_status", 404)
	assertWebProbeValue(t, normalized, "match_rule", "prefix:/admin")
	assertWebProbeValue(t, normalized, "occurred_at", "2026-07-31T03:14:32Z")
}

func TestParseWebProbe_DoesNotMatchSimilarPath(t *testing.T) {
	line := `198.51.100.24 - - [31/Jul/2026:10:14:32 +0700] "GET /administrator HTTP/1.1" 404 153 "-" "scanner"`

	if _, ok := parseWebProbe(line, "ppdb-host", []string{"/admin"}); ok {
		t.Fatal("/administrator must not match the /admin prefix rule")
	}
}

func TestParseWebProbe_DoesNotClassifyMalformedOrOrdinaryLines(t *testing.T) {
	tests := []string{
		`not an Nginx access line`,
		`203.0.113.10 - - [31/Jul/2026:10:14:32 +0700] "GET /news HTTP/1.1" 200 153 "-" "Mozilla/5.0"`,
		`203.0.113.10 - - [31/Jul/2026:10:14:32 +0700] "GET /admin%ZZ HTTP/1.1" 400 153 "-" "Mozilla/5.0"`,
	}
	for _, line := range tests {
		if _, ok := parseWebProbe(line, "ppdb-host", []string{"/admin"}); ok {
			t.Fatalf("line should not be classified: %s", line)
		}
	}
}

func assertWebProbeValue(t *testing.T, values map[string]interface{}, key string, want interface{}) {
	t.Helper()
	if got := values[key]; got != want {
		t.Fatalf("%s = %#v, want %#v", key, got, want)
	}
}
