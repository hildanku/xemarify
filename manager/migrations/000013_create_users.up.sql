CREATE TABLE users (
    id          UUID PRIMARY KEY,
    username    VARCHAR(50)  UNIQUE NOT NULL,
    email       VARCHAR(100) UNIQUE NOT NULL,
    role        VARCHAR(50)  NOT NULL CHECK (role IN ('MANAGER', 'ANALYST', 'VIEWER')),
    avatar      TEXT,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP
);
