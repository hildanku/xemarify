package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/hildanku/xemarify-agent/internal/agent/apiclient"
	"github.com/hildanku/xemarify-agent/internal/agent/collector"
	"github.com/hildanku/xemarify-agent/internal/agent/config"
	"github.com/hildanku/xemarify-agent/internal/agent/model"
	"github.com/hildanku/xemarify-agent/internal/agent/pipeline"
)

const (
	configPath      = "/etc/xemarify/agent.yaml"
	batchSize       = 100
	batchFlushEvery = 2 * time.Second
	retryBaseDelay  = 1 * time.Second
	retryMaxDelay   = 30 * time.Second
)

func main() {
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("failed to load config %s: %v", configPath, err)
	}

	if cfg.Server.Endpoint == "" {
		log.Fatalf("server.endpoint is required in %s", configPath)
	}
	if cfg.Syslog.Listen == "" {
		cfg.Syslog.Listen = ":5514"
	}

	client := apiclient.NewHTTPClient(cfg.Server.Insecure)
	localHostname, _ := os.Hostname()
	if localHostname == "" {
		localHostname = "unknown-host"
	}

	agentHostname := strings.TrimSpace(cfg.Agent.Hostname)
	if agentHostname == "" {
		agentHostname = localHostname
	}

	agentName := strings.TrimSpace(cfg.Agent.Name)
	if agentName == "" {
		agentName = agentHostname
	}

	if strings.TrimSpace(cfg.Agent.ID) == "" {
		if strings.TrimSpace(cfg.EnrollmentToken) == "" {
			log.Fatalf("enrollment_token is required for first registration in %s", configPath)
		}

		resp, err := apiclient.Register(client, cfg.Server.Endpoint, cfg.EnrollmentToken, model.RegisterRequest{
			Name:     agentName,
			Hostname: agentHostname,
			IP:       strings.TrimSpace(cfg.Agent.IPAddress),
			OS:       runtime.GOOS,
			Version:  model.AgentVersion,
		})
		if err != nil {
			log.Fatalf("registration failed: %v", err)
		}

		cfg.Agent.ID = resp.AgentID
		cfg.Agent.AgentSecret = resp.AgentSecret
		cfg.EnrollmentToken = ""

		if err := config.Save(configPath, cfg); err != nil {
			log.Fatalf("failed to persist registration state: %v", err)
		}
		log.Printf("agent registered successfully: agent_id=%s", cfg.Agent.ID)
	}

	if strings.TrimSpace(cfg.Agent.AgentSecret) == "" {
		log.Fatalf("agent.agent_secret is required after registration")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var eventsDelivered atomic.Int64
	startedAt := time.Now()
	eventCh := make(chan model.IngestEvent, 2048)
	queue, err := pipeline.NewDiskBackedQueue(cfg.DiskBuffer.Path, cfg.DiskBuffer.MaxBytes)
	if err != nil {
		log.Fatalf("failed to initialize disk buffer queue: %v", err)
	}
	retryPolicy := pipeline.ExponentialBackoffPolicy{
		BaseDelay: retryBaseDelay,
		MaxDelay:  retryMaxDelay,
	}

	go pipeline.RunIngestor(ctx, eventCh, queue)
	go pipeline.RunSender(ctx, client, cfg.Server.Endpoint, cfg.Agent.ID, cfg.Agent.AgentSecret, queue, retryPolicy, batchSize, batchFlushEvery, &eventsDelivered)
	go pipeline.RunHeartbeat(ctx, client, cfg.Server.Endpoint, cfg.Agent.ID, cfg.Agent.AgentSecret, 60*time.Second, &eventsDelivered, startedAt)
	go collector.RunSyslogUDP(ctx, cfg.Syslog.Listen, agentHostname, eventCh)

	if cfg.FileLog.Enabled && len(cfg.FileLog.Paths) > 0 {
		go collector.RunFileLog(ctx, cfg.FileLog.Paths, agentHostname, eventCh, cfg.FileLog.PollInterval)
	}

	if cfg.Inventory.Enabled {
		go collector.RunInventory(ctx, agentHostname, eventCh, cfg.Inventory.Interval)
	}

	log.Printf(
		"xemarify-agent started: endpoint=%s syslog_listen=%s filelog_enabled=%t filelog_paths=%d inventory_enabled=%t inventory_interval=%s disk_buffer_path=%s disk_buffer_max_bytes=%d",
		cfg.Server.Endpoint,
		cfg.Syslog.Listen,
		cfg.FileLog.Enabled,
		len(cfg.FileLog.Paths),
		cfg.Inventory.Enabled,
		cfg.Inventory.Interval,
		cfg.DiskBuffer.Path,
		cfg.DiskBuffer.MaxBytes,
	)
	<-ctx.Done()
	log.Println("xemarify-agent shutting down")
}
