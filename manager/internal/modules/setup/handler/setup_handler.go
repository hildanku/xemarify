package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	setupService "github.com/hildanku/xemarify/internal/modules/setup/service"
	"github.com/hildanku/xemarify/internal/modules/setup/transport"
	"github.com/hildanku/xemarify/pkg/response"
	"github.com/sirupsen/logrus"
)

// SetupHandler handles first-run setup requests.
type SetupHandler struct {
	svc *setupService.SetupService
	log *logrus.Logger
}

func NewSetupHandler(svc *setupService.SetupService, log *logrus.Logger) *SetupHandler {
	return &SetupHandler{svc: svc, log: log}
}

// Register wires public setup endpoints.
func (h *SetupHandler) Register(rg *gin.RouterGroup) {
	rg.POST("/initialize", h.Initialize)
}

// Initialize handles POST /setup/initialize.
func (h *SetupHandler) Initialize(c *gin.Context) {
	var req transport.InitializeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Write(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	result, err := h.svc.InitializeFirstManager(c.Request.Context(), req.Username, req.Email, req.Password, req.SetupToken)
	if err != nil {
		switch {
		case errors.Is(err, setupService.ErrInvalidSetupToken):
			response.Write(c, http.StatusUnauthorized, "invalid setup token", nil)
			return
		case errors.Is(err, setupService.ErrSetupTokenUnavailable):
			response.Write(c, http.StatusFailedDependency, "manager setup token is not configured", nil)
			return
		case errors.Is(err, setupService.ErrAlreadyInitialized):
			response.Write(c, http.StatusConflict, "system already initialized", nil)
			return
		}

		h.log.WithError(err).Error("failed to initialize first manager")
		response.Write(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	response.Write(c, http.StatusCreated, "initial manager created", gin.H{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
	})
}
