package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hildanku/xemarify/internal/modules/agent/domain"
	agentRepo "github.com/hildanku/xemarify/internal/modules/agent/repository"
	"github.com/hildanku/xemarify/pkg/response"
	"github.com/sirupsen/logrus"
)

const (
	// AgentContextKey is the key used to store the authenticated agent in the Gin context.
	AgentContextKey = "authenticated_agent"

	agentKeyHeader = "X-Agent-Key"
)

// AgentAuth returns a Gin middleware that validates the X-Agent-Key header
// against the agents table and rejects unknown or OFFLINE agents.
//
// On success the *domain.Agent is stored in the context under AgentContextKey
// so downstream handlers don't need to re-fetch it.
func AgentAuth(repo agentRepo.AgentRepository, log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader(agentKeyHeader)
		if key == "" {
			log.WithField("remote_addr", c.ClientIP()).Warn("missing agent key header")
			response.WriteWithAbort(c, http.StatusUnauthorized, "missing X-Agent-Key header", nil)
			return
		}

		agent, err := repo.GetByKey(c.Request.Context(), key)
		if err != nil {
			log.WithError(err).Error("agent lookup failed")
			response.WriteWithAbort(c, http.StatusInternalServerError, "internal server error", nil)
			return
		}
		if agent == nil {
			log.WithField("remote_addr", c.ClientIP()).Warn("invalid agent key")
			response.WriteWithAbort(c, http.StatusUnauthorized, "invalid agent key", nil)
			return
		}
		if agent.Status == domain.AgentStatusOffline {
			log.WithFields(logrus.Fields{
				"agent_id":   agent.ID,
				"agent_name": agent.Name,
			}).Warn("rejected event from OFFLINE agent")
			response.WriteWithAbort(c, http.StatusForbidden, "agent is OFFLINE", nil)
			return
		}

		c.Set(AgentContextKey, agent)
		c.Next()
	}
}

// AgentFromContext retrieves the authenticated agent stored by AgentAuth middleware.
// Returns nil if the middleware was not applied or authentication failed.
func AgentFromContext(c *gin.Context) *domain.Agent {
	val, exists := c.Get(AgentContextKey)
	if !exists {
		return nil
	}
	agent, _ := val.(*domain.Agent)
	return agent
}
