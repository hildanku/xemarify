package transport

// ListAlertsQuery holds the query parameters for GET /api/v1/alerts.
type ListAlertsQuery struct {
	Search        string `form:"search"`
	Order         string `form:"order,default=desc" binding:"omitempty,oneof=asc desc"`
	Limit         int    `form:"limit,default=10" binding:"omitempty,min=1,max=100"`
	Cursor        string `form:"cursor"`
	Severity      string `form:"severity"`
	Status        string `form:"status"`
	RuleID        string `form:"rule_id"`
	TriggeredFrom string `form:"triggered_from"`
	TriggeredTo   string `form:"triggered_to"`
}

type UpdateAlertStatusRequest struct {
	Status string `json:"status" binding:"required"`
}