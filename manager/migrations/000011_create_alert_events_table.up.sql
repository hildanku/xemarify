CREATE TABLE alert_events (
    alert_id     UUID NOT NULL,
    event_id     UUID NOT NULL,
    received_at  TIMESTAMPTZ NOT NULL,

    PRIMARY KEY (alert_id, event_id, received_at),

    FOREIGN KEY (alert_id)
        REFERENCES alerts(id)
        ON DELETE CASCADE,

    FOREIGN KEY (event_id, received_at)
        REFERENCES events(id, received_at)
        ON DELETE CASCADE
);
