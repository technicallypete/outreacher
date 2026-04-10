-- +goose Up
-- Grant DML on the app schema to the mcp runtime user.
-- The mcp server only touches the app schema — nextauth is Next.js only.

GRANT USAGE ON SCHEMA app TO mcp;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES    IN SCHEMA app TO mcp;
GRANT USAGE                          ON ALL SEQUENCES IN SCHEMA app TO mcp;

-- Enumerate current custom types (ON ALL TYPES is not valid SQL).
GRANT USAGE ON TYPE app.lead_status   TO mcp;
GRANT USAGE ON TYPE app.org_role      TO mcp;
GRANT USAGE ON TYPE app.campaign_role TO mcp;

-- Automatically grant DML on future objects created by admin in this schema.
ALTER DEFAULT PRIVILEGES FOR ROLE admin IN SCHEMA app
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES    TO mcp;
ALTER DEFAULT PRIVILEGES FOR ROLE admin IN SCHEMA app
  GRANT USAGE                          ON SEQUENCES TO mcp;
ALTER DEFAULT PRIVILEGES FOR ROLE admin IN SCHEMA app
  GRANT USAGE                          ON TYPES     TO mcp;

-- +goose Down
ALTER DEFAULT PRIVILEGES FOR ROLE admin IN SCHEMA app
  REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES    FROM mcp;
ALTER DEFAULT PRIVILEGES FOR ROLE admin IN SCHEMA app
  REVOKE USAGE                          ON SEQUENCES FROM mcp;
ALTER DEFAULT PRIVILEGES FOR ROLE admin IN SCHEMA app
  REVOKE USAGE                          ON TYPES     FROM mcp;

REVOKE ALL ON ALL TABLES    IN SCHEMA app FROM mcp;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA app FROM mcp;
REVOKE USAGE ON TYPE app.lead_status   FROM mcp;
REVOKE USAGE ON TYPE app.org_role      FROM mcp;
REVOKE USAGE ON TYPE app.campaign_role FROM mcp;
REVOKE USAGE ON SCHEMA app FROM mcp;
