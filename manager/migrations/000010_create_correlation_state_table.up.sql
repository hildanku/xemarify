CREATE TABLE correlation_state (
    id              UUID PRIMARY KEY,
    rule_id         UUID NOT NULL,
    correlation_key TEXT NOT NULL,
    state_data      JSONB NOT NULL,
    first_seen_at   TIMESTAMPTZ NOT NULL,
    last_seen_at    TIMESTAMPTZ NOT NULL,
    expires_at      TIMESTAMPTZ NOT NULL
);

UNIQUE (rule_id, correlation_key)
