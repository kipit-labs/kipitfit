# kipitfit

Design doc: `docs/design.md` — read before architectural changes. Roadmap in "Build order".

## Rules
- ONE process with one SQLite file: no Postgres/Redis/worker containers for its own state. No Node at runtime.
- Layers: `cmd/kipitfit` only wires; `internal/api` decodes/encodes and calls `internal/core`; all logic is in core.
  Handlers never touch SQL.
- Store access only via `store.Store`; keep SQL portable (SQLite now, maybe Postgres later). STRICT tables, TEXT ULID
  ids (`internal/ids`), times as UTC TEXT, goose migrations in `internal/store/sqlite/migrations` (never edit a
  released one, add the next).
- Weights are stored as integer grams (distance metres, durations seconds): no floats in the database. kg/lb is only
  a display choice (`users.unit`).
- User-facing errors are `core.Error` values with a stable `code` (`coded(...)` in `internal/core/errors.go`); the API
  adds its own in `apiCodes`. Every code needs `errors.<code>` in both `web/src/locales/en.json` and `fr.json` — a Go
  test fails otherwise.
- Every UI string goes through i18next, in both catalogs. French plurals need `_one`, `_many` and `_other`; the vitest
  catalog test fails on a missing key or a variable mismatch.
- UI primitives come from `@kipit-labs/kipitidy`, imported per file (`@kipit-labs/kipitidy/button`). Kit first: a
  component any app could use goes into the kipitidy repo (change, release, bump here), not into `web/src`. App
  components stay in `web/src` (set table, rest timer, plate visual: fitness wording).
- Mobile-first PWA (phone at the gym, 390×844 first), no native wrapper. Theme via kipitidy's `theme.css`; the only
  override is the teal `--primary` (kipitfit's kipit-labs colour) in `web/src/index.css`.
- Sign-up is invite-only: an admin creates an invite, the link is the credential.

## Commands
`make dev-api` / `make dev-web` / `make test` / `make lint` / `make build` / `make docker`.
`go:embed` needs `web/dist`; make targets stub it via `ui-stub`. Run oxlint with `--threads 1` if it runs out of
memory.
