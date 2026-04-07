DROP TABLE IF EXISTS engine_processing_checkpoint;

DROP INDEX IF EXISTS idx_detection_alert_dedup_expires_at;
DROP TABLE IF EXISTS detection_alert_dedup;

DROP INDEX IF EXISTS idx_correlation_state_expires_at;
ALTER TABLE correlation_state
    DROP COLUMN IF EXISTS state_type;
