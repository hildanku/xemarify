DROP INDEX IF EXISTS idx_rule_evaluations_matched_evaluated_at;
DROP INDEX IF EXISTS idx_rule_evaluations_rule_event_received;

ALTER TABLE rule_evaluations
    DROP COLUMN IF EXISTS evaluated_at,
    DROP COLUMN IF EXISTS evaluation_details,
    DROP COLUMN IF EXISTS correlation_key,
    DROP COLUMN IF EXISTS reason,
    DROP COLUMN IF EXISTS matched;
