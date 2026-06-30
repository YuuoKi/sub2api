-- Value-Output-C dashboard verification — extra seed (throwaway, plaintext).
-- Runs AFTER the base M1B seed.sql (which creates the gateway user/group/account/api_key).
-- Adds: an admin user (so GetFirstAdmin works), an admin API key in settings (for x-api-key auth),
-- and extra users/groups so the dashboard's distinct-counts and joins have variety.

-- Admin user (role=admin, active) — required by validateAdminAPIKey -> GetFirstAdmin.
INSERT INTO users (email, password_hash, role, balance, status, username)
VALUES ('admin@local.test', 'x', 'admin', 0, 'active', 'admin-user');

-- Admin API key stored in settings; we send it via the x-api-key header.
INSERT INTO settings (key, value) VALUES ('admin_api_key', 'admin-localtest-key')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;

-- Extra teams (distinct group names) for the sample wall.
INSERT INTO groups (name, platform, status) VALUES ('product-team', 'anthropic', 'active');
INSERT INTO groups (name, platform, status) VALUES ('research-team', 'anthropic', 'active');

-- Extra employees (distinct usernames) for the sample wall.
INSERT INTO users (email, password_hash, role, balance, status, username)
VALUES ('alice@local.test', 'x', 'user', 10, 'active', 'alice');
INSERT INTO users (email, password_hash, role, balance, status, username)
VALUES ('bob@local.test', 'x', 'user', 10, 'active', 'bob');
INSERT INTO users (email, password_hash, role, balance, status, username)
VALUES ('carol@local.test', 'x', 'user', 10, 'active', 'carol');
