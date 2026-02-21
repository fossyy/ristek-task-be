BEGIN;

DROP INDEX IF EXISTS idx_question_options_question;
DROP INDEX IF EXISTS idx_questions_form_id;
DROP INDEX IF EXISTS idx_forms_user_id;
DROP TABLE IF EXISTS question_options;
DROP TABLE IF EXISTS questions;
DROP TABLE IF EXISTS forms;
DROP TYPE IF EXISTS question_type;

COMMIT;