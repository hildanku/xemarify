package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/hildanku/xemarify/config"
	"github.com/hildanku/xemarify/internal/engine"
	infraLogger "github.com/hildanku/xemarify/internal/infrastructure/logger"
	"github.com/hildanku/xemarify/internal/infrastructure/metrics"
	"github.com/hildanku/xemarify/internal/infrastructure/middleware"
	agentHandler "github.com/hildanku/xemarify/internal/modules/agent/handler"
	agentRepo "github.com/hildanku/xemarify/internal/modules/agent/repository"
	agentService "github.com/hildanku/xemarify/internal/modules/agent/service"
	auditHandler "github.com/hildanku/xemarify/internal/modules/audit/handler"
	auditRepo "github.com/hildanku/xemarify/internal/modules/audit/repository"
	auditService "github.com/hildanku/xemarify/internal/modules/audit/service"
	authHandler "github.com/hildanku/xemarify/internal/modules/auth/handler"
	authRepo "github.com/hildanku/xemarify/internal/modules/auth/repository"
	authService "github.com/hildanku/xemarify/internal/modules/auth/service"
	eventHandler "github.com/hildanku/xemarify/internal/modules/event/handler"
	eventRepo "github.com/hildanku/xemarify/internal/modules/event/repository"
	eventService "github.com/hildanku/xemarify/internal/modules/event/service"
	ruleHandler "github.com/hildanku/xemarify/internal/modules/rule/handler"
	ruleRepo "github.com/hildanku/xemarify/internal/modules/rule/repository"
	ruleService "github.com/hildanku/xemarify/internal/modules/rule/service"
	userDomain "github.com/hildanku/xemarify/internal/modules/user/domain"
	userHandler "github.com/hildanku/xemarify/internal/modules/user/handler"
	userRepo "github.com/hildanku/xemarify/internal/modules/user/repository"
	userService "github.com/hildanku/xemarify/internal/modules/user/service"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Logger
	log := infraLogger.New(cfg.LogLevel)
	log.Info("starting xemarify manager")

	// Database
	db, err := config.NewDatabasePool(cfg.Database, log)
	if err != nil {
		log.WithError(err).Fatal("failed to connect to database")
	}
	defer db.Close()

	// Metrics
	m := metrics.New()

	// Repositories
	agentRepository := agentRepo.NewPgAgentRepository(db)
	eventRepository := eventRepo.NewPgEventRepository(db)
	userRepository := userRepo.NewPgUserRepository(db)
	authRepository := authRepo.NewPgAuthRepository(db)
	auditLogRepository := auditRepo.NewPgAuditLogRepository(db)
	ruleRepository := ruleRepo.NewPgRuleRepository(db)

	// Services
	agentSvc := agentService.NewAgentService(agentRepository, log)
	ruleEngine, err := engine.NewRuleEngine(context.Background(), db, log)
	if err != nil {
		log.WithError(err).Fatal("failed to initialize rule engine")
	}
	defer ruleEngine.Stop()

	evtService := eventService.NewEventService(eventRepository, agentRepository, ruleEngine, m, log)
	auditLogService := auditService.NewAuditLogService(auditLogRepository, log)
	authSvc := authService.NewAuthService(userRepository, authRepository, auditLogService, cfg.JWT, log)
	userSvc := userService.NewUserService(db, userRepository, auditLogService, log)
	ruleSvc := ruleService.NewRuleService(ruleRepository, log)

	// HTTP router
	if cfg.LogLevel != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())

	// CORS: allow everything in development (LOG_LEVEL=debug), restrict in production
	if cfg.LogLevel == "debug" {
		router.Use(cors.New(cors.Config{
			AllowAllOrigins:  true,
			AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"*"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: false,
		}))
	} else {
		router.Use(cors.Default())
	}

	// Public endpoints
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Auth routes (public)
	authGroup := router.Group("/auth")
	authHandle := authHandler.NewAuthHandler(authSvc, log)
	authHandle.Register(authGroup)

	// Auth logout requires JWT
	authProtected := router.Group("/auth")
	authProtected.Use(middleware.UserAuth(cfg.JWT))
	authHandle.RegisterProtected(authProtected)

	// Agent API v1 (agent key + rate-limit)
	rateCfg := middleware.DefaultRateLimiterConfig()
	apiV1 := router.Group("/api/v1")
	apiV1.Use(middleware.AgentAuth(agentRepository, log))
	apiV1.Use(middleware.AgentRateLimit(rateCfg, log))

	evtHandler := eventHandler.NewEventHandler(evtService, m, log)
	evtHandler.Register(apiV1)

	// Manaager API v1 (jwt+rbac)
	managerV1 := router.Group("/api/v1")
	managerV1.Use(middleware.UserAuth(cfg.JWT))

	// Users - Manager only
	usersGroup := managerV1.Group("/users")
	usersGroup.Use(middleware.RequireRole(userDomain.RoleManager))
	userHandle := userHandler.NewUserHandler(userSvc, log)
	userHandle.Register(usersGroup)

	// Agents - Manager only
	agentsGroup := managerV1.Group("/agents")
	agentsGroup.Use(middleware.RequireRole(userDomain.RoleManager))
	agentHandle := agentHandler.NewAgentHandler(agentSvc, log)
	agentHandle.Register(agentsGroup)

	// Audit Logs - Manager & Analyst
	auditGroup := managerV1.Group("/audit-logs")
	auditGroup.Use(middleware.RequireRole(userDomain.RoleManager, userDomain.RoleAnalyst))
	auditHandle := auditHandler.NewAuditLogHandler(auditLogService, log)
	auditHandle.Register(auditGroup)

	// Events read (list) - Manager & Analyst
	eventsGroup := managerV1.Group("/events")
	eventsGroup.Use(middleware.RequireRole(userDomain.RoleManager, userDomain.RoleAnalyst))
	evtHandler.RegisterManager(eventsGroup)

	// Detection Rules - Manager only
	rulesGroup := managerV1.Group("/rules")
	rulesGroup.Use(middleware.RequireRole(userDomain.RoleManager))
	ruleHandle := ruleHandler.NewRuleHandler(ruleSvc, log)
	ruleHandle.Register(rulesGroup)

	// Http Server with graceful shutdown
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.WithField("addr", addr).Info("http server listening")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.WithError(err).Fatal("http server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.WithError(err).Fatal("server forced to shutdown")
	}

	log.Info("server exited cleanly")
}
