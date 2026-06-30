-- M1-B BC verification seed (throwaway, plaintext). Single apikey account -> mock@9099.
-- Reused identically for golden/after/flagoff (schema reset between runs).
INSERT INTO users (email, password_hash, role, balance, status, username)
VALUES ('bc@local.test', 'x', 'user', 100, 'active', 'bc-user');

INSERT INTO groups (name, platform, status)
VALUES ('anthropic', 'anthropic', 'active');

INSERT INTO accounts (name, platform, type, credentials, status, schedulable)
VALUES ('mock-anthropic', 'anthropic', 'apikey',
        '{"base_url":"http://127.0.0.1:9099","api_key":"sk-up-mock"}', 'active', true);

INSERT INTO account_groups (account_id, group_id)
SELECT a.id, g.id FROM accounts a, groups g
WHERE a.name='mock-anthropic' AND g.name='anthropic';

INSERT INTO api_keys (user_id, group_id, key, name, status)
SELECT u.id, g.id, 'sk-smoke-localtest-0001', 'bc-key', 'active'
FROM users u, groups g
WHERE u.email='bc@local.test' AND g.name='anthropic';
