-- +goose Up
ALTER TABLE app.leads
    ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN sourced_at TIMESTAMPTZ;

ALTER TABLE app.companies
    ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.set_updated_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER leads_set_updated_at
    BEFORE UPDATE ON app.leads
    FOR EACH ROW EXECUTE FUNCTION app.set_updated_at();

CREATE TRIGGER companies_set_updated_at
    BEFORE UPDATE ON app.companies
    FOR EACH ROW EXECUTE FUNCTION app.set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS leads_set_updated_at ON app.leads;
DROP TRIGGER IF EXISTS companies_set_updated_at ON app.companies;
DROP FUNCTION IF EXISTS app.set_updated_at();

ALTER TABLE app.leads
    DROP COLUMN IF EXISTS created_at,
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS sourced_at;

ALTER TABLE app.companies
    DROP COLUMN IF EXISTS created_at,
    DROP COLUMN IF EXISTS updated_at;
