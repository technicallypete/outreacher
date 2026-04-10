-- name: SearchLeads :many
-- Search leads within a campaign by name/email substring, status, and/or company
-- name substring. All filters are optional and AND-ed together.
SELECT
    l.id,
    l.name,
    l.email,
    l.linkedin_url,
    l.title,
    l.status,
    l.score,
    c.name AS company
FROM app.leads l
LEFT JOIN app.companies c ON l.company_id = c.id
WHERE
    l.campaign_id = $1
    AND ($2 = '' OR l.name ILIKE '%' || $2 || '%' OR l.email ILIKE '%' || $2 || '%')
    AND ($3 = '' OR l.status::text = $3)
    AND ($4 = '' OR c.name ILIKE '%' || $4 || '%');

-- name: GetLead :one
-- Fetch full lead detail with joined company fields. Lead must belong to the
-- given campaign to prevent cross-campaign data leakage.
SELECT
    l.id,
    l.name,
    l.email,
    l.title,
    l.status,
    l.score,
    l.company_id,
    l.linkedin_url,
    l.location,
    l.phone,
    l.created_at,
    l.updated_at,
    l.sourced_at,
    c.name     AS company,
    c.domain   AS domain,
    c.industry AS industry
FROM app.leads l
LEFT JOIN app.companies c ON l.company_id = c.id
WHERE l.id = $1 AND l.campaign_id = $2;

-- name: UpdateLeadStatus :one
-- Advance a lead's status. Scoped to campaign_id to prevent cross-campaign updates.
UPDATE app.leads
SET status = $3
WHERE id = $1 AND campaign_id = $2
RETURNING id, name, status;

-- name: CreateLead :one
-- Insert a new lead into a campaign.
INSERT INTO app.leads (campaign_id, name, email, company_id, title, status, score, linkedin_url, location, phone, email_status, department, sourced_at)
VALUES ($1, $2, $3, $4, $5, 'new', $6, $7, $8, $9, $10, $11, $12)
RETURNING id;

-- name: UpdateLeadFields :exec
-- Update mutable lead fields. COALESCE ensures existing values are not
-- overwritten by nulls when a re-import omits optional columns.
-- updated_at is handled automatically by the leads_set_updated_at trigger.
-- Scoped to campaign_id to prevent cross-campaign updates.
UPDATE app.leads SET
    name         = $3,
    email        = COALESCE($4, email),
    company_id   = COALESCE($5, company_id),
    title        = COALESCE($6, title),
    score        = COALESCE($7, score),
    linkedin_url = COALESCE($8, linkedin_url),
    location     = COALESCE($9, location),
    phone        = COALESCE($10, phone),
    email_status = COALESCE($11, email_status),
    department   = COALESCE($12, department),
    sourced_at   = COALESCE($13, sourced_at)
WHERE id = $1 AND campaign_id = $2;
