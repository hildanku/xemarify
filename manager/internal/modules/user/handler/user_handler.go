package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/infrastructure/middleware"
	userRepo "github.com/hildanku/xemarify/internal/modules/user/repository"
	"github.com/hildanku/xemarify/internal/modules/user/service"
	"github.com/hildanku/xemarify/internal/modules/user/transport"
	"github.com/hildanku/xemarify/pkg/query"
	"github.com/hildanku/xemarify/pkg/response"
	"github.com/sirupsen/logrus"
)

// UserHandler handles HTTP requests for the user management endpoints.
type UserHandler struct {
	svc *service.UserService
	log *logrus.Logger
}

// NewUserHandler constructs a UserHandler.
func NewUserHandler(svc *service.UserService, log *logrus.Logger) *UserHandler {
	return &UserHandler{svc: svc, log: log}
}

// Register wires the user routes onto the given router group.
// The group must already have JWT + RBAC(MANAGER) middleware applied.
func (h *UserHandler) Register(rg *gin.RouterGroup) {
	rg.GET("", h.List)
	rg.POST("", h.Create)
	rg.GET("/:id", h.GetByID)
	rg.PUT("/:id", h.Update)
	rg.DELETE("/:id", h.Delete)
}

// List handles GET /api/v1/users.
//
// Query params:
//
//	search   - case-insensitive partial match on username and email
//	sort_by  - field to sort by (username|email|role|created_at); default: created_at
//	order    - sort direction (asc|desc); default: asc
//	limit    - max rows (1-100); default: 10
//	offset   - rows to skip; default: 0
func (h *UserHandler) List(c *gin.Context) {
	var q transport.ListUsersQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Write(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	filter := userRepo.ListFilter{
		BaseFilter: query.BaseFilter{
			Search: q.Search,
			SortBy: q.SortBy,
			Order:  query.SortOrder(q.Order),
			Limit:  q.Limit,
			Offset: q.Offset,
		},
	}

	users, total, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		h.log.WithError(err).Error("failed to list users")
		response.Write(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	items := make([]*transport.UserResponse, 0, len(users))
	for _, u := range users {
		items = append(items, transport.ToUserResponse(u))
	}

	totalPages := 0
	if filter.Limit > 0 {
		totalPages = (total + filter.Limit - 1) / filter.Limit
	}

	response.Write(c, http.StatusOK, "users retrieved", transport.ListUsersResponse{
		Items: items,
		Metadata: transport.ListUsersMetadata{
			Total:      total,
			TotalPages: totalPages,
			Limit:      filter.Limit,
			Offset:     filter.Offset,
		},
	})
}

// Create handles POST /api/v1/users.
func (h *UserHandler) Create(c *gin.Context) {
	var req transport.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Write(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	claims := middleware.UserClaimsFromContext(c)

	u, err := h.svc.Create(c.Request.Context(), service.CreateUserInput{
		Username: req.Username,
		Email:    req.Email,
		Role:     req.Role,
		Password: req.Password,
		Avatar:   req.Avatar,
	}, claims, c.ClientIP())
	if err != nil {
		h.log.WithError(err).Error("failed to create user")
		response.Write(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	response.Write(c, http.StatusCreated, "user created", transport.ToUserResponse(u))
}

// GetByID handles GET /api/v1/users/:id.
func (h *UserHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Write(c, http.StatusBadRequest, "invalid user id", nil)
		return
	}

	u, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			response.Write(c, http.StatusNotFound, "user not found", nil)
			return
		}
		h.log.WithError(err).Error("failed to get user")
		response.Write(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	response.Write(c, http.StatusOK, "user retrieved", transport.ToUserResponse(u))
}

// Update handles PUT /api/v1/users/:id.
func (h *UserHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Write(c, http.StatusBadRequest, "invalid user id", nil)
		return
	}

	var req transport.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Write(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	claims := middleware.UserClaimsFromContext(c)

	u, err := h.svc.Update(c.Request.Context(), id, service.UpdateUserInput{
		Username: req.Username,
		Email:    req.Email,
		Role:     req.Role,
		Avatar:   req.Avatar,
	}, claims, c.ClientIP())
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			response.Write(c, http.StatusNotFound, "user not found", nil)
			return
		}
		h.log.WithError(err).Error("failed to update user")
		response.Write(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	response.Write(c, http.StatusOK, "user updated", transport.ToUserResponse(u))
}

// Delete handles DELETE /api/v1/users/:id.
func (h *UserHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Write(c, http.StatusBadRequest, "invalid user id", nil)
		return
	}

	claims := middleware.UserClaimsFromContext(c)

	if err := h.svc.Delete(c.Request.Context(), id, claims, c.ClientIP()); err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			response.Write(c, http.StatusNotFound, "user not found", nil)
			return
		}
		h.log.WithError(err).Error("failed to delete user")
		response.Write(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	response.Write(c, http.StatusOK, "user deleted", nil)
}
