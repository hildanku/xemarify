package engine

import (
	"encoding/json"
	"strconv"
	"strings"

	eventDomain "github.com/hildanku/xemarify/internal/modules/event/domain"
)

type RuleMatcher struct{}

func NewRuleMatcher() *RuleMatcher {
	return &RuleMatcher{}
}

// EventType resolves the runtime event type used for indexed rule lookup.
// Priority: normalized.event_type -> normalized.type -> category.
func (m *RuleMatcher) EventType(event *eventDomain.Event) string {
	if event == nil {
		return ""
	}

	if value, ok := m.fieldFromNormalized(event, "event_type"); ok {
		return strings.ToLower(strings.TrimSpace(value))
	}
	if value, ok := m.fieldFromNormalized(event, "type"); ok {
		return strings.ToLower(strings.TrimSpace(value))
	}
	if event.Category != "" {
		return strings.ToLower(strings.TrimSpace(event.Category))
	}
	if value, ok := m.fieldFromNormalized(event, "category"); ok {
		return strings.ToLower(strings.TrimSpace(value))
	}

	return ""
}

func (m *RuleMatcher) FieldValue(event *eventDomain.Event, field string) (string, bool) {
	if event == nil {
		return "", false
	}

	normalizedField := normalizeGroupField(field)

	switch normalizedField {
	case "src_ip", "source_ip":
		if event.SourceIP != "" {
			return event.SourceIP, true
		}
		if value, ok := m.fieldFromNormalized(event, "source_ip"); ok {
			return value, true
		}
		return m.fieldFromNormalized(event, "src_ip")
	case "hostname":
		if event.Hostname != "" {
			return event.Hostname, true
		}
		return m.fieldFromNormalized(event, "hostname")
	case "severity":
		if event.Severity != "" {
			return event.Severity, true
		}
		return m.fieldFromNormalized(event, "severity")
	case "category":
		if event.Category != "" {
			return event.Category, true
		}
		return m.fieldFromNormalized(event, "category")
	case "facility":
		if event.Facility != "" {
			return event.Facility, true
		}
		return m.fieldFromNormalized(event, "facility")
	case "input_type":
		if event.InputType != "" {
			return event.InputType, true
		}
		return m.fieldFromNormalized(event, "input_type")
	case "agent_id":
		if event.AgentID.String() != "" {
			return event.AgentID.String(), true
		}
	}

	if value, ok := m.fieldFromNormalized(event, normalizedField); ok {
		return value, true
	}

	return "", false
}

func (m *RuleMatcher) fieldFromNormalized(event *eventDomain.Event, key string) (string, bool) {
	if event.Normalized == nil {
		return "", false
	}

	raw, ok := event.Normalized[key]
	if !ok || raw == nil {
		return "", false
	}

	value := stringifyValue(raw)
	if value == "" {
		return "", false
	}

	return value, true
}

func stringifyValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case bool:
		return strconv.FormatBool(v)
	case int:
		return strconv.Itoa(v)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case json.Number:
		return v.String()
	default:
		return ""
	}
}

func normalizeGroupField(field string) string {
	normalized := strings.ToLower(strings.TrimSpace(field))
	if normalized == "src_ip" {
		return "source_ip"
	}
	return normalized
}
