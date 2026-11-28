# kipitfit

Design doc: `docs/design.md` — read before architectural changes. Roadmap in "Build order".

## Rules
- ONE process with one SQLite file: no Postgres/Redis/worker containers for its own state. No Node at runtime.
- Clean architecture, dependencies point inward: `internal/domain` (entities, ports, coded errors, rules several use
  cases share) has no framework imports; `internal/usecase/<area>` holds the logic, one `Service` per area (auth,
  exercises, library, routines, workouts, records); `internal/infrastructure` the adapters (SQLite, the S3 media
  bucket, bcrypt, the built-in exercise dataset, pack readers); `internal/delivery/http` the handlers. Wiring only in
  `cmd/kipitfit/main.go` (`wire()`). Handlers never touch SQL. depguard in `.golangci.yml` enforces it: never
  `//nolint` it, fix the dependency.
- Interfaces only where the code meets the outside (database, bucket, password hashing, files), declared as ports in
  `domain`. A use case that needs another takes its `*Service`: no interface for code with one implementation inside
  the app.
- Files stay under 500 lines (revive, oxlint); split by concern, not by size.
- Persistence only through the `domain` repositories, all implemented by `sqlite.Store`, which maps its own bun models
  (`models.go`) to the entities; keep SQL portable (SQLite now, maybe Postgres later). STRICT tables, TEXT ULID ids
  (`internal/ids`), times as UTC TEXT, goose migrations in `internal/infrastructure/sqlite/migrations` (never edit a
  released one, add the next).
- Weights are stored as integer grams (distance metres, durations seconds): no floats in the database. kg/lb is only
  a display choice (`users.unit`).
- User-facing errors are `domain.Error` values with a stable `code` (`domain.Coded(...)`, declared by the use case that
  returns them, or in `domain` when several do); the API adds its own in `apiCodes` (`internal/delivery/http/api.go`). Every code needs `errors.<code>` in both `web/src/locales/en.json` and `fr.json` — a Go
  test fails otherwise.
- Every UI string goes through i18next, in both catalogs. French plurals need `_one`, `_many` and `_other`; the vitest
  catalog test fails on a missing key or a variable mismatch.
- UI primitives come from `@kipit-labs/kipitidy`, imported per file (`@kipit-labs/kipitidy/button`). Kit first: a
  component any app could use goes into the kipitidy repo (change, release, bump here), not into `web/src`. App
  components stay in `web/src` (set table, rest timer, plate visual: fitness wording).
- Web pages and the logic only they use live in `web/src/features/<area>/` (the live workout's reducer and store in
  `features/live`); `web/src/lib` holds what several features share (API client, units, records, i18n helpers).
- Mobile-first PWA (phone at the gym, 390×844 first), no native wrapper. Theme via kipitidy's `theme.css`; the only
  override is the teal `--primary` (kipitfit's kipit-labs colour) in `web/src/index.css`.
- Sign-up is invite-only: an admin creates an invite, the link is the credential.
- Never commit third-party exercise pictures or pack data (RepDB, ExerciseDB…): their licences forbid republishing.
  Packs are imported per deployment (`kipitfit pack import`); tests build tiny synthetic packs.
- Pictures live in an S3-compatible bucket and are served only through `/api/media` (`packs/` keys, images/videos);
  the CSP stays `'self'`. Media off (no `KIPITFIT_S3_ENDPOINT`) must keep working with equipment icons.

## Testing
The tests are the safety net that lets agents change code: a change is done when `make cover`, `make e2e` and
`make lint` pass, run, not assumed. Five layers, each one required for what it covers:

| Layer | Where | Covers |
|---|---|---|
| Unit | `internal/usecase/*/*_test.go`, fakes in `fakes_test.go`; `internal/domain` | business rules, edge cases |
| Adapter integration | `internal/infrastructure/sqlite/*_test.go` (real SQLite), `infrastructure/pack`, `exercisedb`, `security` | SQL, files, packs, hashing |
| HTTP contract | `internal/delivery/http/*_test.go` (on the real SQLite store), `cmd/kipitfit/routes_test.go` (`routeAccess`) | status and error codes, cookies, who may call each route |
| API end to end | `cmd/kipitfit/journey*_test.go`: the real `wire()` over HTTP on SQLite, media in memory | user journeys across every layer |
| UI end to end | `web/e2e/*.e2e.ts` (Playwright): the real binary on a fresh database, WebKit at iPhone size and desktop Chromium | user journeys in a browser, phone layout, uncaught page errors |

- A new route goes into `routeAccess`; a new user-facing flow adds or extends a journey in both end-to-end suites
  (API, and UI for what users tap).
- A new page or tab is opened by `e2e/pages.e2e.ts`, with data to show: a render error shows the router's error page
  instead of throwing, so only that test sees it.
- UI tests find elements by role and label, as users and screen readers do, never by CSS class or test id: an element
  a test cannot find by name is an accessibility bug. They wait on state or responses, never on timeouts. Any
  uncaught page error fails the test (`e2e/fixtures.ts`). Each test joins with its own account through an invite;
  each run starts on a fresh database.
- Write the test first and watch it fail for the right reason: a test that never failed proves nothing. For a bug,
  the failing test is the reproduction.
- Never weaken, skip or delete a test, add `//nolint`, or lower `COVERAGE_FLOOR` to get green. If a test is wrong,
  fix it in its own commit and say why in the PR.
- A refactor keeps the end-to-end, route and handler tests unchanged: editing one means behavior changed, and the
  PR says so.
- Tests are deterministic: no sleeps, no real network, synthetic packs only.
- `COVERAGE_FLOOR` only goes up: raise it when coverage does.
- Frontend logic (formatting, units, records) gets vitest tests next to it.
- A new test runs repeated before it is trusted: `make e2e ARGS="--repeat-each 5"`. A flaky test is a bug, never
  retried away (`retries: 0`).

## Commands
`make dev-api` / `make dev-web` / `make test` / `make cover` / `make e2e` / `make lint` / `make build` / `make docker`.
`make e2e` runs Playwright in its Docker image (no browser to install); `ARGS=` passes options, e.g.
`ARGS="--project=iphone workout"`.
`go:embed` needs `web/dist`; make targets stub it via `ui-stub`. Run oxlint with `--threads 1` if it runs out of
memory.
