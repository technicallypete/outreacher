-- +goose Up

-- New identifiers table: (type, value) is the unique dedup key.
-- Supported types: email, linkedin, twitter, phone, etc.
CREATE TABLE app.contact_identifiers (
    lead_id  INTEGER NOT NULL REFERENCES app.leads(id) ON DELETE CASCADE,
    type     TEXT    NOT NULL,
    value    TEXT    NOT NULL,
    PRIMARY KEY (type, value)
);

-- Migrate existing email values.
INSERT INTO app.contact_identifiers (lead_id, type, value)
SELECT id, 'email', email
FROM app.leads
WHERE email IS NOT NULL AND email != '';

-- Migrate existing linkedin_url values.
INSERT INTO app.contact_identifiers (lead_id, type, value)
SELECT id, 'linkedin', linkedin_url
FROM app.leads
WHERE linkedin_url IS NOT NULL AND linkedin_url != ''
ON CONFLICT DO NOTHING;

-- Email is now just a convenience column — not the canonical unique key.
ALTER TABLE app.leads ALTER COLUMN email DROP NOT NULL;
ALTER TABLE app.leads DROP CONSTRAINT leads_email_key;

-- +goose Down
ALTER TABLE app.leads ADD CONSTRAINT leads_email_key UNIQUE (email);
ALTER TABLE app.leads ALTER COLUMN email SET NOT NULL;
DROP TABLE app.contact_identifiers;
