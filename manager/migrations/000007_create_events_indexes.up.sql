CREATE INDEX idx_events_received_at ON events (received_at);
CREATE INDEX idx_events_hostname ON events (hostname);
CREATE INDEX idx_events_level ON events (severity);
CREATE INDEX idx_events_category ON events (category);