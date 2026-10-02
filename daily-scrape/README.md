# Daily board scrape

Scrapes LinkedIn, Seek and Indeed each morning for newly posted roles and feeds them to
the tracker's **Discovered** page (`/discovered`). It never touches the `jobs` table —
application status stays on the boards, and this is purely a reading surface.

Runs as a k3s CronJob at **09:00 Australia/Melbourne**
(`k8s-services/job-tracker/09-cronjob-linkedin-scrape.yaml` in the `raspi` repo). The
CronJob's `timeZone` handles AEST/AEDT, so nothing in the script deals with daylight
saving.

## How it works

1. For each board and each term in `config.json`, runs a JobSpy search for postings
   from the last `hours_old` hours — one board at a time, so a board that errors or
   throttles only loses its own results. JobSpy runs in-process, from the
   `jobspy-mcp-server` fork (which adds Seek and `apply_type`), not through the MCP
   server: long searches inside the MCP pod stalled it until Kubernetes restarted it,
   which also took down the app's own job search.
   Seek is asked for two days, because its one-day filter means "listed today"; the
   date check in step 2 trims it back.
2. Filters titles through `title_include` / `title_exclude`, re-checks the posted date
   as a backstop (Indeed ignores its own date filter), and drops repeats of the same
   posting across terms.
3. Classifies each survivor as `easy_apply` or `external`:
   - **LinkedIn:** the guest posting page `/jobs-guest/jobs/api/jobPosting/<id>` —
     `apply-link-onsite` → `easy_apply`, `apply-link-offsite` → `external`.
   - **Seek:** the `isLinkOut` field of the `jobDetails` GraphQL query — `false` is
     Quick apply, `true` hands off to the employer.
   - **Indeed:** already in the search result (JobSpy's `apply_type`, from the fork).

   Anything that cannot be determined is `unknown`. This runs only for postings that
   survived step 2, which keeps the per-posting requests to LinkedIn and Seek small.
4. POSTs the batch to `POST /api/discovered-jobs` with a short-lived HS256 token minted
   from `JWT_SECRET`, including `apply_url` — the employer's own application page,
   which only Indeed exposes without signing in.

The **tracker owns de-duplication and retention**: rows upsert on
`(user_id, external_id)` and anything older than 7 days is pruned on each ingest. The
same role found on several boards stays one row per board, but the API gives rows with
the same normalised company and title a shared `match_key`, and the Discovered page
shows them as one posting with a link per board. `external_id` is the bare posting id
for LinkedIn and `seek-<id>` / `indeed-<key>` for the others. The script keeps no local
state, and re-running it is harmless. The `applied_before` flag is computed server-side
by matching the company against your tracked jobs.

## Configuration

`config.json` holds the search — boards, terms, location, `hours_old`,
`results_per_search`, and the title regexes — and is baked into the image.
`remote_search_terms` are also searched as remote roles across `remote_location`
(currently Elixir, across Australia); those only keep postings whose title names the
term, because LinkedIn pads a niche search that has no matches with unrelated postings
from any city. A single
search is abandoned after `search_timeout_seconds` (default 300), with the rest of that
board skipped, so one stuck board cannot hold up the others; JobSpy only times out
individual requests. Deployment
settings come from the environment and override the file:

| Env var | Purpose |
|---|---|
| `TRACKER_URL` | Tracker base URL (in-cluster: `http://job-tracker-backend:8080`) |
| `TRACKER_USER_ID`, `TRACKER_EMAIL` | Claims for the minted token (not in `config.json`; the CronJob sets them) |
| `JWT_SECRET` | Signing secret, from the `job-tracker-secret` secret |
| `SEARCH_LOCATION`, `HOURS_OLD`, `SITES` | Occasional overrides without rebuilding |

## Running locally

Needs Python 3.10 with JobSpy's dependencies — the repo's `.venv` from `mise install`
has them — and a dev backend (`mise run backend`, on :8081). Point `PYTHONPATH` at the
fork's JobSpy in the submodule:

```bash
PYTHONPATH=../jobspy-mcp-server/jobspy \
TRACKER_URL=http://localhost:8081 \
TRACKER_USER_ID=<your user id> \
TRACKER_EMAIL=<your login email> \
JWT_SECRET=<dev secret> \
../.venv/bin/python board_scrape.py --dry-run
```

`--dry-run` scrapes and prints the digest without posting. Other flags: `--hours N`,
`--sites seek,indeed`, `--config PATH`, `--verbose` (per-search counts on stderr).

Dependencies are JobSpy's (installed from the fork's `jobspy/requirements.txt`, at the
commit pinned by `JOBSPY_REF` in the Dockerfile) plus PyJWT.

## Throttling

A board that returns nothing for every term gets a warning naming it, so an
under-reported run is visible in `kubectl logs` instead of silently looking like a
quiet day. LinkedIn in particular soft-throttles by serving empty pages, and running the
scrape several times in quick succession reliably triggers it. The run only fails
outright when every search on every board comes back empty.
