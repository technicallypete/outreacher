-- name: GetOrgBySlug :one
-- Look up an organization by its stable slug identifier.
SELECT id, name, slug, is_system FROM app.organizations WHERE slug = $1;

-- name: GetUserBySlug :one
-- Look up a user by their stable slug identifier.
SELECT id, name, slug, is_system FROM app.users WHERE slug = $1;

-- name: GetCampaign :one
-- Look up a campaign by ID.
SELECT id, name, slug, is_default FROM app.campaigns WHERE id = $1;

-- name: GetDefaultCampaignForOrg :one
-- Returns the is_default campaign for an org, falling back to the lowest id if
-- none is explicitly flagged. Used by the binary to resolve its operating campaign.
SELECT id, name, slug, is_default
FROM app.campaigns
WHERE organization_id = $1
ORDER BY is_default DESC, id ASC
LIMIT 1;

-- name: GetOrgMembership :one
-- Check whether a user is already a member of an org.
SELECT role FROM app.organization_memberships
WHERE organization_id = $1 AND user_id = $2;

-- name: GetCampaignMembership :one
-- Check whether a user is already a member of a campaign.
SELECT role FROM app.campaign_memberships
WHERE campaign_id = $1 AND user_id = $2;

-- name: CreateOrganization :one
-- Insert a new organization. Caller is responsible for creating the Default campaign.
INSERT INTO app.organizations (name, slug, is_system)
VALUES ($1, $2, $3)
RETURNING id, name, slug, is_system;

-- name: CreateCampaign :one
-- Insert a new campaign under an org. is_default should be TRUE for the first
-- campaign created for an org (the "Default" campaign created on org creation).
INSERT INTO app.campaigns (organization_id, name, slug, is_default)
VALUES ($1, $2, $3, $4)
RETURNING id, name, slug, is_default;

-- name: CreateUser :one
-- Insert a new user. email is optional (system users have none).
INSERT INTO app.users (name, slug, email, is_system)
VALUES ($1, $2, $3, $4)
RETURNING id, name, slug, is_system;

-- name: AddOrgMember :exec
-- Add a user to an org with the given role. No-op on conflict so callers can
-- call this idempotently during bootstrap.
INSERT INTO app.organization_memberships (organization_id, user_id, role)
VALUES ($1, $2, $3)
ON CONFLICT (organization_id, user_id) DO NOTHING;

-- name: AddCampaignMember :exec
-- Add a user to a campaign with the given role. No-op on conflict so callers can
-- call this idempotently during bootstrap.
INSERT INTO app.campaign_memberships (campaign_id, user_id, role)
VALUES ($1, $2, $3)
ON CONFLICT (campaign_id, user_id) DO NOTHING;

-- name: ListOrganizations :many
-- List all organizations a user is a member of, ordered by name.
SELECT o.id, o.name, o.slug, o.is_system, om.role
FROM app.organizations o
JOIN app.organization_memberships om ON o.id = om.organization_id
WHERE om.user_id = $1
ORDER BY o.name;

-- name: ListCampaigns :many
-- List all campaigns belonging to an org, ordered by is_default desc then name.
SELECT c.id, c.name, c.slug, c.is_default
FROM app.campaigns c
WHERE c.organization_id = $1
ORDER BY c.is_default DESC, c.name;

-- name: RenameCampaign :one
-- Update the display name of a campaign. Slug is intentionally left unchanged so
-- existing references remain stable.
UPDATE app.campaigns SET name = $2 WHERE id = $1 RETURNING id, name, slug, is_default;
