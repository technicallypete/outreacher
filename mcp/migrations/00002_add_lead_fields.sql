-- +goose Up
ALTER TABLE app.leads ADD COLUMN linkedin_url TEXT;
ALTER TABLE app.leads ADD COLUMN location     TEXT;
ALTER TABLE app.leads ADD COLUMN phone        TEXT;

ALTER TABLE app.companies ADD COLUMN linkedin_url TEXT;

ALTER TABLE app.companies ADD CONSTRAINT companies_name_unique UNIQUE (name);
ALTER TABLE app.signals   ADD CONSTRAINT signals_description_unique UNIQUE (description);

CREATE TABLE app.signal_keywords (
    id        SERIAL  PRIMARY KEY,
    signal_id INTEGER NOT NULL REFERENCES app.signals(id) ON DELETE CASCADE,
    keyword   TEXT    NOT NULL,
    UNIQUE (signal_id, keyword)
);

-- +goose Down
DROP TABLE app.signal_keywords;
ALTER TABLE app.signals   DROP CONSTRAINT signals_description_unique;
ALTER TABLE app.companies DROP CONSTRAINT companies_name_unique;
ALTER TABLE app.companies DROP COLUMN linkedin_url;
ALTER TABLE app.leads DROP COLUMN phone;
ALTER TABLE app.leads DROP COLUMN location;
ALTER TABLE app.leads DROP COLUMN linkedin_url;
