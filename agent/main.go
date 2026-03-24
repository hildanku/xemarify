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
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown-host"
	}

	if strings.TrimSpace(cfg.Agent.ID) == "" {
		if strings.TrimSpace(cfg.Agent.AgentKey) == "" {
			log.Fatalf("agent.agent_key is required for first registration in %s", configPath)
		}

		resp, err := apiclient.Register(client, cfg.Server.Endpoint, cfg.Agent.AgentKey, model.RegisterRequest{
			Name:     hostname,
			Hostname: hostname,
			IP:       "",
			OS:       runtime.GOOS,
			Version:  model.AgentVersion,
		})
		if err != nil {
			log.Fatalf("registration failed: %v", err)
		}

		cfg.Agent.ID = resp.AgentID
		cfg.Agent.Key = resp.Key
		cfg.Agent.AgentKey = ""

		if err := config.Save(configPath, cfg); err != nil {
			log.Fatalf("failed to persist registration state: %v", err)
		}
		log.Printf("agent registered successfully: agent_id=%s", cfg.Agent.ID)
	}

	if strings.TrimSpace(cfg.Agent.Key) == "" {
		log.Fatalf("agent.key is required after registration")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var eventsDelivered atomic.Int64
	startedAt := time.Now()
	eventCh := make(chan model.IngestEvent, 2048)
	queue := pipeline.NewMemoryQueue()
	retryPolicy := pipeline.ExponentialBackoffPolicy{
		BaseDelay: retryBaseDelay,
		MaxDelay:  retryMaxDelay,
	}

	go pipeline.RunIngestor(ctx, eventCh, queue)
	go pipeline.RunSender(ctx, client, cfg.Server.Endpoint, cfg.Agent.ID, cfg.Agent.Key, queue, retryPolicy, batchSize, batchFlushEvery, &eventsDelivered)
	go pipeline.RunHeartbeat(ctx, client, cfg.Server.Endpoint, cfg.Agent.ID, cfg.Agent.Key, 60*time.Second, &eventsDelivered, startedAt)
	go collector.RunSyslogUDP(ctx, cfg.Syslog.Listen, hostname, eventCh)

	log.Printf("xemarify-agent started: endpoint=%s syslog_listen=%s", cfg.Server.Endpoint, cfg.Syslog.Listen)
	<-ctx.Done()
	log.Println("xemarify-agent shutting down")
}
