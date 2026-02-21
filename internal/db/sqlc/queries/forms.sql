-- name: ListForms :many
SELECT * FROM forms
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: GetFormByID :one
SELECT * FROM forms
WHERE id = $1
    LIMIT 1;

-- name: CreateForm :one
INSERT INTO forms (user_id, title, description)
VALUES ($1, $2, $3)
    RETURNING *;

-- name: UpdateForm :one
UPDATE forms
SET title       = $2,
    description = $3,
    updated_at  = now()
WHERE id = $1
    RETURNING *;

-- name: DeleteForm :exec
DELETE FROM forms
WHERE id = $1;

-- name: IncrementResponseCount :one
UPDATE forms
SET response_count = response_count + 1,
    updated_at     = now()
WHERE id = $1
    RETURNING *;