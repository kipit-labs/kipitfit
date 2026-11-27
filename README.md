<p align="center"><img src="web/public/icon.svg" alt="" width="96"></p>

<h1 align="center">kipitfit</h1>

<p align="center">
  <b>Self-hosted workout tracker.</b><br>
  Plan routines, log every set at the gym, see your records and progress, and follow your friends' training.
</p>

<p align="center">
  <a href="#quick-start">Quick start</a> ·
  <a href="docs/design.md">Design</a>
</p>

One Go binary with SQLite and the React app embedded, installed on the phone as a PWA, in the spirit of Hevy.
Invite-only, English and French, kg and lb.

**Status: skeleton.** Accounts work (first admin, sign-in, invites, profile). Exercises, routines, workouts,
records, friends and import come next; see [docs/design.md](docs/design.md) for the plan.

## Quick start

```sh
docker compose up --build
docker compose logs kipitfit | grep setup_token
```

Open http://localhost:8080, paste the setup token and create the admin account. Invite people from **Me → Invites**:
each invite is a link that works once, for 7 days.

For development, run `make dev-api` and `make dev-web` side by side, then open http://localhost:5173 (Vite proxies
`/api` to the Go server on :8080). The setup token is in the `make dev-api` output.

## Configuration

Environment variables, all optional:

| Variable | Default | What |
|---|---|---|
| `KIPITFIT_ADDR` | `:8080` | Listen address for the app and API. |
| `KIPITFIT_DATA_DIR` | `/data` | Holds `kipitfit.db`. Local disk only, never NFS/SMB. |
| `KIPITFIT_LOG_LEVEL` | `info` | `debug`, `info`, `warn` or `error`. |
| `KIPITFIT_APP_URL` | `http://localhost:<port>` | Public URL, used in invite links. |
| `KIPITFIT_COOKIE_SECURE` | `false` | `true` behind HTTPS: Secure session cookie and HSTS. |
| `KIPITFIT_SETUP_TOKEN` | random | Token required to create the first admin, instead of the one printed in the logs. |
| `KIPITFIT_REAL_IP_HEADER` | none | Header a reverse proxy puts the client IP in (`CF-Connecting-IP`, `X-Forwarded-For`). Needs `KIPITFIT_TRUSTED_PROXIES`. |
| `KIPITFIT_TRUSTED_PROXIES` | none | Comma-separated CIDRs or IPs allowed to set that header, e.g. the cluster's pod network. |

## API

JSON over HTTP with a session cookie (`kipitfit_session`, 30 days). Mutations from another origin are refused.
Errors answer `{"error": "...", "code": "...", "params": {...}}`; the app shows `errors.<code>` from its catalogs.

| Method | Path | Auth | Does |
|---|---|---|---|
| GET | `/api/health` | none | 204 when up. |
| GET | `/api/auth/state` | none | `{setupRequired, user?}`. |
| POST | `/api/auth/setup` | setup token | `{setupToken, email, name, password}`: creates the first admin and signs in. |
| POST | `/api/auth/login` | none | `{email, password}`: signs in. 10 failures per IP per 15 minutes, then 429. |
| POST | `/api/auth/logout` | none | Ends the session. |
| POST | `/api/invites/accept` | invite token | `{token, name, password}`: creates the invited account and signs in. |
| GET | `/api/me` | session | The signed-in user. |
| PATCH | `/api/me` | session | `{name?, language?, unit?}`: language `""`, `en` or `fr`; unit `kg` or `lb`. |
| GET | `/api/admin/invites` | admin | Pending invites, newest first. |
| POST | `/api/admin/invites` | admin | `{email}`: returns the invite with its `link`, shown only once. |
| DELETE | `/api/admin/invites/{id}` | admin | Revokes an invite. |
| GET | `/api/exercises` | session | The library and your own exercises, by name, without instructions. Search and filters run in the app. |
| GET | `/api/exercises/{id}` | session | One exercise with its instructions. Another user's exercise is a 404. |
| POST | `/api/exercises` | session | `{name, kind, category?, equipment?, primaryMuscles, secondaryMuscles?}`: creates your own exercise. |
| PATCH | `/api/exercises/{id}` | session | Same fields, all optional. Library exercises answer 403 `library_exercise_readonly`. |
| DELETE | `/api/exercises/{id}` | session | Deletes your own exercise; one a routine or workout uses is archived instead: hidden from the list, still in routines, its GET says `archived: true`. |
| GET | `/api/routines` | session | `{folders, routines}` in order; each routine has its `exercises` (with the `exercise` itself, archived too), target `sets` and `lastDoneAt`. |
| POST | `/api/routines` | session | `{name, notes?, folderId?, exercises: [{exerciseId, superset?, restS?, notes?, sets: [{type, weightG?, repsMin?, repsMax?, durationS?, distanceM?}]}]}`: added after your others. At most 50 exercises and 50 sets each. |
| PUT | `/api/routines/{id}` | session | Same body: replaces the whole routine. Another user's routine is a 404. |
| DELETE | `/api/routines/{id}` | session | Deletes a routine. |
| PUT | `/api/routines/order` | session | `{folders?: [id], routines?: [id]}`: each listed folder or routine takes its index as position. |
| GET | `/api/folders` | session | Your folders, in order. |
| POST | `/api/folders` | session | `{name}`: added after your others. |
| PATCH | `/api/folders/{id}` | session | `{name}`: renames. |
| DELETE | `/api/folders/{id}` | session | Deletes a folder; its routines stay, in no folder. |
| PUT | `/api/workouts/{id}` | session | `{title, notes?, routineId?, startedAt, endedAt, exercises: [{exerciseId, superset?, notes?, sets: [{type, weightG?, reps?, rpe?, durationS?, distanceM?}]}]}` under the ULID the phone made at start: creates the workout, or replaces it, so a retried finish saves once. Values follow the exercise kind; `rpe` 6–10 by 0.5. Returns it with `volumeG` (weight × reps of the non-warm-up sets). Another user's id is a 404. |
| GET | `/api/workouts?after=` | session | `{workouts, next?}`: 20 workouts without their exercises, newest first; pass `next` as `after` for the next page. |
| GET | `/api/workouts/{id}` | session | One workout with its exercises and sets. |
| DELETE | `/api/workouts/{id}` | session | Deletes a workout. |
| GET | `/api/previous?exercise=a,b,c` | session | `{exerciseId: sets}`: each exercise's sets in your latest workout that has it (at most 50 exercises). |

Passwords: 10 characters minimum, 72 bytes maximum (bcrypt).

## Development

`make test`, `make lint`, `make build` (binary in `bin/`), `make docker`. Rules for contributors (and Claude) are in
[CLAUDE.md](CLAUDE.md).

## License

MIT. The exercise library comes from [free-exercise-db](https://github.com/yuhonas/free-exercise-db) by yuhonas
(Unlicense); see [internal/exercisedb/NOTICE.md](internal/exercisedb/NOTICE.md).
