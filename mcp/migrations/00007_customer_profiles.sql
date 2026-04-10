-- +goose Up

-- Adds customer_profiles for brand-level ICPs (Ideal Customer Profiles).
-- A brand can have multiple active profiles used to score and filter target accounts.
-- Criteria columns use TEXT[] so filters can be applied with the && overlap operator.

CREATE TABLE app.customer_profiles (
    id                 SERIAL  PRIMARY KEY,
    brand_id           INTEGER NOT NULL REFERENCES app.brands(id),
    name               TEXT    NOT NULL,
    description        TEXT,
    target_industries  TEXT[],
    target_geographies TEXT[],
    target_tech        TEXT[],
    funding_stages     TEXT[],
    min_employees      INTEGER,
    max_employees      INTEGER,
    min_funding_usd    BIGINT,
    max_funding_usd    BIGINT,
    is_active          BOOLEAN NOT NULL DEFAULT TRUE,
    created_at         TIMESTAMP NOT NULL DEFAULT NOW()
);

GRANT SELECT, INSERT, UPDATE, DELETE ON app.customer_profiles TO app;
GRANT USAGE ON SEQUENCE app.customer_profiles_id_seq TO app;

-- +goose Down
DROP TABLE app.customer_profiles;
