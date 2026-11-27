---
name: feature
description: Build a kipitfit feature end to end — explore, plan and ask, implement, test for real in the app, report coverage in the PR. Use when the user asks for a new feature or invokes /feature <description>.
---

# Feature

Four phases, in order. Never skip the questions or the real-usage tests.

## 1. Explore and plan

- Read `CLAUDE.md`, the part of `docs/design.md` the feature belongs to (its "Build order" phase, data model,
  endpoints, domain rules) and the code it touches: migrations, `store`, `core`, `api`, the web route, both locale
  catalogs.
- Write a plan: what changes where, the PR split (multi-step work = a `gh stack`, one `feat/...` branch per step),
  and the real-usage test list for phase 3.
- **Ask before implementing.** List every open question, ambiguity and contradiction (with CLAUDE.md rules,
  docs/design.md, existing behavior, or the request itself) with AskUserQuestion, wait for the answers, then start. A
  plan with zero questions is suspicious: say why nothing is unclear.

## 2. Implement

Follow CLAUDE.md: a new goose migration (never edit a released one), store methods with tests on a temp SQLite, logic
and its tests in `core`, thin handlers in `api`, coded errors translated in both catalogs, every UI string in both
locales. Update `README.md` (API table, configuration) and `docs/design.md` when behavior or the plan changes.
`make cover`, `make test`, `make e2e` and `make lint` green before phase 3.

## 3. Test in real usage

- `make ui && make dev-api` (or `make dev-api` + `make dev-web`), with `KIPITFIT_DATA_DIR` on a temp dir so no real
  data is touched. Create the admin from the logged setup token, invite a second account for the friend's view.
- Drive it through the API (curl with a cookie jar) and a real browser (playwright-core with the cached Chromium),
  at phone width (390×844) first, in light and dark, in English and French.

For each case record what you did, expected and actual. Failures are fixed and retested, not just reported.

## 4. Report

In the PR description (the last PR of a stack), under `## Testing`:

```markdown
## Testing

`make cover` / `make test` / `make e2e` / `make lint`: green.

| # | Case | How | Expected | Result |
|---|------|-----|----------|--------|
| 1 | Log a set in lb | PUT /api/workouts/{id} with 225 lb | stored as 102 058 g | ✅ |

Not covered: <case> — <why>.
```

✅ works, ❌ broken, ⚠️ works with a caveat. In the final message, repeat only the counts, the ⚠️/❌ rows and the PR link.
