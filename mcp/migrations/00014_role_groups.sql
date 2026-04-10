-- +goose Up
-- Introduce group roles for scalable permission management.
-- New tables/sequences/types are auto-granted via DEFAULT PRIVILEGES on the
-- group role rather than on each user individually.
--
-- app_read  — SELECT on app schema  → reporter user
-- app_crud  — full DML on app schema → app, mcp users
--
-- Future schemas (e.g. jobs) follow the same pattern with their own group roles.

CREATE ROLE app_read;
CREATE ROLE app_crud;

-- Schema access.
GRANT USAGE ON SCHEMA app TO app_read;
GRANT USAGE ON SCHEMA app TO app_crud;

-- Table privileges.
GRANT SELECT                         ON ALL TABLES IN SCHEMA app TO app_read;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA app TO app_crud;

-- Sequence privileges (needed for INSERT with serial/identity columns).
GRANT USAGE ON ALL SEQUENCES IN SCHEMA app TO app_crud;

-- Type privileges.
GRANT USAGE ON TYPE app.lead_status   TO app_read;
GRANT USAGE ON TYPE app.lead_status   TO app_crud;
GRANT USAGE ON TYPE app.org_role      TO app_read;
GRANT USAGE ON TYPE app.org_role      TO app_crud;
GRANT USAGE ON TYPE app.campaign_role TO app_read;
GRANT USAGE ON TYPE app.campaign_role TO app_crud;

-- DEFAULT PRIVILEGES: future objects created by admin are auto-granted to group roles.
ALTER DEFAULT PRIVILEGES FOR ROLE admin IN SCHEMA app
  GRANT SELECT                         ON TABLES    TO app_read;
ALTER DEFAULT PRIVILEGES FOR ROLE admin IN SCHEMA app
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES    TO app_crud;
ALTER DEFAULT PRIVILEGES FOR ROLE admin IN SCHEMA app
  GRANT USAGE                          ON SEQUENCES TO app_crud;
ALTER DEFAULT PRIVILEGES FOR ROLE admin IN SCHEMA app
  GRANT USAGE                          ON TYPES     TO app_read;
ALTER DEFAULT PRIVILEGES FOR ROLE admin IN SCHEMA app
  GRANT USAGE                          ON TYPES     TO app_crud;

-- Assign group roles to runtime users.
GRANT app_crud TO app;
GRANT app_crud TO mcp;
GRANT app_read TO reporter;

-- Revoke old direct grants on app schema from app and mcp
-- (access is now inherited through group roles).
REVOKE SELECT, INSERT, UPDATE, DELETE ON ALL TABLES    IN SCHEMA app FROM app;
REVOKE USAGE                          ON ALL SEQUENCES IN SCHEMA app FROM app;
REVOKE USAGE ON TYPE app.lead_status   FROM app;
REVOKE USAGE ON TYPE app.org_role      FROM app;
REVOKE USAGE ON TYPE app.campaign_role FROM app;
REVOKE USAGE ON SCHEMA app FROM app;

REVOKE SELECT, INSERT, UPDATE, DELETE ON ALL TABLES    IN SCHEMA app FROM mcp;
REVOKE USAGE                          ON ALL SEQUENCES IN SCHEMA app FROM mcp;
REVOKE USAGE ON TYPE app.lead_status   FROM mcp;
REVOKE USAGE ON TYPE app.org_role      FROM mcp;
REVOKE USAGE ON TYPE app.campaign_role FROM mcp;
REVOKE USAGE ON SCHEMA app FROM mcp;

-- Clean up old per-user DEFAULT PRIVILEGES (replaced by group role defaults above).
ALTER DEFAULT PRIVILEGES FOR ROLE admin IN SCHEMA app
  REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES    FROM app;
ALTER DEFAULT PRIVILEGES FOR ROLE admin IN SCHEMA app
  REVOKE USAGE                          ON SEQUENCES FROM app;
ALTER DEFAULT PRIVILEGES FOR ROLE admin IN SCHEMA app
  REVOKE USAGE                          ON TYPES     FROM app;
ALTER DEFAULT PRIVILEGES FOR ROLE admin IN SCHEMA app
  REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES    FROM mcp;
ALTER DEFAULT PRIVILEGES FOR ROLE admin IN SCHEMA app
  REVOKE USAGE                          ON SEQUENCES FROM mcp;
ALTER DEFAULT PRIVILEGES FOR ROLE admin IN SCHEMA app
  REVOKE USAGE                          ON TYPES     FROM mcp;

-- +goose Down
-- Restore direct grants to runtime users.
GRANT USAGE ON SCHEMA app TO app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES    IN SCHEMA app TO app;
GRANT USAGE                          ON ALL SEQUENCES IN SCHEMA app TO app;
GRANT USAGE ON TYPE app.lead_status   TO app;
GRANT USAGE ON TYPE app.org_role      TO app;
GRANT USAGE ON TYPE app.campaign_role TO app;

GRANT USAGE ON SCHEMA app TO mcp;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES    IN SCHEMA app TO mcp;
GRANT USAGE                          ON ALL SEQUENCES IN SCHEMA app TO mcp;
GRANT USAGE ON TYPE app.lead_status   TO mcp;
GRANT USAGE ON TYPE app.org_role      TO mcp;
GRANT USAGE ON TYPE app.campaign_role TO mcp;

-- Restore per-user DEFAULT PRIVILEGES.
ALTER DEFAULT PRIVILEGES FOR ROLE admin IN SCHEMA app
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES    TO app;
ALTER DEFAULT PRIVILEGES FOR ROLE admin IN SCHEMA app
  GRANT USAGE                          ON SEQUENCES TO app;
ALTER DEFAULT PRIVILEGES FOR ROLE admin IN SCHEMA app
  GRANT USAGE                          ON TYPES     TO app;
ALTER DEFAULT PRIVILEGES FOR ROLE admin IN SCHEMA app
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES    TO mcp;
ALTER DEFAULT PRIVILEGES FOR ROLE admin IN SCHEMA app
  GRANT USAGE                          ON SEQUENCES TO mcp;
ALTER DEFAULT PRIVILEGES FOR ROLE admin IN SCHEMA app
  GRANT USAGE                          ON TYPES     TO mcp;

REVOKE app_read FROM reporter;
REVOKE app_crud FROM mcp;
REVOKE app_crud FROM app;

-- Remove group role grants and default privileges.
ALTER DEFAULT PRIVILEGES FOR ROLE admin IN SCHEMA app
  REVOKE SELECT                         ON TABLES    FROM app_read;
ALTER DEFAULT PRIVILEGES FOR ROLE admin IN SCHEMA app
  REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES    FROM app_crud;
ALTER DEFAULT PRIVILEGES FOR ROLE admin IN SCHEMA app
  REVOKE USAGE                          ON SEQUENCES FROM app_crud;
ALTER DEFAULT PRIVILEGES FOR ROLE admin IN SCHEMA app
  REVOKE USAGE                          ON TYPES     FROM app_read;
ALTER DEFAULT PRIVILEGES FOR ROLE admin IN SCHEMA app
  REVOKE USAGE                          ON TYPES     FROM app_crud;

REVOKE ALL ON ALL TABLES    IN SCHEMA app FROM app_read;
REVOKE ALL ON ALL TABLES    IN SCHEMA app FROM app_crud;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA app FROM app_crud;
REVOKE USAGE ON TYPE app.lead_status   FROM app_read;
REVOKE USAGE ON TYPE app.lead_status   FROM app_crud;
REVOKE USAGE ON TYPE app.org_role      FROM app_read;
REVOKE USAGE ON TYPE app.org_role      FROM app_crud;
REVOKE USAGE ON TYPE app.campaign_role FROM app_read;
REVOKE USAGE ON TYPE app.campaign_role FROM app_crud;
REVOKE USAGE ON SCHEMA app FROM app_read;
REVOKE USAGE ON SCHEMA app FROM app_crud;

DROP ROLE app_crud;
DROP ROLE app_read;
