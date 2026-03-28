package collector

import (
	"testing"
)

func TestHostFromCIDR(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "ipv4 cidr", input: "192.168.10.20/24", want: "192.168.10.20"},
		{name: "ipv6 cidr", input: "fe80::1/64", want: "fe80::1"},
		{name: "plain host", input: "10.0.0.1", want: "10.0.0.1"},
		{name: "empty", input: "", want: ""},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := hostFromCIDR(testCase.input)
			if got != testCase.want {
				t.Fatalf("hostFromCIDR(%q) = %q, want %q", testCase.input, got, testCase.want)
			}
		})
	}
}

func TestDedupStrings(t *testing.T) {
	t.Parallel()

	in := []string{"10.0.0.1", "10.0.0.2", "10.0.0.1", "10.0.0.3", "10.0.0.2"}
	out := dedupStrings(in)

	if len(out) != 3 {
		t.Fatalf("expected 3 unique values, got %d (%#v)", len(out), out)
	}
	if out[0] != "10.0.0.1" || out[1] != "10.0.0.2" || out[2] != "10.0.0.3" {
		t.Fatalf("unexpected dedup order/result: %#v", out)
	}
}

func TestBinaryExists_EmptyName(t *testing.T) {
	t.Parallel()

	if binaryExists("") {
		t.Fatal("binaryExists(\"\") must be false")
	}
}

func TestBuildInventoryEvent_BasicShape(t *testing.T) {
	t.Parallel()

	event := buildInventoryEvent("test-host")

	if event.InputType != "inventory" {
		t.Fatalf("unexpected input_type: %s", event.InputType)
	}
	if event.Category != "inventory" {
		t.Fatalf("unexpected category: %s", event.Category)
	}
	if event.Hostname != "test-host" {
		t.Fatalf("unexpected hostname: %s", event.Hostname)
	}
	if event.Message == "" {
		t.Fatal("message should not be empty")
	}

	if event.Normalized == nil {
		t.Fatal("normalized should not be nil")
	}
	if event.Normalized["event_type"] != "inventory" {
		t.Fatalf("unexpected event_type in normalized: %#v", event.Normalized["event_type"])
	}
	if event.Normalized["hostname"] != "test-host" {
		t.Fatalf("unexpected normalized hostname: %#v", event.Normalized["hostname"])
	}
}
