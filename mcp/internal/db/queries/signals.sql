-- name: UpsertSignal :one
INSERT INTO app.signals (description)
VALUES ($1)
ON CONFLICT (description) DO UPDATE SET description = EXCLUDED.description
RETURNING id;

-- name: UpsertSignalKeyword :exec
INSERT INTO app.signal_keywords (signal_id, keyword)
VALUES ($1, $2)
ON CONFLICT (signal_id, keyword) DO NOTHING;

-- name: UpsertCompanySignal :exec
INSERT INTO app.company_signals (company_id, signal_id)
VALUES ($1, $2)
ON CONFLICT (company_id, signal_id) DO NOTHING;

-- name: ListCompanySignals :many
SELECT
    s.id,
    s.description,
    array_agg(sk.keyword) FILTER (WHERE sk.keyword IS NOT NULL) AS keywords
FROM app.company_signals cs
JOIN app.signals s ON cs.signal_id = s.id
LEFT JOIN app.signal_keywords sk ON sk.signal_id = s.id
WHERE cs.company_id = $1
GROUP BY s.id, s.description;
