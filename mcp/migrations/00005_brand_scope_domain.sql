-- +goose Up

-- Scopes domain objects (companies, leads, contact_identifiers) to a brand so
-- all data is isolated per tenant. Existing rows are assigned to the Default
-- brand of system_default_org. Depends on the seed inserted by migration 00004.

-- ── companies ────────────────────────────────────────────────────────────────

ALTER TABLE app.companies
    ADD COLUMN brand_id INTEGER REFERENCES app.brands(id);

UPDATE app.companies
SET brand_id = (
    SELECT b.id
    FROM app.brands b
    JOIN app.organizations o ON b.organization_id = o.id
    WHERE o.slug = 'system_default_org' AND b.slug = 'default'
);

ALTER TABLE app.companies ALTER COLUMN brand_id SET NOT NULL;

-- Old constraint was on name alone; name is now unique per brand.
ALTER TABLE app.companies DROP CONSTRAINT companies_name_unique;
ALTER TABLE app.companies ADD CONSTRAINT companies_brand_name_unique UNIQUE (brand_id, name);

-- ── leads ────────────────────────────────────────────────────────────────────

ALTER TABLE app.leads
    ADD COLUMN brand_id INTEGER REFERENCES app.brands(id);

UPDATE app.leads
SET brand_id = (
    SELECT b.id
    FROM app.brands b
    JOIN app.organizations o ON b.organization_id = o.id
    WHERE o.slug = 'system_default_org' AND b.slug = 'default'
);

ALTER TABLE app.leads ALTER COLUMN brand_id SET NOT NULL;

-- ── contact_identifiers ──────────────────────────────────────────────────────
-- PK must include brand_id so the same email/linkedin can exist as separate
-- leads across brands (e.g. two brands targeting the same person independently).

ALTER TABLE app.contact_identifiers
    ADD COLUMN brand_id INTEGER REFERENCES app.brands(id);

UPDATE app.contact_identifiers
SET brand_id = (
    SELECT b.id
    FROM app.brands b
    JOIN app.organizations o ON b.organization_id = o.id
    WHERE o.slug = 'system_default_org' AND b.slug = 'default'
);

ALTER TABLE app.contact_identifiers ALTER COLUMN brand_id SET NOT NULL;

ALTER TABLE app.contact_identifiers DROP CONSTRAINT contact_identifiers_pkey;
ALTER TABLE app.contact_identifiers ADD PRIMARY KEY (brand_id, type, value);

-- +goose Down
ALTER TABLE app.contact_identifiers DROP CONSTRAINT contact_identifiers_pkey;
ALTER TABLE app.contact_identifiers ADD PRIMARY KEY (type, value);
ALTER TABLE app.contact_identifiers DROP COLUMN brand_id;

ALTER TABLE app.leads DROP COLUMN brand_id;

ALTER TABLE app.companies DROP CONSTRAINT companies_brand_name_unique;
ALTER TABLE app.companies ADD CONSTRAINT companies_name_unique UNIQUE (name);
ALTER TABLE app.companies DROP COLUMN brand_id;
