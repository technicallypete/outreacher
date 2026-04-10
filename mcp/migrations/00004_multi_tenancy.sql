-- +goose Up

-- Introduces the Organization > Brand > User hierarchy for multi-tenancy.
-- A brand always belongs to exactly one org. All domain objects (companies,
-- leads, contact_identifiers) will be scoped to brand_id in migration 00005.
-- Users relate to orgs and brands via membership tables with explicit roles.
--
-- A minimal seed (system_default_org + Default brand) is included at the tail
-- so that migration 00005 can backfill brand_id on existing rows. The MCP
-- binary re-runs equivalent upserts at startup via ON CONFLICT DO NOTHING.

CREATE TYPE app.org_role   AS ENUM ('owner', 'admin', 'member');
CREATE TYPE app.brand_role AS ENUM ('admin', 'member', 'viewer');

-- Organizations: top-level tenant container.
-- slug is the stable string key used for env var lookups and bootstrap detection.
-- is_system marks the built-in system org so the SaaS UI can filter it out.
CREATE TABLE app.organizations (
    id         SERIAL  PRIMARY KEY,
    name       TEXT    NOT NULL,
    slug       TEXT    NOT NULL UNIQUE,
    is_system  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Brands: a named context within an org. Domain objects are scoped to brand_id.
-- is_default marks the brand created automatically when an org is created.
-- slug is unique within an org.
CREATE TABLE app.brands (
    id              SERIAL  PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES app.organizations(id),
    name            TEXT    NOT NULL,
    slug            TEXT    NOT NULL,
    is_default      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, slug)
);

-- Users: identity records. email is nullable (system users have none).
-- slug is the stable key used for env var lookups and bootstrap detection.
-- is_system marks the built-in system user so the SaaS UI can filter it out.
CREATE TABLE app.users (
    id         SERIAL  PRIMARY KEY,
    email      TEXT    UNIQUE,
    slug       TEXT    NOT NULL UNIQUE,
    name       TEXT    NOT NULL,
    is_system  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Organization memberships: many:many users <-> orgs with a role per pair.
CREATE TABLE app.organization_memberships (
    organization_id INTEGER      NOT NULL REFERENCES app.organizations(id),
    user_id         INTEGER      NOT NULL REFERENCES app.users(id),
    role            app.org_role NOT NULL,
    PRIMARY KEY (organization_id, user_id)
);

-- Brand memberships: many:many users <-> brands with a role per pair.
CREATE TABLE app.brand_memberships (
    brand_id INTEGER        NOT NULL REFERENCES app.brands(id),
    user_id  INTEGER        NOT NULL REFERENCES app.users(id),
    role     app.brand_role NOT NULL,
    PRIMARY KEY (brand_id, user_id)
);

-- Grant runtime app user access to all new tables and types.
GRANT SELECT, INSERT, UPDATE, DELETE
    ON app.organizations, app.brands, app.users,
       app.organization_memberships, app.brand_memberships
    TO app;
GRANT USAGE ON ALL SEQUENCES IN SCHEMA app TO app;
GRANT USAGE ON TYPE app.org_role, app.brand_role TO app;

-- ── Minimal seed ─────────────────────────────────────────────────────────────
-- Insert just enough for migration 00005 to backfill brand_id on existing rows.
-- The MCP binary calls equivalent upserts at startup (ON CONFLICT DO NOTHING),
-- so these rows are safe to insert here and safe to re-check later.
-- Users and memberships are intentionally omitted here; the binary creates them.

INSERT INTO app.organizations (name, slug, is_system)
VALUES ('System Default', 'system_default_org', TRUE)
ON CONFLICT (slug) DO NOTHING;

INSERT INTO app.brands (organization_id, name, slug, is_default)
SELECT id, 'Default', 'default', TRUE
FROM app.organizations
WHERE slug = 'system_default_org'
ON CONFLICT (organization_id, slug) DO NOTHING;

-- +goose Down
DELETE FROM app.brands
WHERE slug = 'default'
  AND organization_id = (SELECT id FROM app.organizations WHERE slug = 'system_default_org');

DELETE FROM app.organizations WHERE slug = 'system_default_org';

DROP TABLE app.brand_memberships;
DROP TABLE app.organization_memberships;
DROP TABLE app.users;
DROP TABLE app.brands;
DROP TABLE app.organizations;
DROP TYPE app.brand_role;
DROP TYPE app.org_role;
