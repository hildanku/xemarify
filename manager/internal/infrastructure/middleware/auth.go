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

	agentSecretHeader = "X-Agent-Secret"
)

// AgentAuth returns a Gin middleware that validates the X-Agent-Secret header
// against the agents table and rejects unknown or OFFLINE agents.
//
// Validated agents are kept in a short-lived in-memory cache keyed by secret
// to avoid hitting the database on every request. Cache entries expire after
// agentAuthCacheTTL, so a status change in the database is reflected within
// that window at worst.
//
// On success the *domain.Agent is stored in the context under AgentContextKey
// so downstream handlers don't need to re-fetch it.
func AgentAuth(repo agentRepo.AgentRepository, log *logrus.Logger) gin.HandlerFunc {
	cache := currentAgentAuthCache()

	return func(c *gin.Context) {
		secret := c.GetHeader(agentSecretHeader)
		if secret == "" {
			log.WithField("remote_addr", c.ClientIP()).Warn("missing agent secret header")
			response.WriteWithAbort(c, http.StatusUnauthorized, "missing X-Agent-Secret header", nil)
			return
		}

		if cached, hit := cache.get(secret); hit {
			if cached.Status == domain.AgentStatusOffline {
				log.WithFields(logrus.Fields{
					"agent_id":   cached.ID,
					"agent_name": cached.Name,
				}).Debug("rejected OFFLINE agent from cache")
				response.WriteWithAbort(c, http.StatusForbidden, "agent is OFFLINE", nil)
				return
			}
			log.WithFields(logrus.Fields{
				"agent_id":   cached.ID,
				"agent_name": cached.Name,
			}).Debug("agent auth cache hit")
			c.Set(AgentContextKey, cached)
			c.Next()
			return
		}

		agent, err := repo.GetBySecret(c.Request.Context(), secret)
		if err != nil {
			log.WithError(err).Error("agent lookup failed")
			response.WriteWithAbort(c, http.StatusInternalServerError, "internal server error", nil)
			return
		}
		if agent == nil {
			log.WithField("remote_addr", c.ClientIP()).Warn("invalid agent secret")
			response.WriteWithAbort(c, http.StatusUnauthorized, "invalid agent secret", nil)
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

		cache.set(secret, agent)
		log.WithFields(logrus.Fields{
			"agent_id":   agent.ID,
			"agent_name": agent.Name,
		}).Debug("agent auth cache miss; populated from DB")

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
