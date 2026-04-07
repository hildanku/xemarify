package engine

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type runtimeCheckpoint struct {
	LastEventID   uuid.UUID
	LastEventTime time.Time
}

type persistedRuntimeStateRow struct {
	RuleID         uuid.UUID
	CorrelationKey string
	StateType      string
	StateData      []byte
	FirstSeenAt    time.Time
	LastSeenAt     time.Time
	ExpiresAt      time.Time
}

type persistentRuntimeStore struct {
	db *pgxpool.Pool
}

func newPersistentRuntimeStore(db *pgxpool.Pool) *persistentRuntimeStore {
	return &persistentRuntimeStore{db: db}
}

func (s *persistentRuntimeStore) upsertState(
	ctx context.Context,
	ruleID uuid.UUID,
	correlationKey string,
	stateType string,
	stateData any,
	firstSeenAt, lastSeenAt, expiresAt time.Time,
) error {
	payload, err := json.Marshal(stateData)
	if err != nil {
		return err
	}

	const q = `
		INSERT INTO correlation_state (
			id,
			rule_id,
			correlation_key,
			state_type,
			state_data,
			first_seen_at,
			last_seen_at,
			expires_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (rule_id, correlation_key)
		DO UPDATE SET
			state_type = EXCLUDED.state_type,
			state_data = EXCLUDED.state_data,
			first_seen_at = EXCLUDED.first_seen_at,
			last_seen_at = EXCLUDED.last_seen_at,
			expires_at = EXCLUDED.expires_at
	`

	_, err = s.db.Exec(ctx, q,
		uuid.New(),
		ruleID,
		correlationKey,
		stateType,
		payload,
		firstSeenAt,
		lastSeenAt,
		expiresAt,
	)
	return err
}

func (s *persistentRuntimeStore) deleteState(ctx context.Context, ruleID uuid.UUID, correlationKey string) error {
	const q = `
		DELETE FROM correlation_state
		WHERE rule_id = $1
		  AND correlation_key = $2
	`
	_, err := s.db.Exec(ctx, q, ruleID, correlationKey)
	return err
}

func (s *persistentRuntimeStore) loadActiveStates(ctx context.Context) ([]persistedRuntimeStateRow, error) {
	const q = `
		SELECT rule_id, correlation_key, state_type, state_data, first_seen_at, last_seen_at, expires_at
		FROM correlation_state
		WHERE expires_at > NOW()
	`

	rows, err := s.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]persistedRuntimeStateRow, 0, 128)
	for rows.Next() {
		var row persistedRuntimeStateRow
		if err := rows.Scan(
			&row.RuleID,
			&row.CorrelationKey,
			&row.StateType,
			&row.StateData,
			&row.FirstSeenAt,
			&row.LastSeenAt,
			&row.ExpiresAt,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *persistentRuntimeStore) pruneExpiredStates(ctx context.Context) error {
	const q = `DELETE FROM correlation_state WHERE expires_at <= NOW()`
	_, err := s.db.Exec(ctx, q)
	return err
}

func (s *persistentRuntimeStore) tryAcquireAlertDedup(ctx context.Context, dedupKey string, expiresAt time.Time) (bool, error) {
	const cleanup = `
		DELETE FROM detection_alert_dedup
		WHERE dedup_key = $1
		  AND expires_at <= NOW()
	`
	if _, err := s.db.Exec(ctx, cleanup, dedupKey); err != nil {
		return false, err
	}

	const insert = `
		INSERT INTO detection_alert_dedup (dedup_key, expires_at, created_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (dedup_key) DO NOTHING
	`
	cmd, err := s.db.Exec(ctx, insert, dedupKey, expiresAt)
	if err != nil {
		return false, err
	}

	return cmd.RowsAffected() == 1, nil
}

func (s *persistentRuntimeStore) saveCheckpoint(ctx context.Context, checkpoint runtimeCheckpoint) error {
	const q = `
		INSERT INTO engine_processing_checkpoint (engine_name, last_event_id, last_event_time, updated_at)
		VALUES ('rule_engine', $1, $2, NOW())
		ON CONFLICT (engine_name)
		DO UPDATE SET
			last_event_id = EXCLUDED.last_event_id,
			last_event_time = EXCLUDED.last_event_time,
			updated_at = NOW()
	`

	_, err := s.db.Exec(ctx, q, checkpoint.LastEventID, checkpoint.LastEventTime)
	return err
}

func (s *persistentRuntimeStore) loadCheckpoint(ctx context.Context) (runtimeCheckpoint, bool, error) {
	const q = `
		SELECT last_event_id, last_event_time
		FROM engine_processing_checkpoint
		WHERE engine_name = 'rule_engine'
	`

	var checkpoint runtimeCheckpoint
	err := s.db.QueryRow(ctx, q).Scan(&checkpoint.LastEventID, &checkpoint.LastEventTime)
	if err != nil {
		if err == pgx.ErrNoRows {
			return runtimeCheckpoint{}, false, nil
		}
		return runtimeCheckpoint{}, false, err
	}

	return checkpoint, true, nil
}
