BEGIN;

DROP INDEX IF EXISTS idx_response_answers_response_id;
DROP INDEX IF EXISTS idx_form_responses_form_id;
DROP TABLE IF EXISTS response_answers;
DROP TABLE IF EXISTS form_responses;

COMMIT;
