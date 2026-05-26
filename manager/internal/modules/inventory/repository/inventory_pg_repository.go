package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/hildanku/xemarify/internal/modules/inventory/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgInventoryRepository struct {
	db *pgxpool.Pool
}

// NewPgInventoryRepository creates a Postgres-backed InventoryRepository.
func NewPgInventoryRepository(db *pgxpool.Pool) InventoryRepository {
	return &pgInventoryRepository{db: db}
}

func (r *pgInventoryRepository) Upsert(ctx context.Context, inv *domain.Inventory) error {
	const q = `
		INSERT INTO agent_inventory (
			agent_id, os, arch, kernel_version, cpu_model, cpu_cores,
			memory_total_mb, uptime_seconds, ip_addresses,
			nginx_installed, apache_installed, collected_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9,
			$10, $11, $12, NOW()
		)
		ON CONFLICT (agent_id) DO UPDATE SET
			os               = EXCLUDED.os,
			arch             = EXCLUDED.arch,
			kernel_version   = EXCLUDED.kernel_version,
			cpu_model        = EXCLUDED.cpu_model,
			cpu_cores        = EXCLUDED.cpu_cores,
			memory_total_mb  = EXCLUDED.memory_total_mb,
			uptime_seconds   = EXCLUDED.uptime_seconds,
			ip_addresses     = EXCLUDED.ip_addresses,
			nginx_installed  = EXCLUDED.nginx_installed,
			apache_installed = EXCLUDED.apache_installed,
			collected_at     = EXCLUDED.collected_at,
			updated_at       = NOW()
	`

	_, err := r.db.Exec(ctx, q,
		inv.AgentID,
		inv.OS,
		inv.Arch,
		inv.KernelVersion,
		inv.CPUModel,
		inv.CPUCores,
		inv.MemoryTotalMB,
		inv.UptimeSeconds,
		inv.IPAddresses,
		inv.NginxInstalled,
		inv.ApacheInstalled,
		inv.CollectedAt,
	)
	return err
}

func (r *pgInventoryRepository) GetByAgentID(ctx context.Context, agentID uuid.UUID) (*domain.Inventory, error) {
	const q = `
		SELECT
			agent_id, os, arch, kernel_version, cpu_model, cpu_cores,
			memory_total_mb, uptime_seconds, ip_addresses,
			nginx_installed, apache_installed, collected_at, updated_at
		FROM agent_inventory
		WHERE agent_id = $1
		LIMIT 1
	`

	row := r.db.QueryRow(ctx, q, agentID)

	var inv domain.Inventory
	var collectedAt *time.Time
	var updatedAt time.Time

	err := row.Scan(
		&inv.AgentID,
		&inv.OS,
		&inv.Arch,
		&inv.KernelVersion,
		&inv.CPUModel,
		&inv.CPUCores,
		&inv.MemoryTotalMB,
		&inv.UptimeSeconds,
		&inv.IPAddresses,
		&inv.NginxInstalled,
		&inv.ApacheInstalled,
		&collectedAt,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	inv.CollectedAt = collectedAt
	inv.UpdatedAt = updatedAt

	return &inv, nil
}
