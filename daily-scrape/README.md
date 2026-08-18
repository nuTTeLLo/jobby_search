# Daily LinkedIn scrape

Scrapes LinkedIn each morning for newly posted roles and feeds them to the tracker's
**Discovered** page (`/discovered`). It never touches the `jobs` table — application status stays on
LinkedIn, and this is purely a reading surface.

Runs as a k3s CronJob at **09:00 Australia/Melbourne**
(`k8s-services/job-tracker/09-cronjob-linkedin-scrape.yaml`). The CronJob's `timeZone`
handles AEST/AEDT, so nothing in the script deals with daylight saving.

## How it works

1. Queries LinkedIn's public guest endpoint
   `/jobs-guest/jobs/api/seeMoreJobPostings/search` with `f_TPR=r86400` (last 24h) for
   each term in `config.json`, paging until a page comes back empty. No login needed.
2. Filters card titles through `title_include` / `title_exclude`, and re-checks the
   posted date as a backstop.
3. Classifies each survivor by fetching `/jobs-guest/jobs/api/jobPosting/<id>`:
   `apply-link-onsite` → `easy_apply`, `apply-link-offsite` → `external`, neither →
   `unknown`.
4. POSTs the batch to `POST /api/discovered-jobs` with a short-lived HS256 token
   minted from `JWT_SECRET`.

The **tracker owns de-duplication and retention**: rows upsert on
`(user_id, external_id)` and anything older than 7 days is pruned on each ingest. The
script therefore keeps no local state, and re-running it is harmless. The
`applied_before` flag is computed server-side by matching the company against your
tracked jobs.

## Configuration

`config.json` holds the search — terms, location, `hours_old`, `max_pages_per_term`,
and the title regexes — and is baked into the image. Deployment settings come from
the environment and override the file:

| Env var | Purpose |
|---|---|
| `TRACKER_URL` | Tracker base URL (in-cluster: `http://job-tracker-backend:8080`) |
| `TRACKER_USER_ID`, `TRACKER_EMAIL` | Claims for the minted token |
| `JWT_SECRET` | Signing secret, from the `job-tracker-secret` secret |
| `SEARCH_LOCATION`, `HOURS_OLD` | Occasional overrides without rebuilding |

## Running locally

Against a dev backend (see `mise run backend`, which listens on :8081):

```bash
TRACKER_URL=http://localhost:8081 \
TRACKER_USER_ID=<your user id> \
JWT_SECRET=<dev secret> \
python3 linkedin_daily_scrape.py --dry-run
```

`--dry-run` scrapes and prints the digest without posting. Other flags: `--hours N`,
`--config PATH`, `--verbose` (per-page counts on stderr).

Only dependency is PyJWT; everything else is stdlib.

## Throttling

LinkedIn soft-throttles by serving an empty HTTP 200 page rather than an error. The
script retries an empty first page twice with backoff and, if a term still yields
nothing, prints a warning naming that term — so an under-reported run is visible in
`kubectl logs` instead of silently looking like a quiet day. Running the scrape
several times in quick succession reliably triggers this.
