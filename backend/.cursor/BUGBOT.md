# Bugbot — backend-specific rules

## Layering

- New HTTP endpoints: handler only wires Gin routes and calls service interfaces.
- No raw SQL or Ent client usage in handlers.
- Service layer must not reach into repository packages directly (depguard).

## Go quality

- Check error returns from `errcheck`-relevant calls.
- Prefer context propagation on I/O and external API calls.
- Integration tests use build tag `integration`; unit tests use `unit`.

## Migrations

- SQL migrations live in `backend/migrations/` with sequential numbering.
- Migrations must be idempotent where the repo pattern expects `IF NOT EXISTS`.
