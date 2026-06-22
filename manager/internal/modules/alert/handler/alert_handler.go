package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/infrastructure/middleware"
	alertRepo "github.com/hildanku/xemarify/internal/modules/alert/repository"
	alertService "github.com/hildanku/xemarify/internal/modules/alert/service"
	"github.com/hildanku/xemarify/internal/modules/alert/transport"
	"github.com/hildanku/xemarify/pkg/query"
	"github.com/hildanku/xemarify/pkg/response"
	"github.com/sirupsen/logrus"
)

type AlertHandler struct {
	svc *alertService.AlertService
	log *logrus.Logger
}

func NewAlertHandler(svc *alertService.AlertService, log *logrus.Logger) *AlertHandler {
	return &AlertHandler{svc: svc, log: log}
}

func (h *AlertHandler) Register(rg *gin.RouterGroup) {
	rg.GET("", h.List)
	rg.GET("/:id", h.GetByID)
	rg.PATCH("/:id/status", h.UpdateStatus)
}

func (h *AlertHandler) List(c *gin.Context) {
	var q transport.ListAlertsQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Write(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	var ruleID *uuid.UUID
	if strings.TrimSpace(q.RuleID) != "" {
		parsed, err := uuid.Parse(q.RuleID)
		if err != nil {
			response.Write(c, http.StatusBadRequest, "invalid rule_id", nil)
			return
		}
		ruleID = &parsed
	}

	triggeredFrom, err := parseTime(q.TriggeredFrom)
	if err != nil {
		response.Write(c, http.StatusBadRequest, "invalid triggered_from, expected RFC3339", nil)
		return
	}
	triggeredTo, err := parseTime(q.TriggeredTo)
	if err != nil {
		response.Write(c, http.StatusBadRequest, "invalid triggered_to, expected RFC3339", nil)
		return
	}

	filter := alertRepo.ListFilter{
		BaseFilter: query.BaseFilter{
			Search: q.Search,
			Order:  query.SortOrder(q.Order),
			Limit:  q.Limit,
		},
		Severity:      strings.TrimSpace(strings.ToUpper(q.Severity)),
		Status:        strings.TrimSpace(strings.ToLower(q.Status)),
		RuleID:        ruleID,
		TriggeredFrom: triggeredFrom,
		TriggeredTo:   triggeredTo,
		Cursor:        q.Cursor,
	}

	alerts, nextCursor, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		if errors.Is(err, alertService.ErrInvalidAlertStatus) {
			response.Write(c, http.StatusBadRequest, err.Error(), nil)
			return
		}
		h.log.WithError(err).Error("failed to list alerts")
		response.Write(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	items := make([]*transport.AlertResponse, 0, len(alerts))
	for _, alert := range alerts {
		items = append(items, transport.ToAlertResponse(alert))
	}

	response.Write(c, http.StatusOK, "alerts retrieved", transport.ListAlertsResponse{
		Items: items,
		Metadata: transport.ListAlertsMetadata{
			NextCursor: nextCursor,
			HasMore:    nextCursor != "",
			Limit:      filter.Limit,
		},
	})
}

func (h *AlertHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Write(c, http.StatusBadRequest, "invalid alert id", nil)
		return
	}

	detail, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, alertService.ErrAlertNotFound) {
			response.Write(c, http.StatusNotFound, "alert not found", nil)
			return
		}
		h.log.WithError(err).Error("failed to get alert")
		response.Write(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	events := make([]*transport.AlertEventResponse, 0, len(detail.Events))
	for _, event := range detail.Events {
		events = append(events, transport.ToAlertEventResponse(event))
	}

	response.Write(c, http.StatusOK, "alert retrieved", transport.AlertDetailResponse{
		Alert:       transport.ToAlertResponse(detail.Alert),
		Events:      events,
		Explanation: transport.ToAlertExplanationResponse(detail.Explanation),
	})
}

func (h *AlertHandler) UpdateStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Write(c, http.StatusBadRequest, "invalid alert id", nil)
		return
	}

	var req transport.UpdateAlertStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Write(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	claims := middleware.UserClaimsFromContext(c)

	if err := h.svc.UpdateStatus(c.Request.Context(), id, req.Status, claims, c.ClientIP()); err != nil {
		if errors.Is(err, alertService.ErrAlertNotFound) {
			response.Write(c, http.StatusNotFound, "alert not found", nil)
			return
		}
		if errors.Is(err, alertService.ErrInvalidAlertStatus) {
			response.Write(c, http.StatusBadRequest, err.Error(), nil)
			return
		}
		h.log.WithError(err).Error("failed to update alert status")
		response.Write(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	response.Write(c, http.StatusOK, "alert status updated", nil)
}

func parseTime(value string) (*time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
