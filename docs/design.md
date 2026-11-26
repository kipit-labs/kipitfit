# kipitfit design

A self-hosted workout tracker in the spirit of Hevy: plan routines, log every set at the gym, see your records and
progress, and follow your friends' training. Built on the kipitiny pattern: one Go binary, SQLite, the React app
embedded.

| Topic | Decision |
|---|---|
| Backend | kipitiny pattern: Go, SQLite, bun, goose |
| Social | friends + feed, kudos, comments |
| Exercises | open dataset seed + custom |
| v1 extras | Hevy/Strong import, body measurements, supersets + set types, plate calc + 1RM |
| Languages | English + French from day one |
| UI | `@kipit-labs/kipitidy` |
| Deploy | homelab k3s, ArgoCD |
| Sign-up | invite-only |
| Units | kg and lb |

## Screens

Mobile first, installed as a PWA. Teal (#3dd6b0) is kipitfit's colour: its slot among the ten kipit-labs dots (the
kipit-labs site's `projects.ts`; coral is kipitrek's, yellow kipitiny's). Other dots each mark one kind of set, so
colour carries meaning on the set table: **W** warm-up (left out of volume and records), **D** drop set, **F** to
failure, **PR** personal record. Bottom tabs: Feed, Workout, Progress, Me.

- **Feed** — friends' finished workouts, newest first, paged like kipitrek's feed: title, time, volume, records, the
  first exercises with their set count, kudos and comments under each.
- **Workout (routines)** — "Start empty workout", then routines grouped in folders (Push · Pull · Legs). Each card
  lists its exercises and when it was last done. Starting one copies it into a live workout; the routine itself never
  changes unless you ask on finish.
- **Live workout** — the screen that matters. Header: elapsed time, volume, sets, Finish. Per exercise a set table
  (set type, previous, kg, reps, ✓) where "previous" is the same set from last time. Ticking a set starts the rest
  timer (−15 s / +15 s / skip) and checks for a record on the spot (e.g. "PR · e1RM 111.0 kg"). Supersets are grouped
  with one rest after both.
- **Plate calculator** — opens from any barbell weight cell: plates per side for the target and the bar, what's left
  over, a warm-up ramp from the working weight (20×10 · 60×5 · 90×3 · 115×2 · 130×1). Uses the plates you said you own
  (Settings).
- **Finish** — summary (time, volume, vs last), new records, title, notes, who can see it, and optionally write today's
  weights or set list back into the routine.
- **Exercise** — history, records (best e1RM, heaviest weight, best set volume), rep records, and a chart switchable
  between e1RM, heaviest weight, session volume and reps.
- **Progress** — workouts per week, training-days calendar, sets per muscle over 30 days, bodyweight trend.

## Features in v1, and what waits

### Log a workout
- Start empty or from a routine; add, reorder, swap or remove exercises mid-session.
- Set table with "previous" values, kg/lb, reps, optional RPE, notes per exercise.
- Set types: normal, warm-up, drop, failure; supersets of 2+ exercises.
- Exercise kinds: weight × reps, reps only, weighted bodyweight, assisted bodyweight, duration, distance + duration.
- Rest timer per exercise, adjustable live; push notification when it ends with the app in the background.
- Works with no signal: the live workout lives on the phone and syncs on finish.
- Edit or delete past workouts; records recompute.

### Plan
- Routines with target sets, reps or rep range, weight, rest.
- Folders to group routines.
- Save a finished workout as a new routine; on finish, update the routine's weights or set list.
- Duplicate a friend's routine from their workout.

### Exercises
- Library seeded from an open dataset (all 876 exercises of free-exercise-db, Unlicense): name, category, equipment,
  level, primary and secondary muscles, instructions. Names stay English for now; muscles, equipment, categories,
  kinds and levels are translated.
- Search (words in any order, case and accents ignored) and filters by muscle, equipment and category, run in the app
  over the whole list; custom exercises per user (name unique per user, case-insensitive). Library rows are read-only.
- Next: a `name_fr` column with reviewed French names, and a `kipitfit fetch-images` command that downloads the
  dataset's photos into the data dir once, by hand: the app itself never calls out. Until then a row shows its
  equipment icon.
- Per exercise: history, records, rep records 1–12, chart.

### Progress
- Records: best e1RM, heaviest weight, best set volume, best session volume, most reps; shown live while logging.
- Workouts per week, streak calendar, sets per muscle.
- Body measurements (bodyweight, body fat %, waist, chest, arms, thighs…) with charts.
- Tools: plate calculator from your plate set, e1RM calculator, warm-up ramp.

### Friends
- Friend requests, accepted both ways.
- Profile visibility: private, friends, public (per-workout override).
- Feed of friends' workouts, kudos, comments.
- Public link to a workout with an Open Graph card.

### Account and data
- Email + password, passkeys, sessions list (from kipitiny's auth).
- Invite-only: the first account is admin, then admins invite by link (emailed when SMTP is set, else copied).
- English and French, picked per account; emails go out in it.
- Import Hevy and Strong CSV with an exercise-matching step.
- Export everything as CSV (Hevy columns) and JSON.
- Delete account and all data.

### Later, not in v1
Programs (multi-week plans with progression rules), auto-progression suggestions, public follow and discovery,
leaderboards, Apple Health / Google Fit sync (needs a native wrapper), watch app, an MCP server so Claude can read your
training (kipitiny ships one to copy), coach/client accounts, a Postgres store behind the same interface.

## Architecture

Same layers and rules as kipitiny. The fitness domain lives in `internal/core`; handlers and the importer only call
core; core only talks to `store.Store`.

```
cmd/kipitfit/        main.go: config, open db, wire, serve
internal/
  config/            env → Config (KIPITFIT_*)
  ids/               ULIDs
  sqlitedb/          WAL, 1 conn, goose, pre-migrate snapshot
  store/             Store interface + entity structs
    sqlite/          bun implementation
      migrations/    00001_init.sql …
  core/              all logic: auth, workouts, routines, records, stats, social, measurements
  exercisedb/        embedded seed JSON + loader            (later)
  importer/          Hevy / Strong CSV parsers              (later)
  api/               net/http ServeMux, JSON, auth middleware
  mail/              verification + invite emails            (later)
  notify/            Web Push: rest timer, kudos, comments  (later)
web/                 React app, go:embed web/dist
  src/routes/        Feed, Workout, Live, Exercise, Progress, Me
  src/live/          live-workout store (persisted on device)
```

### Stack
- Go 1.26, `net/http` ServeMux patterns, `log/slog`.
- SQLite via `modernc.org/sqlite` (no CGO), bun queries, goose migrations.
- bcrypt passwords, hashed session tokens, per-IP fail limiter: lifted from kipitiny. Passkeys (`go-webauthn`) later.
- Web Push from kipitiny's `notify` package (later).
- React 19 + Vite + TypeScript, Tailwind v4, `@kipit-labs/kipitidy` imported per file.
- TanStack Query for server data, react-router, recharts (through the kit's chart) for charts.
- react-i18next with `en.json` + `fr.json` (French plurals need `_many`); API errors carry a `code`; tests fail on a
  missing key, as in kipitrek.
- PWA: manifest, later a service worker for the app shell and a wake lock during a workout.
- Image `ghcr.io/kipit-labs/kipitfit`, pinned in the homelab ops repo, synced by ArgoCD on k3s.
- k3s, set up like expense-tracker (PocketBase SQLite): 1 replica, `Recreate`, `/data` on a `longhorn` ReadWriteOnce
  PVC, secrets through proton-pass-sync.
- Backups: the `kipitfit` namespace goes into Velero's `daily-backup` `includedNamespaces`, so Longhorn snapshots reach
  R2 at 02:00 UTC and are kept 7 days. A snapshot of a running SQLite in WAL mode restores like after a power cut,
  which SQLite handles. kipitfit adds no backup code of its own.

## Data model

SQLite STRICT tables, TEXT ULID ids, UTC TEXT times. Weights are stored as integer grams, distance as metres, time as
seconds. No floats in the database, so 92.5 kg is always 92 500 and conversion to lb happens only on display.

| Table | Key columns | Notes |
|---|---|---|
| `users` | email, password_hash, name, unit (kg\|lb), visibility, language, is_admin, plates_g (JSON) | kipitiny's users + settings for units and plates |
| `sessions`, `invites` | token_hash, user_id / email, expires_at | in the skeleton |
| `passkeys`, `email_tokens` | as in kipitiny | later |
| `exercises` | owner_id (NULL = library), slug, name, kind, category, equipment, primary_muscles / secondary_muscles / instructions (JSON), force, level, mechanic | library rows upserted by slug on start when the embedded dataset changes (`meta.exercise_library_version`), never deleted; custom rows per user, archived (`archived_at`) instead of deleted while a routine or workout uses them |
| `routine_folders` | user_id, name, position | |
| `routines` | user_id, folder_id, name, notes, position | |
| `routine_exercises` | routine_id, exercise_id, position, superset, rest_s, notes | `superset` = group letter, NULL when alone |
| `routine_sets` | routine_exercise_id, position, type, weight_g, reps_min, reps_max, duration_s, distance_m | targets, all nullable |
| `workouts` | id (made on the phone), user_id, routine_id, title, notes, started_at, ended_at, visibility, volume_g, records | client ULID makes finish idempotent when the network drops |
| `workout_exercises` | workout_id, exercise_id, position, superset, notes | |
| `sets` | workout_exercise_id, position, type, weight_g, reps, rpe, duration_s, distance_m, e1rm_g | index on (user, exercise, workout start) via join for "previous" |
| `records` | user_id, exercise_id, kind, value, set_id, achieved_at | current best per kind; rebuilt for one exercise on edit/delete |
| `measurements` | user_id, kind, value, unit, measured_at | one row per value; kinds: weight, body_fat, waist, chest, arm_l, arm_r… |
| `friendships` | user_a, user_b, requested_by, accepted_at | one row per pair, like kipitrek |
| `kudos`, `comments` | workout_id, user_id, body | |
| `push_subscriptions` | user_id, endpoint, keys | from kipitiny's Web Push |
| `imports` | user_id, source, status, counts, mapping (JSON) | remembers your exercise-name mapping for the next import |

## HTTP API

JSON, session cookie. The endpoints built so far are in the README (routines: one `PUT` replaces the whole routine,
simplest for a reorderable editor; `PUT /api/routines/order` reorders folders and routines); planned ones:

| Method | Path | Does |
|---|---|---|
| GET | `/api/exercises` | library + your custom ones, compact; the app searches and filters (876 rows, ~190 kB) |
| POST PATCH DELETE | `/api/exercises/{id}` | custom exercises only |
| GET | `/api/exercises/{id}/history` · `/records` · `/series?metric=` | per-exercise sets, records, chart points |
| GET | `/api/previous?exercise=a,b,c` | last logged sets per exercise, fetched when a workout starts |
| PUT | `/api/workouts/{id}` | save a finished workout by its phone-made id; returns new records. Same call edits it later |
| GET | `/api/workouts?after=` · `/api/workouts/{id}` | your history, paged; one workout |
| DELETE | `/api/workouts/{id}` | records of its exercises rebuilt |
| GET | `/api/stats?from=&to=` | per-week counts, volume, sets per muscle, training days |
| GET POST DELETE | `/api/measurements` | body measurements |
| GET | `/api/feed?after=` | friends' visible workouts, 20 a page |
| PUT DELETE | `/api/friends/{id}` · `/api/workouts/{id}/kudos` | request/accept, unfriend; kudos |
| GET POST DELETE | `/api/workouts/{id}/comments` | |
| POST | `/api/import` (multipart) → `/api/import/{id}/confirm` | parse and propose exercise matches; then write |
| GET | `/api/export.csv` · `/api/export.json` | all your data |
| POST | `/api/push/subscribe` · `/api/push/rest` | Web Push; schedule a "rest over" push at a time |

## Rules of the domain

### Live workout on the phone
Gyms have bad signal. The live workout is a client store persisted to IndexedDB after every change, never a server
row. Starting fetches routine + "previous" once; finishing sends one `PUT /api/workouts/{ulid}`, retried until it
lands. A reload or a dead battery resumes where you were. One live workout per device.

### Rest timer
Stored as `endsAt`, not a countdown, so it survives the screen locking. In the foreground the app beeps and vibrates;
for the background the app asks the server to send a Web Push at `endsAt` (an in-memory timer in core, dropped on
skip). iOS sends pushes only to a PWA added to the home screen.

### Records
e1RM uses Epley, `w × (1 + reps / 30)`, for 1–12 reps; above 12 reps the set counts for rep records only. Warm-up sets
count for nothing. Core computes records when a workout is saved, compares to `records`, and returns what was beaten.
The phone runs the same formula live from the bests it fetched at start; a Go test and a TS test share one fixture
file so both agree.

### Volume and units
Volume = Σ weight × reps over non-warm-up sets. Weighted bodyweight adds the latest bodyweight measurement; assisted
subtracts it. Everything is grams on the server. kg and lb both ship in v1, picked per account: lb shows rounded to
0.5 lb, typing 225 lb stores 102 058 g, and plate sets and bar weight exist in both units (45 lb bar,
45/35/25/10/5/2.5 lb plates).

### Import Hevy / Strong
Hevy CSV: `title, start_time, end_time, exercise_title, superset_id, set_index, set_type, weight_kg, reps,
distance_km, duration_seconds, rpe`. Strong CSV: `Date, Workout Name, Exercise Name, Set Order, Weight, Reps, Seconds,
RPE`. Names are matched to the library (exact, then normalised, then closest); you confirm or pick, unmatched ones
become custom exercises. Re-importing skips workouts already there (same start time).

### Exercise dataset
`yuhonas/free-exercise-db`: about 800 exercises with muscles, equipment, instructions and photos, released as public
domain (check before shipping). The JSON is embedded and upserted by slug on start, so updates land with a release.
Photos are too heavy to embed (tens of MB): a `kipitfit fetch-images` command downloads them into the data volume, and
the app shows a muscle icon without them.

## Build order

One PR stack per phase.

1. **kipitidy 0.2.0** — shared components moved into the kit and released. Done.
2. **Skeleton from kipitiny** — config, ids, sqlitedb, store pattern, auth (sessions, fail limiter), api middleware
   and the embed. Invite-only accounts and admin invites. React shell with kipitidy, i18n en/fr with the missing-key
   tests, login, tab bar. Docker image, CI, release workflow. *Done locally, minus passkeys and mail; still to do: the
   ops repo (namespace, Deployment, Service, Ingress, longhorn PVC, ProtonPassSecret, ArgoCD Application, the namespace
   added to the Velero schedule).*
3. **Exercise library** — seed loader, search and filters, custom exercises, exercise page without stats. *Done.*
4. **Routines and the live workout** — folders, routine editor, live workout store on the device, set types,
   supersets, rest timer in the foreground, finish and save. This is the core: it gets the most testing on a real
   iPhone.
5. **History, records, progress** — workout history and edit, records with live detection, exercise charts, weekly
   stats, calendar, sets per muscle.
6. **Friends and feed** — friend requests, visibility, feed, kudos, comments, public workout link with OG card.
7. **Import, measurements, tools** — Hevy and Strong CSV import with matching, export, body measurements, plate
   calculator, warm-up ramp, e1RM calculator.
8. **PWA polish** — Web Push for rest timer, kudos and comments; wake lock; offline app shell; install prompt.

## UI kit

kipitfit starts on `@kipit-labs/kipitidy` 0.2.0, which took in kipitrek's chart, avatar, popover, command, slider,
toggle and input-group, and kipitiny's EmptyState, StatCard, ConfirmDialog, Loading, Section, CopyField and friends.
Components only kipitfit needs stay in the app because of their fitness wording: set table row, rest timer bar, plate
visual, superset bracket, set-type badge.
