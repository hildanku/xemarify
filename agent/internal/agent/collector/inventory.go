package collector

import (
	"context"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/hildanku/xemarify-agent/internal/agent/model"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	gnet "github.com/shirou/gopsutil/v4/net"
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
	sourceName := "inventory:host"
	normalized := map[string]interface{}{
		"event_type":          "inventory",
		"source_type":         "inventory",
		"source_name":         sourceName,
		"hostname":            hostname,
		"os":                  runtime.GOOS,
		"arch":                runtime.GOARCH,
		"kernel_version":      kernelVersion(),
		"ip_addresses":        listIPAddresses(),
		"cpu_model":           cpuModel(),
		"cpu_cores":           cpuCores(),
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
		SourceName: sourceName,
		Facility:   "inventory",
		Severity:   "INFO",
		Category:   "inventory",
		Message:    "inventory snapshot",
		Raw:        raw,
		Attributes: normalized,
		Normalized: normalized,
	}
}

func inventorySummary(fields map[string]interface{}) string {
	return "inventory snapshot"
}

func listIPAddresses() []string {
	interfaces, err := gnet.Interfaces()
	if err != nil {
		return nil
	}

	ips := make([]string, 0, 8)
	for _, iface := range interfaces {
		if !hasNetFlag(iface.Flags, "up") {
			continue
		}
		if hasNetFlag(iface.Flags, "loopback") {
			continue
		}

		for _, addr := range iface.Addrs {
			ipStr := hostFromCIDR(addr.Addr)
			if ipStr == "" {
				continue
			}
			ips = append(ips, ipStr)
		}
	}

	return dedupStrings(ips)
}

func hasNetFlag(flags []string, want string) bool {
	want = strings.ToLower(strings.TrimSpace(want))
	if want == "" {
		return false
	}
	for _, flag := range flags {
		if strings.ToLower(strings.TrimSpace(flag)) == want {
			return true
		}
	}
	return false
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
	info, err := host.Info()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(info.KernelVersion)
}

func cpuModel() string {
	infos, err := cpu.Info()
	if err != nil {
		return ""
	}
	if len(infos) == 0 {
		return ""
	}
	return strings.TrimSpace(infos[0].ModelName)
}

func cpuCores() int {
	count, err := cpu.Counts(true)
	if err != nil || count <= 0 {
		return runtime.NumCPU()
	}
	return count
}

func memoryTotalMB() int64 {
	vm, err := mem.VirtualMemory()
	if err != nil {
		return 0
	}
	return int64(vm.Total / (1024 * 1024))
}

func uptimeSeconds() int64 {
	info, err := host.Info()
	if err != nil {
		return 0
	}
	return int64(info.Uptime)
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
