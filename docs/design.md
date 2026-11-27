# kipitfit design

A self-hosted workout tracker in the spirit of Hevy: plan routines, log every set at the gym, see your records and
progress, and follow your friends' training. Built on the kipitiny pattern: one Go binary, SQLite, the React app
embedded.

| Topic | Decision |
|---|---|
| Backend | kipitiny pattern: Go, SQLite, bun, goose |
| Social | friends + feed, kudos, comments |
| Exercises | open dataset seed, or an imported pack with pictures; + custom |
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
- Next: a `name_fr` column with reviewed French names.
- Pictures come from an exercise pack (see "Exercise media"); without one, a row shows its equipment icon.
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

### Exercise media
- No pictures ship with kipitfit: the ones worth having (faceless illustrations or animations, the target muscle
  highlighted) are licensed, and their licences forbid republishing them. free-exercise-db's photos show real faces
  and trace back to Bodybuilding.com, so they aren't used. Each deployment imports its own **pack**: a vendor's ZIP with
  an exercise list and its pictures.
- `kipitfit pack import <pack> [ZIP or URL] [--sha256 HEX]`, a subcommand of the same binary (so it runs in the
  distroless image, e.g. `kubectl exec deploy/kipitfit -- /kipitfit pack import repdb-free`). It uploads the pictures
  to an S3-compatible bucket (RustFS, MinIO, R2…), skipping keys already there, so a re-run resumes; then, in one
  transaction, it makes the pack's exercises the library. Adapters: `repdb-free` (RepDB's free tier, downloaded from
  repdb.co, 708 exercises, flat start/peak stills, credit "Exercise data by RepDB (repdb.co)" shown on Me → About).
  Paid RepDB and ExerciseDB adapters come from their preview bundles; slugs are `<vendor>:<id>`, so moving from the
  free to the paid RepDB keeps every exercise's id.
- **The pack replaces the library.** A pack's names, muscles and instructions describe its own pictures; matching
  them to free-exercise-db by hand would be guesswork. free-exercise-db stays the built-in library of deployments
  without a pack; once a pack is imported, the boot sync no longer brings it back. Routines and workouts on a built-in
  exercise move to the pack exercise with the same name (case, spaces and punctuation ignored) **and the same kind**,
  so targets and logged sets still fit. Every old library exercise is archived, never deleted: hidden from the list,
  still shown where it's used, and still valid for a workout that was under way on a phone during the import. Custom
  exercises are untouched.
- The pack's muscles, equipment and categories map onto kipitfit's keys (RepDB's 30 muscles onto 18, adding
  `hip_flexors`; its ~70 pieces of equipment onto 11: bars and rings you hang from count as body weight, every
  `*_machine` as a machine). The kind is guessed with the same rules as the built-in library.
- Keys are `packs/<pack>/<version>/<file>`: a new pack version gets new keys, so a key's content never changes.
  `GET /api/media/{key}` streams from the private bucket behind the session (licensed, in-app use), with the object's
  ETag and `Cache-Control: private, max-age=31536000, immutable`; only `packs/` keys, images and videos are served (any
  other type goes out as `application/octet-stream`), so the CSP stays `'self'`. Without `KIPITFIT_S3_ENDPOINT`,
  media is off and everything shows equipment icons; a picture that fails to load falls back the same way.
- UI: a 40 px thumbnail (the first frame) in the library, the picker and on live-workout cards (tapping it opens the
  exercise); on the exercise page the frames loop every 900 ms with a pause button, one still frame under
  `prefers-reduced-motion`.

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
  exercisedb/        embedded seed JSON + loader
  media/             S3 bucket for pack pictures
  pack/              exercise pack readers (repdb-free)
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
- Exercise media in an S3-compatible bucket through `minio-go` (path-style URLs); the homelab's will be the
  cluster-wide RustFS.
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
| `workouts` | id (made on the phone), user_id, routine_id, title, notes, started_at, ended_at, visibility, volume_g | client ULID makes finish idempotent when the network drops; visibility comes with P6 |
| `workout_exercises` | workout_id, exercise_id, position, superset, notes | |
| `sets` | workout_exercise_id, position, type, weight_g, reps, rpe, duration_s, distance_m | rpe stored ×10 (8.5 → 85); "previous" joins workouts by exercise and start; records are worked out from the sets, not stored (see Records) |
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
| GET | `/api/exercises/{id}/progress` | built: records, rep records, and each workout with it (sets, chart values, records beaten) |
| GET | `/api/bests?exercise=` | built: current bests, for live detection |
| PUT GET | `/api/workouts/{id}` | built; answers carry the `records` the workout beat |
| GET | `/api/stats?from=` | built: each workout since `from` (at most 400 days back) in figures — times, volume, sets, working sets per primary muscle; the phone buckets them into its own weeks and days, so time zones stay on the phone |
| GET POST DELETE | `/api/measurements` | body measurements |
| GET | `/api/feed?after=` | friends' visible workouts, 20 a page |
| PUT DELETE | `/api/friends/{id}` · `/api/workouts/{id}/kudos` | request/accept, unfriend; kudos |
| GET POST DELETE | `/api/workouts/{id}/comments` | |
| POST | `/api/import` (multipart) → `/api/import/{id}/confirm` | parse and propose exercise matches; then write |
| GET | `/api/export.csv` · `/api/export.json` | all your data |
| POST | `/api/push/subscribe` · `/api/push/rest` | Web Push; schedule a "rest over" push at a time |

## Rules of the domain

### Live workout on the phone
Gyms have bad signal. The live workout is a client store (`web/src/live`, a pure reducer) saved as JSON to
localStorage after every change, never a server row: a few kB, so IndexedDB isn't worth its API. Starting copies the
routine into it and fetches "previous" once. Finishing queues one `PUT /api/workouts/{ulid}` (plus the routine's
update, if picked) in a localStorage outbox, sent at once and retried with backoff (2 s doubling to a minute) until a
2xx; the outbox is sent again when the app opens and when the signal comes back, and the Workout tab shows Saving /
Will retry. A refusal that retrying won't fix (a 4xx) stays queued with its error until the user retries or discards
it. A reload or a dead battery resumes where you were. One live workout per device: starting another asks to resume or
discard it.

### Rest timer
Stored as `endsAt`, not a countdown, so it survives the screen locking. In the foreground the app beeps and vibrates;
for the background the app asks the server to send a Web Push at `endsAt` (an in-memory timer in core, dropped on
skip). iOS sends pushes only to a PWA added to the home screen.

### Records
e1RM uses Epley, `w × (1 + reps / 30)` in grams rounded half up, for 1–12 reps; above 12 reps the set counts for rep
records only. Warm-up sets count for nothing; drop and failure sets count. What an exercise keeps depends on its kind:

| Kind | Records |
|---|---|
| weight × reps | e1RM, heaviest weight, best set volume (w × r), best workout volume, most reps in a set |
| weighted bodyweight | heaviest added weight, best set volume, best workout volume, most reps (no e1RM until bodyweight is known, P7) |
| reps only, assisted bodyweight | most reps in a set, most reps in a workout (less assistance is progress too, but doesn't add up with reps) |
| duration | longest set |
| distance + duration | longest distance, longest set |

Rep records (weight kinds): the heaviest weight lifted for **at least** n reps, n = 1–12, so 100 kg × 5 also counts
for 1 to 4 reps and the table has no holes.

A workout beats a record when its best for a kind is strictly above every earlier workout's (by start time); a tie
isn't a record, and the first workout with a value sets none (else every first workout would be all records). Within
a workout only its best set holds the record, the earliest on a tie.

Records are **worked out from the sets when read**, not stored: core replays an exercise's sets in order
(`ExerciseSets`, one query). So editing or deleting a workout, even an old one, re-ranks every later workout with
nothing to rebuild, and workouts saved before P5 have their records with no backfill. Replaying a few thousand sets
takes milliseconds; a records table is the upgrade if it ever shows. `PUT /api/workouts/{id}` answers with the records
the workout beat; the phone runs the same rules live from the bests it fetched at start, and a Go test and a TS test
share one fixture file (`internal/core/testdata/records.json`) so both agree.

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
   tests, login, tab bar. Docker image, CI, release workflow. *Done, minus passkeys and mail. 0.2.0 runs on the
   homelab (ops repo: namespace, Deployment, Service, Ingress, longhorn PVC, ArgoCD Application, the namespace in the
   Velero schedule).*
3. **Exercise library** — seed loader, search and filters, custom exercises, exercise page without stats. *Done.*
4. **Routines and the live workout** — folders, routine editor, live workout store on the device, set types,
   supersets, rest timer in the foreground, finish and save. This is the core: it gets the most testing on a real
   iPhone. *Built; still to test on an iPhone. Records and e1RM wait for P5; with no service worker yet (P8) the app
   itself needs signal to open, though a running workout and its finish don't.*

   **Then exercise media**, before P5 — media bucket and `/api/media`, `pack import` with the `repdb-free` adapter
   (the pack replaces the library), pictures in the list, picker, live workout and exercise page. *Built; the homelab
   bucket waits for the cluster RustFS.*
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
