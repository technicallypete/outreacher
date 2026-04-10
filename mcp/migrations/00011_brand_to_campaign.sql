-- +goose Up

-- Rename Brand → Campaign throughout.
-- Brands are now called Campaigns: a named ICP context within an org.
-- This migration renames tables, columns, constraints, types, and grants.

-- 1. Rename the enum type
ALTER TYPE app.brand_role RENAME TO campaign_role;

-- 2. Rename the brands table → campaigns
ALTER TABLE app.brands RENAME TO campaigns;

-- 3. Rename brand_memberships → campaign_memberships
ALTER TABLE app.brand_memberships RENAME TO campaign_memberships;

-- 4. Rename brand_id column in campaign_memberships
ALTER TABLE app.campaign_memberships RENAME COLUMN brand_id TO campaign_id;

-- 5. Rename role column type reference (implicit via enum rename — no action needed)

-- 6. Rename brand_id → campaign_id on domain tables
ALTER TABLE app.companies         RENAME COLUMN brand_id TO campaign_id;
ALTER TABLE app.leads             RENAME COLUMN brand_id TO campaign_id;
ALTER TABLE app.contact_identifiers RENAME COLUMN brand_id TO campaign_id;
ALTER TABLE app.customer_profiles RENAME COLUMN brand_id TO campaign_id;
ALTER TABLE app.chat_threads      RENAME COLUMN brand_id TO campaign_id;

-- 7. Update FK constraint names for clarity (optional but keeps pg_dump clean)
ALTER TABLE app.campaign_memberships
    RENAME CONSTRAINT brand_memberships_brand_id_fkey TO campaign_memberships_campaign_id_fkey;
ALTER TABLE app.campaign_memberships
    RENAME CONSTRAINT brand_memberships_user_id_fkey TO campaign_memberships_user_id_fkey;
ALTER TABLE app.campaign_memberships
    RENAME CONSTRAINT brand_memberships_pkey TO campaign_memberships_pkey;

-- 8. Revoke old grants, re-grant on renamed tables
REVOKE ALL ON app.campaigns, app.campaign_memberships FROM app;
GRANT SELECT, INSERT, UPDATE, DELETE ON app.campaigns, app.campaign_memberships TO app;
GRANT USAGE ON TYPE app.campaign_role TO app;

-- +goose Down

REVOKE ALL ON app.campaigns, app.campaign_memberships FROM app;
GRANT SELECT, INSERT, UPDATE, DELETE ON app.campaigns, app.campaign_memberships TO app;
GRANT USAGE ON TYPE app.campaign_role TO app;

ALTER TABLE app.campaign_memberships
    RENAME CONSTRAINT campaign_memberships_pkey TO brand_memberships_pkey;
ALTER TABLE app.campaign_memberships
    RENAME CONSTRAINT campaign_memberships_user_id_fkey TO brand_memberships_user_id_fkey;
ALTER TABLE app.campaign_memberships
    RENAME CONSTRAINT campaign_memberships_campaign_id_fkey TO brand_memberships_brand_id_fkey;

ALTER TABLE app.chat_threads        RENAME COLUMN campaign_id TO brand_id;
ALTER TABLE app.customer_profiles   RENAME COLUMN campaign_id TO brand_id;
ALTER TABLE app.contact_identifiers RENAME COLUMN campaign_id TO brand_id;
ALTER TABLE app.leads               RENAME COLUMN campaign_id TO brand_id;
ALTER TABLE app.companies           RENAME COLUMN campaign_id TO brand_id;

ALTER TABLE app.campaign_memberships RENAME COLUMN campaign_id TO brand_id;
ALTER TABLE app.campaign_memberships RENAME TO brand_memberships;
ALTER TABLE app.campaigns RENAME TO brands;
ALTER TYPE app.campaign_role RENAME TO brand_role;
