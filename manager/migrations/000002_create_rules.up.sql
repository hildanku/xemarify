CREATE TYPE severity AS ENUM (
    'INFO',
    'LOW',
    'MEDIUM',
    'HIGH',
    'CRITICAL'
);

CREATE TABLE rules (
    id          UUID PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT,
    level       severity NOT NULL,
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    condition   JSONB NOT NULL,
    tags        TEXT[] DEFAULT '{}',
    version INT NOT NULL DEFAULT 1,
    created_by UUID,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
