-- +goose Up
-- @auth/pg-adapter does not insert `id` on createUser/createSession/createAccount;
-- it expects the DB to generate a UUID default.

ALTER TABLE nextauth.users ALTER COLUMN id SET DEFAULT gen_random_uuid()::TEXT;
ALTER TABLE nextauth.accounts ALTER COLUMN id SET DEFAULT gen_random_uuid()::TEXT;
ALTER TABLE nextauth.sessions ALTER COLUMN id SET DEFAULT gen_random_uuid()::TEXT;

-- +goose Down
ALTER TABLE nextauth.users ALTER COLUMN id DROP DEFAULT;
ALTER TABLE nextauth.accounts ALTER COLUMN id DROP DEFAULT;
ALTER TABLE nextauth.sessions ALTER COLUMN id DROP DEFAULT;
