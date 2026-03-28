package collector

import (
	"context"
	"log"
	"net"
	"strings"
	"time"

	"github.com/hildanku/xemarify-agent/internal/agent/model"
)

func RunSyslogUDP(ctx context.Context, listenAddr, hostname string, output chan<- model.IngestEvent) {
	conn, err := net.ListenPacket("udp", listenAddr)
	if err != nil {
		log.Fatalf("failed to start syslog listener: %v", err)
	}
	defer conn.Close()

	buf := make([]byte, 64*1024)

	for {
		_ = conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, remoteAddr, err := conn.ReadFrom(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				select {
				case <-ctx.Done():
					return
				default:
					continue
				}
			}
			log.Printf("syslog read error: %v", err)
			continue
		}

		message := strings.TrimSpace(string(buf[:n]))
		if message == "" {
			continue
		}

		sourceIP := parseSourceIP(remoteAddr)
		sourceName := "udp:" + listenAddr
		attributes := map[string]interface{}{
			"source_ip":   sourceIP,
			"source_name": sourceName,
		}
		event := model.IngestEvent{
			EventTime:  time.Now().UTC(),
			Hostname:   hostname,
			SourceIP:   sourceIP,
			InputType:  "syslog",
			SourceName: sourceName,
			Facility:   "syslog",
			Severity:   "INFO",
			Category:   "syslog",
			Message:    message,
			Raw:        message,
			Attributes: attributes,
			Normalized: map[string]interface{}{
				"event_type":  "syslog",
				"source_ip":   sourceIP,
				"source_name": sourceName,
				"source_type": "syslog",
				"severity":    "INFO",
				"category":    "syslog",
			},
		}

		select {
		case output <- event:
		case <-ctx.Done():
			return
		}
	}
}

func parseSourceIP(addr net.Addr) string {
	if addr == nil {
		return ""
	}

	host, _, err := net.SplitHostPort(addr.String())
	if err == nil {
		return host
	}

	return addr.String()
}
