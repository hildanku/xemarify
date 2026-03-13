package transport

// ListRulesQuery holds the query parameters for GET /api/v1/rules.
type ListRulesQuery struct {
	Search  string `form:"search"`
	SortBy  string `form:"sort_by,default=created_at"`
	Sort    string `form:"sort"`
	Order   string `form:"order,default=desc" binding:"omitempty,oneof=asc desc"`
	Page    int    `form:"page,default=1"     binding:"omitempty,min=1"`
	Limit   int    `form:"limit,default=10"   binding:"omitempty,min=1,max=100"`
	Offset  int    `form:"offset,default=0"   binding:"omitempty,min=0"`
	Level   string `form:"level"              binding:"omitempty,oneof=INFO LOW MEDIUM HIGH CRITICAL"`
	Enabled *bool  `form:"enabled"`
}

// RuleConditionRequest is the JSON body for a rule condition.
type RuleConditionRequest struct {
	EventType string   `json:"event_type" binding:"required"`
	GroupBy   []string `json:"group_by"`
	Threshold int      `json:"threshold"  binding:"required,min=1"`
	WindowSec int      `json:"window_sec" binding:"required,min=1"`
	Severity  string   `json:"severity"   binding:"omitempty,oneof=INFO LOW MEDIUM HIGH CRITICAL"`
}

// CreateRuleRequest is the JSON body for POST /api/v1/rules.
type CreateRuleRequest struct {
	Name        string               `json:"name"        binding:"required,min=3,max=120"`
	Description string               `json:"description" binding:"omitempty,max=500"`
	Level       string               `json:"level"       binding:"required,oneof=INFO LOW MEDIUM HIGH CRITICAL"`
	Enabled     bool                 `json:"enabled"`
	Condition   RuleConditionRequest `json:"condition"   binding:"required"`
	Tags        []string             `json:"tags"`
}

// UpdateRuleRequest is the JSON body for PUT /api/v1/rules/:id.
type UpdateRuleRequest struct {
	Name        string                `json:"name"        binding:"omitempty,min=3,max=120"`
	Description string                `json:"description" binding:"omitempty,max=500"`
	Level       string                `json:"level"       binding:"omitempty,oneof=INFO LOW MEDIUM HIGH CRITICAL"`
	Enabled     *bool                 `json:"enabled"`
	Condition   *RuleConditionRequest `json:"condition"`
	Tags        []string              `json:"tags"`
}
