package collector

import (
	"bufio"
	"context"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/hildanku/xemarify-agent/internal/agent/model"
)

func RunInventory(
	ctx context.Context,
	hostname string,
	output chan<- model.IngestEvent,
	interval time.Duration,
) {
	if interval <= 0 {
		interval = 60 * time.Second
	}

	emit := func() {
		event := buildInventoryEvent(hostname)
		select {
		case output <- event:
		case <-ctx.Done():
			return
		default:
			log.Printf("dropping inventory event: forwarder queue is full")
		}
	}

	emit()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			emit()
		}
	}
}

func buildInventoryEvent(hostname string) model.IngestEvent {
	if strings.TrimSpace(hostname) == "" {
		hostname = "unknown-host"
	}

	now := time.Now().UTC()
	normalized := map[string]interface{}{
		"event_type":          "inventory",
		"hostname":            hostname,
		"os":                  runtime.GOOS,
		"arch":                runtime.GOARCH,
		"kernel_version":      kernelVersion(),
		"ip_addresses":        listIPAddresses(),
		"cpu_model":           cpuModel(),
		"cpu_cores":           runtime.NumCPU(),
		"memory_total_mb":     memoryTotalMB(),
		"uptime_seconds":      uptimeSeconds(),
		"nginx_installed":     binaryExists("nginx"),
		"apache_installed":    binaryExists("apache2") || binaryExists("httpd"),
		"inventory_collected": now.Format(time.RFC3339),
	}

	raw := inventorySummary(normalized)

	return model.IngestEvent{
		EventTime:  now,
		Hostname:   hostname,
		SourceIP:   "",
		InputType:  "inventory",
		Facility:   "inventory",
		Severity:   "INFO",
		Category:   "inventory",
		Message:    "inventory snapshot",
		Raw:        raw,
		Normalized: normalized,
	}
}

func inventorySummary(fields map[string]interface{}) string {
	return "inventory snapshot"
}

func listIPAddresses() []string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil
	}

	ips := make([]string, 0, 8)
	for _, iface := range interfaces {
		if (iface.Flags & net.FlagUp) == 0 {
			continue
		}
		if (iface.Flags & net.FlagLoopback) != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipStr := hostFromCIDR(addr.String())
			if ipStr == "" {
				continue
			}
			ips = append(ips, ipStr)
		}
	}

	return dedupStrings(ips)
}

func hostFromCIDR(value string) string {
	if value == "" {
		return ""
	}
	host, _, err := net.ParseCIDR(value)
	if err == nil && host != nil {
		return host.String()
	}
	if strings.Contains(value, "/") {
		parts := strings.SplitN(value, "/", 2)
		return parts[0]
	}
	return value
}

func dedupStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func kernelVersion() string {
	data, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func cpuModel() string {
	f, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, ":") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		key := strings.TrimSpace(strings.ToLower(parts[0]))
		if key != "model name" {
			continue
		}
		return strings.TrimSpace(parts[1])
	}

	return ""
}

func memoryTotalMB() int64 {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "MemTotal:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return 0
		}
		valueKB, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			return 0
		}
		return valueKB / 1024
	}

	return 0
}

func uptimeSeconds() int64 {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return 0
	}
	value, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0
	}
	return int64(value)
}

func binaryExists(name string) bool {
	if strings.TrimSpace(name) == "" {
		return false
	}

	paths := []string{
		"/usr/bin",
		"/usr/sbin",
		"/bin",
		"/sbin",
	}

	for _, dir := range paths {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if err == nil && !info.IsDir() {
			return true
		}
	}

	return false
}
