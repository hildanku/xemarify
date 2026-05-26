package collector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
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

// InventoryPayload is the request body sent to POST /api/v1/agents/inventory.
type InventoryPayload struct {
	AgentID         string    `json:"agent_id"`
	OS              string    `json:"os"`
	Arch            string    `json:"arch"`
	KernelVersion   string    `json:"kernel_version"`
	CPUModel        string    `json:"cpu_model"`
	CPUCores        int       `json:"cpu_cores"`
	MemoryTotalMB   int64     `json:"memory_total_mb"`
	UptimeSeconds   int64     `json:"uptime_seconds"`
	IPAddresses     []string  `json:"ip_addresses"`
	NginxInstalled  bool      `json:"nginx_installed"`
	ApacheInstalled bool      `json:"apache_installed"`
	CollectedAt     time.Time `json:"collected_at"`
}

// RunInventory collects a system snapshot on startup and then on every interval,
// sending it directly to POST /api/v1/agents/inventory — bypassing the event pipeline.
func RunInventory(
	ctx context.Context,
	hostname string,
	client *http.Client,
	endpoint string,
	agentID string,
	agentSecret string,
	interval time.Duration,
) {
	if interval <= 0 {
		interval = 60 * time.Second
	}

	send := func() {
		payload := buildInventoryPayload(hostname, agentID)
		if err := postInventory(ctx, client, endpoint, agentSecret, payload); err != nil {
			log.Printf("inventory send failed: %v", err)
		}
	}

	send()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			send()
		}
	}
}

func postInventory(ctx context.Context, client *http.Client, endpoint, agentSecret string, payload InventoryPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal inventory: %w", err)
	}

	url := strings.TrimRight(endpoint, "/") + "/api/v1/agents/inventory"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(model.AgentSecretHeader, agentSecret)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send inventory: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("inventory endpoint returned status %d", resp.StatusCode)
	}

	return nil
}

func buildInventoryPayload(hostname, agentID string) InventoryPayload {
	if strings.TrimSpace(hostname) == "" {
		hostname = "unknown-host"
	}

	return InventoryPayload{
		AgentID:         agentID,
		OS:              runtime.GOOS,
		Arch:            runtime.GOARCH,
		KernelVersion:   kernelVersion(),
		IPAddresses:     listIPAddresses(),
		CPUModel:        cpuModel(),
		CPUCores:        cpuCores(),
		MemoryTotalMB:   memoryTotalMB(),
		UptimeSeconds:   uptimeSeconds(),
		NginxInstalled:  binaryExists("nginx"),
		ApacheInstalled: binaryExists("apache2") || binaryExists("httpd"),
		CollectedAt:     time.Now().UTC(),
	}
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
