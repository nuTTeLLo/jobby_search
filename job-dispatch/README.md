# job-dispatch

Send the job posting you're looking at straight into a free `Job-Apply-N` Claude
session, from the macOS Share menu in any browser.

Replaces copying the URL out of the browser and hand-typing the 4-line trigger
message that the workflow in `/Users/adrian/Remotes/OneDrive/My Job Search/CLAUDE.md`
expects:

```
<url>
<source>        L=LinkedIn I=Indeed S=Seek G=Glassdoor M=Manual O=Other
<recruiter>     y|n
<easy_apply>    y|n
```

```
File → Share (Brave / Safari / Zen)
   └─> Shortcut "Send to Job-Apply"
         └─> dispatch.py  ──herdr──>  a free Job-Apply-N session
```

## How a session is chosen

A session must be **both** idle **and** cleared.

That second condition is the point of this tool. A session parked at Step 9 —
resume built, waiting for you to review it and decide about a cover letter —
still reports `idle` to herdr. Dispatching into it would bury work you haven't
looked at yet. The convention is that a session is done only once you've
actioned the output and typed `/clear`, so freshness is checked separately:

> A session is cleared if its transcript
> (`~/.claude/projects/-Users-adrian-Remotes-OneDrive-My-Job-Search/<session>.jsonl`)
> contains no `assistant` entry after the last `/clear`.

Count *assistant* turns, not user messages: a cleared transcript still holds
user-role entries for the `/clear` command itself and its
`<local-command-caveat>` wrapper, so "any user message" would mark every session
as used.

Sessions are tried in `Job-Apply-N` numeric order. If none is eligible the run
fails with a per-session reason (`working`, `blocked`, `needs /clear`) rather
than a bare "busy". Nothing is queued and no new session is spawned.

A small state file (`~/.local/state/job-dispatch/last.json`) holds the last
dispatch, so two rapid shares can't both land in the same session before Claude
has flushed the first one to its transcript.

## dispatch.py

```
dispatch.py <url> [recruiter y|n] [easyapply y|n] [--source X] [--target <pane>] [--dry-run]
```

Prints one line on stdout for the notification; exits non-zero with the reason on
stderr. `--dry-run` shows the message and the chosen session without sending.

Source is detected from the host (`linkedin.com`→L, `indeed.`→I, `seek.com.au`→S,
`glassdoor.`→G, anything else→O) and overridable with `--source`. Postings
rehosted on Workday/Greenhouse/LiveHire correctly land on `O`; the workflow's own
source-mismatch rule records the real origin downstream.

Two constraints worth preserving if you edit it:

- **Stdlib only, `/usr/bin/python3` shebang.** Shortcuts' Run Shell Script runs in
  a sandbox with a minimal PATH that doesn't source your profile, so nix/mise
  interpreters aren't reachable from there.
- **`herdr` is called at `/etc/profiles/per-user/adrian/bin/herdr`** — the stable
  profile symlink, not the resolved `/nix/store/...-herdr-<ver>/bin/herdr`, which
  changes on every herdr update.

## Rebuilding the Shortcut

**This section is the source of truth, not the `.shortcut` file.** A Shortcut has
no source form: it lives inside `~/Library/Shortcuts/Shortcuts.sqlite`, a binary
database shared by every shortcut you own, and the `shortcuts` CLI has only
`run`, `list`, `view` and `sign` — no export. Any committed `.shortcut` is a
signed binary blob: not diffable, not reviewable, and stale as soon as you tweak
the live one.

So the Shortcut is kept deliberately dumb — receive, prompt, shell out, notify —
and all the judgement lives in `dispatch.py`, which is plain text. Recreate it in
Shortcuts.app in about two minutes:

1. New shortcut named **Send to Job-Apply**.
2. In the shortcut's settings (ⓘ): tick **Use as Quick Action** → **Share Sheet**,
   and set **Receive** → **URLs**.
3. **Choose from Menu** with four items — `Direct`, `Via recruiter`,
   `Easy apply`, `Recruiter + Easy apply`. One menu rather than two yes/no
   prompts keeps it to a single tap.
4. In each branch, a **Run Shell Script** action (Shell **zsh**, **Pass input as
   arguments**), with the flags for that branch:

   ```
   /usr/bin/python3 /Users/adrian/dev/jobby_search/job-dispatch/dispatch.py "<Shortcut Input>" n n
   ```

   `Via recruiter` → `y n` · `Easy apply` → `n y` · `Recruiter + Easy apply` → `y y`

   > **Insert `Shortcut Input` as a magic variable** — type the quotes, then pick
   > it from the variable bar. Do **not** write `"$1"` here. Inside a menu branch
   > an action's implicit input is the *menu's* output, so `"$1"` arrives as the
   > menu label (`Direct`) rather than the URL. The magic variable binds to the
   > original share-sheet input no matter where the action sits.

   Any POSIX shell works — the command uses no shell-specific syntax. Avoid the
   nix `fish`/`nu` entries in the shell list, which don't use `$1`-style args.
5. **Show Notification** with the Shell Script output.

To snapshot it afterwards: export from Shortcuts.app (File → Export), then
`shortcuts sign --mode anyone -i in.shortcut -o "Send to Job-Apply.shortcut"` so
re-importing doesn't fight you.

### First run

Expect prompts for file access (reading `~/.claude/projects`) and for the herdr
socket. The Quick Action must also be enabled under System Settings → Extensions
→ Sharing if it doesn't appear in the Share menu.

## Notes

- Enabling File → Share differs per browser; Brave ships Chromium's sharing hub
  disabled by default. Zen (Firefox-derived) sets `NSAppleScriptEnabled` but
  ships no `.sdef`, so it has no scriptable tab/URL access — the share sheet is
  the only route that covers all three browsers, which is why it's the trigger.
- Safari's scripting dictionary exposes `source` (the page's HTML). If Step 1's
  LinkedIn login-wall fallback ever becomes annoying, that's the cheapest way to
  hand the session a rendered job description — Safari only.
- `dispatch.py`'s only coupling to the rest of this repo is conceptual; it talks
  to herdr and the OneDrive workflow folder, never to the tracker API or database.
