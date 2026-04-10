-- +goose Up
CREATE SCHEMA IF NOT EXISTS app;

CREATE TYPE app.lead_status AS ENUM ('new', 'contacted', 'qualified', 'disqualified', 'converted');

CREATE TABLE app.companies (
    id       SERIAL PRIMARY KEY,
    name     TEXT NOT NULL,
    domain   TEXT,
    industry TEXT
);

CREATE TABLE app.signals (
    id          SERIAL PRIMARY KEY,
    description TEXT NOT NULL
);

CREATE TABLE app.company_signals (
    company_id INTEGER NOT NULL REFERENCES app.companies(id),
    signal_id  INTEGER NOT NULL REFERENCES app.signals(id),
    PRIMARY KEY (company_id, signal_id)
);

CREATE TABLE app.leads (
    id         SERIAL PRIMARY KEY,
    name       TEXT NOT NULL,
    email      TEXT NOT NULL UNIQUE,
    company_id INTEGER REFERENCES app.companies(id),
    title      TEXT,
    status     app.lead_status NOT NULL DEFAULT 'new',
    score      REAL
);

CREATE TABLE app.notes (
    id         SERIAL PRIMARY KEY,
    lead_id    INTEGER NOT NULL REFERENCES app.leads(id),
    content    TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Grant schema access to the runtime app user.
-- Grant schema access to the runtime app user.
GRANT USAGE ON SCHEMA app TO app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES    IN SCHEMA app TO app;
GRANT USAGE                          ON ALL SEQUENCES IN SCHEMA app TO app;
GRANT USAGE ON TYPE app.lead_status TO app;

-- Automatically grant DML on all future objects created by admin in this schema.
ALTER DEFAULT PRIVILEGES FOR ROLE admin IN SCHEMA app
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES    TO app;
ALTER DEFAULT PRIVILEGES FOR ROLE admin IN SCHEMA app
  GRANT USAGE                          ON SEQUENCES TO app;
ALTER DEFAULT PRIVILEGES FOR ROLE admin IN SCHEMA app
  GRANT USAGE                          ON TYPES     TO app;

-- +goose Down
DROP TABLE app.notes;
DROP TABLE app.leads;
DROP TABLE app.company_signals;
DROP TABLE app.signals;
DROP TABLE app.companies;
DROP TYPE app.lead_status;
DROP SCHEMA app;
