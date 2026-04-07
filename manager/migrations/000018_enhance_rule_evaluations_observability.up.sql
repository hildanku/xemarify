ALTER TABLE rule_evaluations
    ADD COLUMN IF NOT EXISTS matched BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS reason TEXT,
    ADD COLUMN IF NOT EXISTS correlation_key TEXT,
    ADD COLUMN IF NOT EXISTS evaluation_details JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS evaluated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE INDEX IF NOT EXISTS idx_rule_evaluations_rule_event_received
    ON rule_evaluations (rule_id, event_id, received_at);

CREATE INDEX IF NOT EXISTS idx_rule_evaluations_matched_evaluated_at
    ON rule_evaluations (matched, evaluated_at DESC);
