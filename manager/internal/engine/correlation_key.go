package engine

import (
	"strings"

	eventDomain "github.com/hildanku/xemarify/internal/modules/event/domain"
)

// BuildCorrelationKey creates a rule-scoped key used to track threshold state.
// Example: <rule-id>:192.168.1.10
func BuildCorrelationKey(rule CompiledRule, event *eventDomain.Event, matcher *RuleMatcher) (string, bool) {
	if matcher == nil {
		return "", false
	}

	if len(rule.GroupBy) == 0 {
		return rule.ID.String(), true
	}

	var builder strings.Builder
	builder.Grow(64)
	builder.WriteString(rule.ID.String())

	for _, field := range rule.GroupBy {
		value, ok := matcher.FieldValue(event, field)
		if !ok || value == "" {
			return "", false
		}
		builder.WriteByte(':')
		builder.WriteString(value)
	}

	return builder.String(), true
}
