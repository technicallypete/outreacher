-- +goose Up
-- NextAuth v5 tables in a dedicated schema to avoid collision with app.users.
-- Next.js connects with search_path=nextauth,app so @auth/pg-adapter finds
-- its standard table names without qualification.

CREATE SCHEMA IF NOT EXISTS nextauth;

CREATE TABLE IF NOT EXISTS nextauth.users (
    id TEXT NOT NULL PRIMARY KEY,
    name TEXT,
    email TEXT UNIQUE,
    "emailVerified" TIMESTAMPTZ,
    image TEXT
);

CREATE TABLE IF NOT EXISTS nextauth.accounts (
    id TEXT NOT NULL PRIMARY KEY,
    "userId" TEXT NOT NULL REFERENCES nextauth.users(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    provider TEXT NOT NULL,
    "providerAccountId" TEXT NOT NULL,
    refresh_token TEXT,
    access_token TEXT,
    expires_at INTEGER,
    token_type TEXT,
    scope TEXT,
    id_token TEXT,
    session_state TEXT,
    UNIQUE(provider, "providerAccountId")
);

CREATE TABLE IF NOT EXISTS nextauth.sessions (
    id TEXT NOT NULL PRIMARY KEY,
    "sessionToken" TEXT NOT NULL UNIQUE,
    "userId" TEXT NOT NULL REFERENCES nextauth.users(id) ON DELETE CASCADE,
    expires TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS nextauth.verification_token (
    identifier TEXT NOT NULL,
    token TEXT NOT NULL,
    expires TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (identifier, token)
);

-- Link NextAuth users to outreacher app users via email.
-- Populated during onboarding when the user creates their org.
ALTER TABLE app.users ADD COLUMN IF NOT EXISTS nextauth_id TEXT UNIQUE;

-- Grant DML to app user on the nextauth schema
GRANT USAGE ON SCHEMA nextauth TO app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA nextauth TO app;
ALTER DEFAULT PRIVILEGES IN SCHEMA nextauth GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO app;

-- +goose Down
ALTER TABLE app.users DROP COLUMN IF EXISTS nextauth_id;
DROP SCHEMA IF EXISTS nextauth CASCADE;
