package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type pgStatsRepository struct {
	db *pgxpool.Pool
}

func NewPgStatsRepository(db *pgxpool.Pool) StatsRepository {
	return &pgStatsRepository{db: db}
}

func (r *pgStatsRepository) GetDashboardStats(ctx context.Context) (*DashboardStats, error) {
	// Run all aggregate queries in a single round-trip using a CTE.
	// Events uses a time-bounded COUNT to avoid full-partition scans —
	// we count events received in the last 30 days which aligns with the
	// default partition pruning window already used by the List query.
	const q = `
		SELECT
			(SELECT COUNT(*) FROM events
			 WHERE received_at >= NOW() - INTERVAL '30 days')        AS total_events,
			(SELECT COUNT(*) FROM alerts)                             AS total_alerts,
			(SELECT COUNT(*) FROM alerts WHERE status = 'new')        AS new_alerts,
			(SELECT COUNT(*) FROM agents)                             AS total_agents,
			(SELECT COUNT(*) FROM agents WHERE status = 'ONLINE')     AS online_agents,
			(SELECT COUNT(*) FROM audit_logs)                         AS audit_log_total
	`

	var s DashboardStats
	err := r.db.QueryRow(ctx, q).Scan(
		&s.TotalEvents,
		&s.TotalAlerts,
		&s.NewAlerts,
		&s.TotalAgents,
		&s.OnlineAgents,
		&s.AuditLogTotal,
	)
	if err != nil {
		return nil, fmt.Errorf("get dashboard stats: %w", err)
	}
	return &s, nil
}

func (r *pgStatsRepository) GetActivityTrend(ctx context.Context, days int) ([]TrendPoint, error) {
	// Generate a series of days then LEFT JOIN counts so we always get a row
	// for every day even when there is no activity.
	const q = `
		WITH day_series AS (
			SELECT generate_series(
				(NOW() - ($1::int - 1) * INTERVAL '1 day')::date,
				NOW()::date,
				INTERVAL '1 day'
			)::date AS day
		),
		event_counts AS (
			SELECT received_at::date AS day, COUNT(*) AS cnt
			FROM events
			WHERE received_at >= (NOW() - ($1::int - 1) * INTERVAL '1 day')::date
			GROUP BY received_at::date
		),
		alert_counts AS (
			SELECT triggered_at::date AS day, COUNT(*) AS cnt
			FROM alerts
			WHERE triggered_at >= (NOW() - ($1::int - 1) * INTERVAL '1 day')::date
			GROUP BY triggered_at::date
		)
		SELECT
			to_char(d.day, 'YYYY-MM-DD')        AS day,
			COALESCE(ec.cnt, 0)                 AS events,
			COALESCE(ac.cnt, 0)                 AS alerts
		FROM day_series d
		LEFT JOIN event_counts ec ON ec.day = d.day
		LEFT JOIN alert_counts ac ON ac.day = d.day
		ORDER BY d.day ASC
	`

	rows, err := r.db.Query(ctx, q, days)
	if err != nil {
		return nil, fmt.Errorf("get activity trend: %w", err)
	}
	defer rows.Close()

	points := make([]TrendPoint, 0, days)
	for rows.Next() {
		var p TrendPoint
		if err := rows.Scan(&p.Day, &p.Events, &p.Alerts); err != nil {
			return nil, fmt.Errorf("scan trend point: %w", err)
		}
		points = append(points, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("get activity trend rows: %w", err)
	}
	return points, nil
}

func (r *pgStatsRepository) GetAlertStatusDistribution(ctx context.Context) ([]AlertStatusCount, error) {
	const q = `
		SELECT status, COUNT(*) AS count
		FROM alerts
		GROUP BY status
		ORDER BY status ASC
	`

	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("get alert status distribution: %w", err)
	}
	defer rows.Close()

	counts := make([]AlertStatusCount, 0, 4)
	for rows.Next() {
		var c AlertStatusCount
		if err := rows.Scan(&c.Status, &c.Count); err != nil {
			return nil, fmt.Errorf("scan alert status count: %w", err)
		}
		counts = append(counts, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("get alert status distribution rows: %w", err)
	}
	return counts, nil
}
