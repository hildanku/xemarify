package transport

// ListAlertsQuery holds the query parameters for GET /api/v1/alerts.
type ListAlertsQuery struct {
	Search        string `form:"search"`
	SortBy        string `form:"sort_by,default=triggered_at"`
	Sort          string `form:"sort"`
	Order         string `form:"order,default=desc" binding:"omitempty,oneof=asc desc"`
	Page          int    `form:"page,default=1" binding:"omitempty,min=1"`
	Limit         int    `form:"limit,default=10" binding:"omitempty,min=1,max=100"`
	Offset        int    `form:"offset,default=0" binding:"omitempty,min=0"`
	Severity      string `form:"severity"`
	Status        string `form:"status"`
	RuleID        string `form:"rule_id"`
	TriggeredFrom string `form:"triggered_from"`
	TriggeredTo   string `form:"triggered_to"`
}

type UpdateAlertStatusRequest struct {
	Status string `json:"status" binding:"required"`
}
