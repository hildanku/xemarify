CREATE TABLE events (
    id              UUID,
    event_time      TIMESTAMPTZ,
    received_at     TIMESTAMPTZ NOT NULL,

    agent_id        UUID,
    hostname        TEXT,
    source_ip       INET,
    input_type      TEXT,

    facility        TEXT,
    severity        TEXT,
    category        TEXT,

    message         TEXT NOT NULL,
    normalized      JSONB,
    raw             TEXT,

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (id, received_at)
)
PARTITION BY RANGE (received_at);
