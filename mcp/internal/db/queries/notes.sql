-- name: GetNotesByLead :many
SELECT id, lead_id, content, created_at
FROM app.notes
WHERE lead_id = $1
ORDER BY created_at ASC;

-- name: CreateNote :one
INSERT INTO app.notes (lead_id, content)
VALUES ($1, $2)
RETURNING id, lead_id, content, created_at;
