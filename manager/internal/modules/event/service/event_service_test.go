package service

import (
	"testing"

	"github.com/hildanku/xemarify/internal/modules/event/domain"
)

func TestEnsureEventType_PreservesExisting(t *testing.T) {
	event := &domain.Event{
		Message:    "login failed for user",
		Normalized: map[string]interface{}{"event_type": "custom_type"},
	}

	ensureEventType(event)

	got := event.Normalized["event_type"]
	if got != "custom_type" {
		t.Fatalf("event_type should be preserved, got=%v", got)
	}
}

func TestDeriveEventType(t *testing.T) {
	tests := []struct {
		name  string
		event *domain.Event
		want  string
	}{
		{
			name: "login failed",
			event: &domain.Event{
				Message: "Failed password for invalid user root from 10.0.0.1",
				Raw:     "Failed password for invalid user root",
			},
			want: "ssh_invalid_user",
		},
		{
			name: "login success",
			event: &domain.Event{
				Message: "Accepted password for user",
			},
			want: "login_success",
		},
		{
			name: "sudo failed",
			event: &domain.Event{
				Message: "sudo: authentication failure; logname=user",
			},
			want: "sudo_failed",
		},
		{
			name: "sudo used",
			event: &domain.Event{
				Message: "sudo: session opened for user root",
			},
			want: "sudo_used",
		},
		{
			name: "web 401",
			event: &domain.Event{
				Message: "GET /admin 401 Unauthorized",
			},
			want: "web_401",
		},
		{
			name: "web 403",
			event: &domain.Event{
				Message: "GET /admin 403 Forbidden",
			},
			want: "web_403",
		},
		{
			name: "web 500",
			event: &domain.Event{
				Message: "GET /api 500 Server Error",
			},
			want: "web_500",
		},
		{
			name: "port scan",
			event: &domain.Event{
				Message: "possible port scan detected from 10.0.0.2",
			},
			want: "port_scan_detected",
		},
		{
			name: "process suspicious",
			event: &domain.Event{
				Message: "suspicious process execution detected",
			},
			want: "process_exec_suspicious",
		},
		{
			name: "service installed",
			event: &domain.Event{
				Message: "apt install nginx completed",
			},
			want: "service_installed",
		},
		{
			name: "service started",
			event: &domain.Event{
				Message: "systemd started nginx",
			},
			want: "service_started",
		},
		{
			name: "user created",
			event: &domain.Event{
				Message: "useradd new user created",
			},
			want: "user_created",
		},
		{
			name: "file integrity",
			event: &domain.Event{
				Message: "file integrity violation detected",
			},
			want: "file_integrity_changed",
		},
		{
			name: "privilege escalation",
			event: &domain.Event{
				Message: "privilege escalation attempt detected",
			},
			want: "privilege_escalation",
		},
		{
			name: "http status from normalized",
			event: &domain.Event{
				Normalized: map[string]interface{}{"status": "401"},
			},
			want: "web_401",
		},
		{
			name: "no match",
			event: &domain.Event{
				Message: "some random log message",
			},
			want: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := deriveEventType(test.event)
			if got != test.want {
				t.Fatalf("deriveEventType() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestEnsureEventType_SetsDerived(t *testing.T) {
	event := &domain.Event{
		Message:    "GET /admin 401 Unauthorized",
		Normalized: map[string]interface{}{},
	}

	ensureEventType(event)

	got, ok := event.Normalized["event_type"]
	if !ok {
		t.Fatalf("expected event_type to be set")
	}
	if got != "web_401" {
		t.Fatalf("event_type = %v, want %q", got, "web_401")
	}
}
