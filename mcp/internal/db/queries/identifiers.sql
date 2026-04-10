-- name: FindLeadByIdentifier :one
-- Find a lead by contact identifier within a campaign. campaign_id is part of the PK
-- so the same email/linkedin can exist as separate leads across campaigns.
SELECT lead_id FROM app.contact_identifiers
WHERE campaign_id = $1 AND type = $2 AND value = $3;

-- name: UpsertIdentifier :exec
-- Attach a contact identifier to a lead within a campaign. No-op on conflict.
INSERT INTO app.contact_identifiers (campaign_id, lead_id, type, value)
VALUES ($1, $2, $3, $4)
ON CONFLICT (campaign_id, type, value) DO NOTHING;
