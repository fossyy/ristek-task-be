BEGIN;

CREATE TYPE question_type AS ENUM (
  'short_text',
  'long_text',
  'multiple_choice',
  'checkbox',
  'dropdown',
  'date',
  'rating'
);

CREATE TABLE forms (
                       id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
                       user_id        UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                       title          TEXT        NOT NULL,
                       description    TEXT,
                       response_count INTEGER     NOT NULL DEFAULT 0,
                       created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
                       updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE questions (
                           id       UUID          NOT NULL DEFAULT gen_random_uuid(),
                           form_id  UUID          NOT NULL REFERENCES forms(id) ON DELETE CASCADE,
                           type     question_type NOT NULL,
                           title    TEXT          NOT NULL,
                           required BOOLEAN       NOT NULL DEFAULT false,
                           position INTEGER       NOT NULL,
                           PRIMARY KEY (form_id, id)
);

CREATE TABLE question_options (
                                  id          SERIAL  PRIMARY KEY,
                                  form_id     UUID    NOT NULL,
                                  question_id UUID    NOT NULL,
                                  label       TEXT    NOT NULL,
                                  position    INTEGER NOT NULL,
                                  FOREIGN KEY (form_id, question_id)
                                      REFERENCES questions(form_id, id) ON DELETE CASCADE
);

CREATE INDEX idx_forms_user_id ON forms(user_id);
CREATE INDEX idx_questions_form_id ON questions(form_id);
CREATE INDEX idx_question_options_question ON question_options(form_id, question_id);

COMMIT;