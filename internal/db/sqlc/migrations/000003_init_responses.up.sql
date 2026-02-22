BEGIN;

CREATE TABLE form_responses (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    form_id    UUID        NOT NULL REFERENCES forms(id) ON DELETE RESTRICT,
    user_id    UUID        REFERENCES users(id) ON DELETE SET NULL,
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE response_answers (
    id          BIGSERIAL   PRIMARY KEY,
    response_id UUID        NOT NULL REFERENCES form_responses(id) ON DELETE CASCADE,
    question_id UUID        NOT NULL,
    form_id     UUID        NOT NULL,
    answer_text TEXT,
    FOREIGN KEY (form_id, question_id) REFERENCES questions(form_id, id) ON DELETE CASCADE
);

CREATE INDEX idx_form_responses_form_id ON form_responses(form_id);
CREATE INDEX idx_response_answers_response_id ON response_answers(response_id);

COMMIT;
