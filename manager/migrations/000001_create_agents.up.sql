CREATE TYPE agent_status AS ENUM (
    'ONLINE',
    'OFFLINE'
);

CREATE TABLE agents (
    id              UUID PRIMARY KEY,
    name            TEXT NOT NULL,
    hostname        TEXT,
    key             TEXT NOT NULL,
    ip_address      INET,
    version         TEXT,
    status          agent_status DEFAULT 'ONLINE',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at    TIMESTAMPTZ
);
