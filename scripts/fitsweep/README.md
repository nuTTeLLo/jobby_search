# Fit sweep

Scores the Discovered page's new postings against your fit profile (the Profile
page), stores each score in the tracker, and sends a Telegram "top fits today"
message. Runs on this Mac at 09:30, half an hour after the cluster's 09:00 board
scrape, using the Claude Code CLI on your Claude subscription.

It runs here rather than in the cluster because one `claude -p` run peaks around
300 MB, against the backend pod's 96Mi limit on a node already near its memory.
The scoring code lives in the backend (`backend/internal/fit`), so moving it into
the cluster later is mostly packaging: the backend's `POST /api/fit-check` already
uses it, and only needs Claude Code in the image and more memory.

## How it works

1. The board scrape stores each posting's description with the posting.
2. `fitsweep` (`backend/cmd/fitsweep`) asks the tracker for postings that have no
   score, or were scored before the profile was last saved
   (`GET /api/discovered-jobs/unscored`), skipping dismissed ones.
3. Each role is scored once, even when it is on several boards, with
   `claude -p` (no tools, MCP servers or settings loaded), and the score is stored
   on every copy (`PUT /api/discovered-jobs/:id/fit`).
4. Roles scoring 4 or 5 go into the Telegram message, best first.

Postings that fail to score, or that wait while the Mac is off, are picked up by
the next run. It stops after 100 roles (`FIT_MAX_PER_RUN`), or after three
failures in a row.

## Install

1. Fill in `~/.config/jobsearch/fitsweep.env`: `install.sh` creates it from a
   template the first time. The account is the one in `My Job Search/.tracker.env`.
   The file is read from `~/.config` so that the job never touches OneDrive.
2. Run `scripts/fitsweep/install.sh`. It builds the sweep, installs the wrapper as
   `~/.local/bin/jobsearch-fitsweep`, and loads the LaunchAgent
   `com.adriantan.jobsearch.fitsweep`.
3. Try it: `jobsearch-fitsweep --dry-run` scores and prints the message without
   storing anything or sending it.

Telegram uses the same bot as the rejection sweeps (`~/.config/jobsearch/telegram.env`).
Logs go to `~/Library/Logs/JobSearch/fit_sweep.{out,err}.log`.

Optional environment overrides: `FIT_MODEL` (default `claude-opus-5-5`),
`FIT_EFFORT` (default `low`), `FIT_CLAUDE_BIN`, `FIT_MAX_PER_RUN`.
