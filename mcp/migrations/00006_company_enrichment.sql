-- +goose Up

-- Enriches companies with structured fields from Revli startup and investor
-- exports. Adds a short description column plus AI-generated intel as JSON text.
-- Date fields stored as TEXT (YYYY-MM-DD) to keep sqlc type mapping simple.
-- Adds email_status and department to leads for Revli contact data.

ALTER TABLE app.companies
    ADD COLUMN description          TEXT,
    ADD COLUMN headquarters         TEXT,
    ADD COLUMN phone                TEXT,
    ADD COLUMN twitter_url          TEXT,
    ADD COLUMN facebook_url         TEXT,
    ADD COLUMN employee_count       INTEGER,
    ADD COLUMN founded_date         TEXT,
    ADD COLUMN annual_revenue       TEXT,
    ADD COLUMN annual_revenue_date  TEXT,
    ADD COLUMN technologies         TEXT,
    ADD COLUMN funding_stage        TEXT,
    ADD COLUMN funding_status       TEXT,
    ADD COLUMN funding_amount_last  BIGINT,
    ADD COLUMN funding_date_last    TEXT,
    ADD COLUMN funding_amount_total BIGINT,
    ADD COLUMN top_investors        TEXT,
    ADD COLUMN is_hiring            BOOLEAN,
    ADD COLUMN is_vc                BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN firm_type            TEXT,
    ADD COLUMN stage_focus          TEXT,
    ADD COLUMN check_size           TEXT,
    ADD COLUMN portfolio_size       INTEGER,
    ADD COLUMN industry_focus       TEXT,
    ADD COLUMN geography_focus      TEXT,
    ADD COLUMN intel                TEXT;

ALTER TABLE app.leads
    ADD COLUMN email_status TEXT,
    ADD COLUMN department   TEXT;

-- +goose Down
ALTER TABLE app.leads
    DROP COLUMN department,
    DROP COLUMN email_status;

ALTER TABLE app.companies
    DROP COLUMN intel,
    DROP COLUMN geography_focus,
    DROP COLUMN industry_focus,
    DROP COLUMN portfolio_size,
    DROP COLUMN check_size,
    DROP COLUMN stage_focus,
    DROP COLUMN firm_type,
    DROP COLUMN is_vc,
    DROP COLUMN is_hiring,
    DROP COLUMN top_investors,
    DROP COLUMN funding_amount_total,
    DROP COLUMN funding_date_last,
    DROP COLUMN funding_amount_last,
    DROP COLUMN funding_status,
    DROP COLUMN funding_stage,
    DROP COLUMN technologies,
    DROP COLUMN annual_revenue_date,
    DROP COLUMN annual_revenue,
    DROP COLUMN founded_date,
    DROP COLUMN employee_count,
    DROP COLUMN facebook_url,
    DROP COLUMN twitter_url,
    DROP COLUMN phone,
    DROP COLUMN headquarters,
    DROP COLUMN description;
