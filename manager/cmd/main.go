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
	"github.com/hildanku/xemarify/internal/infrastructure/sse"
	agentHandler "github.com/hildanku/xemarify/internal/modules/agent/handler"
	agentRepo "github.com/hildanku/xemarify/internal/modules/agent/repository"
	agentService "github.com/hildanku/xemarify/internal/modules/agent/service"
	inventoryHandler "github.com/hildanku/xemarify/internal/modules/inventory/handler"
	inventoryRepo "github.com/hildanku/xemarify/internal/modules/inventory/repository"
	inventoryService "github.com/hildanku/xemarify/internal/modules/inventory/service"
	alertHandler "github.com/hildanku/xemarify/internal/modules/alert/handler"
	alertRepo "github.com/hildanku/xemarify/internal/modules/alert/repository"
	alertService "github.com/hildanku/xemarify/internal/modules/alert/service"
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
	statsHandler "github.com/hildanku/xemarify/internal/modules/stats/handler"
	statsRepo "github.com/hildanku/xemarify/internal/modules/stats/repository"
	statsService "github.com/hildanku/xemarify/internal/modules/stats/service"
	setupHandler "github.com/hildanku/xemarify/internal/modules/setup/handler"
	setupService "github.com/hildanku/xemarify/internal/modules/setup/service"
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

	// SSE Hub
	eventHub := sse.NewHub()
	defer eventHub.Stop()

	// Repositories
	agentRepository := agentRepo.NewPgAgentRepository(db)
	inventoryRepository := inventoryRepo.NewPgInventoryRepository(db)
	eventRepository := eventRepo.NewPgEventRepository(db)
	userRepository := userRepo.NewPgUserRepository(db)
	authRepository := authRepo.NewPgAuthRepository(db)
	auditLogRepository := auditRepo.NewPgAuditLogRepository(db)
	ruleRepository := ruleRepo.NewPgRuleRepository(db)
	alertRepository := alertRepo.NewPgAlertRepository(db)

	// Services
	auditLogService := auditService.NewAuditLogService(auditLogRepository, log)
	agentSvc := agentService.NewAgentService(agentRepository, auditLogService, log)
	inventorySvc := inventoryService.NewInventoryService(inventoryRepository, log)
	ruleEngine, err := engine.NewRuleEngine(context.Background(), db, log)
	if err != nil {
		log.WithError(err).Fatal("failed to initialize rule engine")
	}
	defer ruleEngine.Stop()

	evtService := eventService.NewEventService(eventRepository, ruleEngine, eventHub, m, log, cfg.Event.WorkerCount, cfg.Event.ChanBuffer)
	evtService.Start()
	defer evtService.Stop()
	authSvc := authService.NewAuthService(userRepository, authRepository, auditLogService, cfg.JWT, log)
	setupSvc := setupService.NewSetupService(db, cfg.JWT, cfg.Setup.Token, log)
	userSvc := userService.NewUserService(db, userRepository, auditLogService, log)
	ruleSvc := ruleService.NewRuleService(ruleRepository, ruleEngine, auditLogService, log)
	alertSvc := alertService.NewAlertService(alertRepository, auditLogService, log)
	agentHandle := agentHandler.NewAgentHandler(agentSvc, log)
	evtHandler := eventHandler.NewEventHandler(evtService, eventHub, m, log)
	inventoryHandle := inventoryHandler.NewInventoryHandler(inventorySvc, log)

	// HTTP router
	if cfg.LogLevel != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())

	// The web dashboard is served from a separate origin in production, so CORS
	// must allow authenticated browser preflight requests as well.
	router.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With", "X-Agent-Secret", "X-Enrollment-Token"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
	}))

	// Public endpoints
	api := router.Group("/api")

	api.GET("/health", func(c *gin.Context) {
		initialized, err := setupSvc.IsInitialized(c.Request.Context())
		if err != nil {
			log.WithError(err).Error("failed to determine setup status")
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok", "initialized": initialized})
	})
	api.GET("/metrics", gin.WrapH(promhttp.Handler()))
	setupGroup := api.Group("/setup")
	setupHandle := setupHandler.NewSetupHandler(setupSvc, log)
	setupHandle.Register(setupGroup)

	// Auth routes (public)
	authGroup := api.Group("/auth")
	authHandle := authHandler.NewAuthHandler(authSvc, log)
	authHandle.Register(authGroup)

	// Auth logout requires JWT
	authProtected := api.Group("/auth")
	authProtected.Use(middleware.UserAuth(cfg.JWT))
	authHandle.RegisterProtected(authProtected)

	// Agent API v1
	rateCfg := middleware.DefaultRateLimiterConfig()
	apiV1 := api.Group("/v1")

	agentPublicGroup := apiV1.Group("/agents")
	agentHandle.RegisterAgentPublic(agentPublicGroup)

	agentSessionGroup := apiV1.Group("/agents")
	agentSessionGroup.Use(middleware.AgentAuth(agentRepository, log))
	agentSessionGroup.Use(middleware.AgentRateLimit(rateCfg, log))
	agentHandle.RegisterAgentSession(agentSessionGroup)
	inventoryHandle.RegisterAgentSession(agentSessionGroup)

	eventIngestGroup := apiV1.Group("")
	eventIngestGroup.Use(middleware.AgentAuth(agentRepository, log))
	eventIngestGroup.Use(middleware.AgentRateLimit(rateCfg, log))
	evtHandler.Register(eventIngestGroup)

	// Manaager API v1 (jwt+rbac)
	managerV1 := api.Group("/v1")
	managerV1.Use(middleware.UserAuth(cfg.JWT))

	// Users - Manager only
	usersGroup := managerV1.Group("/users")
	usersGroup.Use(middleware.RequireRole(userDomain.RoleManager))
	userHandle := userHandler.NewUserHandler(userSvc, log)
	userHandle.Register(usersGroup)

	// Agents read - Manager & Viewer
	agentsReadGroup := managerV1.Group("/agents")
	agentsReadGroup.Use(middleware.RequireRole(userDomain.RoleManager, userDomain.RoleViewer))
	agentHandle.RegisterRead(agentsReadGroup)
	inventoryHandle.Register(agentsReadGroup)

	// Agents write (CRUD) - Manager only
	agentsGroup := managerV1.Group("/agents")
	agentsGroup.Use(middleware.RequireRole(userDomain.RoleManager))
	agentHandle.RegisterWrite(agentsGroup)

	// Admin - Manager only
	adminGroup := managerV1.Group("/admin")
	adminGroup.Use(middleware.RequireRole(userDomain.RoleManager))
	agentHandle.RegisterAdmin(adminGroup)

	// Audit Logs - Manager, Analyst & Viewer
	auditGroup := managerV1.Group("/audit-logs")
	auditGroup.Use(middleware.RequireRole(userDomain.RoleManager, userDomain.RoleAnalyst, userDomain.RoleViewer))
	auditHandle := auditHandler.NewAuditLogHandler(auditLogService, log)
	auditHandle.Register(auditGroup)

	// Events read (list) - Manager, Analyst & Viewer
	eventsGroup := managerV1.Group("/events")
	eventsGroup.Use(middleware.RequireRole(userDomain.RoleManager, userDomain.RoleAnalyst, userDomain.RoleViewer))
	evtHandler.RegisterManager(eventsGroup)

	// Events SSE stream - separate group with query-param auth for EventSource compatibility
	eventsStreamGroup := api.Group("/v1/events")
	eventsStreamGroup.Use(middleware.UserAuthSSE(cfg.JWT))
	eventsStreamGroup.Use(middleware.RequireRole(userDomain.RoleManager, userDomain.RoleAnalyst, userDomain.RoleViewer))
	evtHandler.RegisterStream(eventsStreamGroup)

	// Detection Rules - Manager only
	rulesGroup := managerV1.Group("/rules")
	rulesGroup.Use(middleware.RequireRole(userDomain.RoleManager))
	ruleHandle := ruleHandler.NewRuleHandler(ruleSvc, log)
	ruleHandle.Register(rulesGroup)

	// Alerts read - Manager, Analyst & Viewer
	alertsReadGroup := managerV1.Group("/alerts")
	alertsReadGroup.Use(middleware.RequireRole(userDomain.RoleManager, userDomain.RoleAnalyst, userDomain.RoleViewer))
	alertHandle := alertHandler.NewAlertHandler(alertSvc, log)
	alertHandle.RegisterRead(alertsReadGroup)

	// Alerts write (status update) - Manager & Analyst only
	alertsWriteGroup := managerV1.Group("/alerts")
	alertsWriteGroup.Use(middleware.RequireRole(userDomain.RoleManager, userDomain.RoleAnalyst))
	alertHandle.RegisterWrite(alertsWriteGroup)

	// Stats / Dashboard - Manager, Analyst & Viewer
	statsRepository := statsRepo.NewPgStatsRepository(db)
	statsSvc := statsService.NewStatsService(statsRepository, log)
	statsGroup := managerV1.Group("/stats")
	statsGroup.Use(middleware.RequireRole(userDomain.RoleManager, userDomain.RoleAnalyst, userDomain.RoleViewer))
	statsHandle := statsHandler.NewStatsHandler(statsSvc, log)
	statsHandle.Register(statsGroup)

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
