ALTER TABLE correlation_state
    ADD COLUMN state_type TEXT NOT NULL DEFAULT 'threshold';

CREATE INDEX idx_correlation_state_expires_at
    ON correlation_state (expires_at);

CREATE TABLE detection_alert_dedup (
    dedup_key   TEXT PRIMARY KEY,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_detection_alert_dedup_expires_at
    ON detection_alert_dedup (expires_at);

CREATE TABLE engine_processing_checkpoint (
    engine_name      TEXT PRIMARY KEY,
    last_event_id    UUID NOT NULL,
    last_event_time  TIMESTAMPTZ NOT NULL,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
