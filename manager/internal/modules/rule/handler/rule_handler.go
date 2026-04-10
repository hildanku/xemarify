package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/infrastructure/middleware"
	"github.com/hildanku/xemarify/internal/modules/rule/domain"
	ruleRepo "github.com/hildanku/xemarify/internal/modules/rule/repository"
	ruleService "github.com/hildanku/xemarify/internal/modules/rule/service"
	"github.com/hildanku/xemarify/internal/modules/rule/transport"
	"github.com/hildanku/xemarify/pkg/query"
	"github.com/hildanku/xemarify/pkg/response"
	"github.com/sirupsen/logrus"
)

// RuleHandler handles HTTP requests for the rule management endpoints.
type RuleHandler struct {
	svc *ruleService.RuleService
	log *logrus.Logger
}

// NewRuleHandler constructs a RuleHandler.
func NewRuleHandler(svc *ruleService.RuleService, log *logrus.Logger) *RuleHandler {
	return &RuleHandler{svc: svc, log: log}
}

// Register wires the rule routes onto the given router group.
// The group must already have JWT + RBAC middleware applied.
func (h *RuleHandler) Register(rg *gin.RouterGroup) {
	rg.GET("", h.List)
	rg.POST("", h.Create)
	rg.GET("/:id", h.GetByID)
	rg.PUT("/:id", h.Update)
	rg.DELETE("/:id", h.Delete)
}

// List handles GET /api/v1/rules.
func (h *RuleHandler) List(c *gin.Context) {
	var q transport.ListRulesQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Write(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	sortBy := q.SortBy
	if q.Sort != "" {
		sortBy = q.Sort
	}
	offset := q.Offset
	if offset == 0 && q.Page > 1 {
		offset = (q.Page - 1) * q.Limit
	}

	filter := ruleRepo.ListFilter{
		BaseFilter: query.BaseFilter{
			Search: q.Search,
			SortBy: sortBy,
			Order:  query.SortOrder(q.Order),
			Limit:  q.Limit,
			Offset: offset,
		},
		Level:   q.Level,
		Enabled: q.Enabled,
	}

	rules, total, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		h.log.WithError(err).Error("failed to list rules")
		response.Write(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	items := make([]*transport.RuleResponse, 0, len(rules))
	for _, r := range rules {
		items = append(items, transport.ToRuleResponse(r))
	}

	totalPages := 0
	if filter.Limit > 0 {
		totalPages = (total + filter.Limit - 1) / filter.Limit
	}

	response.Write(c, http.StatusOK, "rules retrieved", transport.ListRulesResponse{
		Items: items,
		Metadata: transport.ListRulesMetadata{
			Total:      total,
			TotalPages: totalPages,
			Limit:      filter.Limit,
			Offset:     filter.Offset,
		},
	})
}

// Create handles POST /api/v1/rules.
func (h *RuleHandler) Create(c *gin.Context) {
	var req transport.CreateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Write(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	claims := middleware.UserClaimsFromContext(c)

	r, err := h.svc.Create(c.Request.Context(), ruleService.CreateRuleInput{
		Name:        req.Name,
		Description: req.Description,
		Level:       req.Level,
		Enabled:     req.Enabled,
		CreatedByID: &claims.UserID,
		Condition: domain.RuleCondition{
			Type:                  req.Condition.Type,
			EventType:             req.Condition.EventType,
			GroupBy:               req.Condition.GroupBy,
			Threshold:             req.Condition.Threshold,
			WindowSec:             req.Condition.WindowSec,
			Severity:              req.Condition.Severity,
			SequenceSteps:         req.Condition.SequenceSteps,
			CorrelationEventTypes: req.Condition.CorrelationEventTypes,
			MinDistinctEventTypes: req.Condition.MinDistinctEventTypes,
			BaselineWindowSec:     req.Condition.BaselineWindowSec,
			SpikeFactor:           req.Condition.SpikeFactor,
			AnomalyMinCount:       req.Condition.AnomalyMinCount,
		},
		Tags: tags,
	}, claims, c.ClientIP())
	if err != nil {
		if errors.Is(err, ruleService.ErrInvalidRuleCondition) {
			response.Write(c, http.StatusBadRequest, err.Error(), nil)
			return
		}
		h.log.WithError(err).Error("failed to create rule")
		response.Write(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	response.Write(c, http.StatusCreated, "rule created", transport.ToRuleResponse(r))
}

// GetByID handles GET /api/v1/rules/:id.
func (h *RuleHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Write(c, http.StatusBadRequest, "invalid rule id", nil)
		return
	}

	r, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		h.log.WithError(err).Error("failed to get rule")
		response.Write(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}
	if r == nil {
		response.Write(c, http.StatusNotFound, "rule not found", nil)
		return
	}

	response.Write(c, http.StatusOK, "rule retrieved", transport.ToRuleResponse(r))
}

// Update handles PUT /api/v1/rules/:id.
func (h *RuleHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Write(c, http.StatusBadRequest, "invalid rule id", nil)
		return
	}

	var req transport.UpdateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Write(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	input := ruleService.UpdateRuleInput{
		Name:        req.Name,
		Description: req.Description,
		Level:       req.Level,
		Enabled:     req.Enabled,
		Tags:        req.Tags,
	}
	if req.Condition != nil {
		cond := &domain.RuleCondition{
			Type:                  req.Condition.Type,
			EventType:             req.Condition.EventType,
			GroupBy:               req.Condition.GroupBy,
			Threshold:             req.Condition.Threshold,
			WindowSec:             req.Condition.WindowSec,
			Severity:              req.Condition.Severity,
			SequenceSteps:         req.Condition.SequenceSteps,
			CorrelationEventTypes: req.Condition.CorrelationEventTypes,
			MinDistinctEventTypes: req.Condition.MinDistinctEventTypes,
			BaselineWindowSec:     req.Condition.BaselineWindowSec,
			SpikeFactor:           req.Condition.SpikeFactor,
			AnomalyMinCount:       req.Condition.AnomalyMinCount,
		}
		input.Condition = cond
	}

	claims := middleware.UserClaimsFromContext(c)

	r, err := h.svc.Update(c.Request.Context(), id, input, claims, c.ClientIP())
	if err != nil {
		if errors.Is(err, ruleService.ErrInvalidRuleCondition) {
			response.Write(c, http.StatusBadRequest, err.Error(), nil)
			return
		}
		h.log.WithError(err).Error("failed to update rule")
		response.Write(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}
	if r == nil {
		response.Write(c, http.StatusNotFound, "rule not found", nil)
		return
	}

	response.Write(c, http.StatusOK, "rule updated", transport.ToRuleResponse(r))
}

// Delete handles DELETE /api/v1/rules/:id.
func (h *RuleHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Write(c, http.StatusBadRequest, "invalid rule id", nil)
		return
	}

	claims := middleware.UserClaimsFromContext(c)

	if err := h.svc.Delete(c.Request.Context(), id, claims, c.ClientIP()); err != nil {
		h.log.WithError(err).Error("failed to delete rule")
		response.Write(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	response.Write(c, http.StatusOK, "rule deleted", nil)
}
