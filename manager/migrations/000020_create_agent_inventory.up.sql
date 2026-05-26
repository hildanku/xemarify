CREATE TABLE agent_inventory (
    agent_id         UUID PRIMARY KEY REFERENCES agents(id) ON DELETE CASCADE,
    os               TEXT,
    arch             TEXT,
    kernel_version   TEXT,
    cpu_model        TEXT,
    cpu_cores        INT,
    memory_total_mb  BIGINT,
    uptime_seconds   BIGINT,
    ip_addresses     TEXT[],
    nginx_installed  BOOLEAN NOT NULL DEFAULT FALSE,
    apache_installed BOOLEAN NOT NULL DEFAULT FALSE,
    collected_at     TIMESTAMPTZ,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
