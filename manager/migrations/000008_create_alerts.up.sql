CREATE TABLE alerts (
    id          UUID PRIMARY KEY,
    rule_id     UUID NOT NULL,
    severity    TEXT,
    correlation_key TEXT,
    triggered_at TIMESTAMPTZ NOT NULL,
    status      TEXT NOT NULL DEFAULT 'new',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
