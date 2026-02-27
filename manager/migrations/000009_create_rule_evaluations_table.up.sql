CREATE TABLE rule_evaluations (
    id              BIGSERIAL PRIMARY KEY,
    rule_id         UUID NOT NULL,
    event_id        UUID NOT NULL,
    received_at     TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    FOREIGN KEY (event_id, received_at)
    REFERENCES events(id, received_at)
    ON DELETE CASCADE
);
