-- Runs once at postgres container first-start as the superuser.
-- Only creates runtime roles and grants database connection.
-- Schema-level grants and DEFAULT PRIVILEGES are applied in migrations
-- after the schemas are created.

DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'app') THEN
    CREATE ROLE app WITH LOGIN PASSWORD 'app';
  END IF;
END
$$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'mcp') THEN
    CREATE ROLE mcp WITH LOGIN PASSWORD 'mcp';
  END IF;
END
$$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'reporter') THEN
    CREATE ROLE reporter WITH LOGIN PASSWORD 'reporter';
  END IF;
END
$$;

GRANT CONNECT ON DATABASE outreacher TO app;
GRANT CONNECT ON DATABASE outreacher TO mcp;
GRANT CONNECT ON DATABASE outreacher TO reporter;
