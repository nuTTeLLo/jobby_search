#!/usr/bin/python3
"""Dispatch a job posting URL into a free Job-Apply Claude session.

Invoked by the "Send to Job-Apply" Shortcut (macOS Share Sheet). Builds the
4-line trigger message that the job-search workflow expects and submits it to a
Job-Apply-N pane, in Rex (the default), tuios or herdr. tuios answers herdr's
command line on its own socket, so both are driven through the herdr CLI; Rex
has no such socket and is driven through its own CLI.

Stdlib only, and run with /usr/bin/python3 on purpose: Shortcuts' Run Shell
Script executes in a sandbox with a minimal PATH that does not source the user's
profile, so nix/mise interpreters are not reachable from there.
"""

import argparse
import getpass
import json
import os
import re
import shutil
import subprocess
import sys
import time
from urllib.parse import unquote, urlparse

# Stable per-user profile symlink, NOT the resolved /nix/store path, which
# changes on every herdr update.
HERDR = "/etc/profiles/per-user/%s/bin/herdr" % getpass.getuser()

# Which multiplexer's sessions to dispatch into. tuios and herdr each run their
# own Job-Apply-N sessions; tuios's herdr-compatible socket lives at a path
# derived from the uid, so it survives daemon restarts. None means herdr's own
# server, which the herdr CLI finds by itself.
BACKEND_SOCKETS = {
    "tuios": "/tmp/tuios-%d/tuios.sock.herdr" % os.getuid(),
    "herdr": None,
}
BACKENDS = sorted(list(BACKEND_SOCKETS) + ["rex"])
DEFAULT_BACKEND = "rex"
# Without --backend, the others are tried in this order after the preferred one.
FALLBACK_ORDER = ("rex", "tuios", "herdr")

# Rex puts its CLI on PATH (as REX_BIN_DIR) only inside its own terminals; the
# Shortcut's sandbox gets neither, so also look where Rex installs it: the
# server it keeps current under Application Support, then the app bundle (Beta
# today; the plain name is for when it drops the tag).
REX_CANDIDATES = [
    os.path.join(os.environ["REX_BIN_DIR"], "rex") if os.environ.get("REX_BIN_DIR") else None,
    shutil.which("rex"),
    os.path.expanduser(
        "~/Library/Application Support/com.superlogical.rex/RexServer/Versions/Current"
        "/Rex Server.app/Contents/MacOS/rex"
    ),
] + [
    "/Applications/%s.app/Contents/Helpers/Rex Server.rex/Contents/MacOS/rex" % app
    for app in ("Rex Beta", "Rex")
]
REX_CANDIDATES = [p for p in REX_CANDIDATES if p]
REX = next((p for p in REX_CANDIDATES if os.access(p, os.X_OK)), REX_CANDIDATES[-1])
# Claude only treats a multi-line message as one prompt if it arrives as a
# paste; typed newlines would submit the URL line on its own.
PASTE_START, PASTE_END = "\x1b[200~", "\x1b[201~"
# Give Claude a moment to take the paste in before Enter submits it.
PASTE_SETTLE_SEC = 0.3
CLAUDE_SESSIONS_DIR = os.path.expanduser("~/.claude/sessions")

JOB_SEARCH_DIR = os.path.expanduser("~/Remotes/OneDrive/My Job Search")
PANE_TITLE_RE = re.compile(r"^Job-Apply-(\d+)$")
# tuios sends only the raw title, which Claude prefixes with a status glyph.
TITLE_GLYPHS_RE = re.compile(r"^[^\w]+")
# tuios marks a session "done" when it finishes a turn; herdr leaves it "idle".
# Either is free as far as status goes -- is_cleared() decides the rest.
FREE_STATUSES = ("idle", "done")
CLEAR_MARKER = "<command-name>/clear</command-name>"
# Claude writes the session's name in its first rows; only read that far.
NAME_ROW_TYPES = ("custom-title", "agent-name")
HEADER_SCAN_LINES = 50

STATE_PATH = os.path.expanduser("~/.local/state/job-dispatch/last.json")
LOG_PATH = os.path.expanduser("~/.local/state/job-dispatch/dispatch.log")
# Claude writes transcripts with a lag, so a just-dispatched pane can still read
# idle-and-cleared for a moment. Hold it back until it has been seen working.
DISPATCH_COOLDOWN_SEC = 10

SOURCE_BY_HOST = (
    ("linkedin.com", "L"),
    ("indeed.", "I"),
    ("seek.com", "S"),  # www.seek.com.au and au.seek.com
    ("glassdoor.", "G"),
)
VALID_SOURCES = set("LISGMO")


class DispatchError(Exception):
    """Fatal, with a message intended for the Shortcut's notification."""


def log(event, **fields):
    """Append one line per invocation.

    The share sheet is the only part of this pipeline we cannot inspect, so
    record exactly what arrived: if the same URL shows up for two genuinely
    different pages, the staleness is upstream of this script.
    """
    try:
        os.makedirs(os.path.dirname(LOG_PATH), exist_ok=True)
        parts = ["%s %s" % (time.strftime("%Y-%m-%dT%H:%M:%S"), event)]
        parts += ["%s=%r" % (k, v) for k, v in sorted(fields.items())]
        with open(LOG_PATH, "a", encoding="utf-8") as handle:
            handle.write(" ".join(parts) + "\n")
    except OSError:
        pass  # never let logging break a dispatch


def transcript_dir(cwd):
    """Claude's per-project transcript dir: path with / and spaces as dashes."""
    slug = cwd.replace("/", "-").replace(" ", "-")
    return os.path.expanduser(os.path.join("~/.claude/projects", slug))


def detect_source(url):
    host = (urlparse(url).hostname or "").lower()
    for needle, letter in SOURCE_BY_HOST:
        if needle in host:
            return letter
    # Postings are often rehosted on Workday/Greenhouse/LiveHire; "O" is
    # correct there, and the workflow's source-mismatch rule handles the rest.
    return "O"

URL_RE = re.compile(r"https?://[^\s<>\"']+")


def extract_url(raw):
    """Pull a web URL out of whatever the Shortcut handed over.

    Browsers don't agree on what a shared "URL" is: some pass a bare string,
    some a web-page item that arrives as "Page Title\nhttps://...". Shortcuts
    also happily passes an empty string when "Pass input as arguments" is off,
    or a menu label when the actions are wired in the wrong order -- so say
    plainly what arrived instead of failing with a bare "not a web URL".
    """
    text = (raw or "").strip()
    if not text:
        raise DispatchError(
            "no URL received (is 'Pass input as arguments' set, and is the "
            "shell script reading the URL rather than the menu choice?)"
        )
    match = URL_RE.search(text)
    if not match:
        raise DispatchError("no http(s) URL in input: %r" % text[:120])
    return match.group(0).rstrip(".,)]}\"'")


# Browsers whose active-tab URL can be read live. Zen sets NSAppleScriptEnabled
# but ships no .sdef, so it has no tab access and is deliberately absent.
SCRIPTABLE_BROWSERS = {
    "Safari": 'tell application "Safari" to get URL of current tab of front window',
    "Orion": 'tell application "Orion" to get URL of current tab of front window',
    "Brave Browser": 'tell application "Brave Browser" to get URL of active tab of front window',
}


def notify(title, body, subtitle=None):
    """Post a macOS notification.

    The Shortcut has no Show Notification action, so without this the script's
    output goes nowhere and a wrong or refused dispatch is invisible.
    """
    def esc(text):
        return str(text).replace("\\", "\\\\").replace('"', '\\"')

    script = 'display notification "%s" with title "%s"' % (esc(body), esc(title))
    if subtitle:
        script += ' subtitle "%s"' % esc(subtitle)
    osascript(script)


def frontmost_app():
    """Frontmost app name via LaunchServices.

    Deliberately not System Events: that needs an Automation grant for the
    *calling* app, and under the Shortcuts sandbox it hung until timeout and
    silently disabled the whole live-URL path.
    """
    try:
        asn = subprocess.run(
            ["/usr/bin/lsappinfo", "front"], capture_output=True, text=True, timeout=3
        ).stdout.strip()
        if not asn:
            return None
        out = subprocess.run(
            ["/usr/bin/lsappinfo", "info", "-only", "name", asn],
            capture_output=True, text=True, timeout=3,
        ).stdout
    except (OSError, subprocess.TimeoutExpired):
        return None
    match = re.search(r'"LSDisplayName"="([^"]*)"', out or "")
    return match.group(1) if match else None


def osascript(script, timeout=3):
    try:
        proc = subprocess.run(
            ["/usr/bin/osascript", "-e", script],
            capture_output=True, text=True, timeout=timeout,
        )
    except (OSError, subprocess.TimeoutExpired):
        return None
    if proc.returncode != 0:
        return None
    return (proc.stdout or "").strip() or None


def live_url():
    """The frontmost browser's actual tab URL, if it can be asked.

    The share sheet hands over a snapshot that some browsers (Orion in
    particular) fail to refresh, so the same URL comes back for different
    pages. Asking the browser directly sidesteps that. Returns None when the
    frontmost app isn't a scriptable browser -- Zen, or the Shortcuts UI itself
    -- and the caller then trusts the share input.
    """
    app = frontmost_app()
    if app not in SCRIPTABLE_BROWSERS:
        return None, app
    url = osascript(SCRIPTABLE_BROWSERS[app])
    if url and urlparse(url).scheme in ("http", "https"):
        return url, app
    # Scriptable browser that wouldn't answer: almost always a missing
    # Automation grant for the calling app. Worth naming, since it silently
    # demotes us back to the share input.
    log("live_url_failed", front=app, hint="grant Automation for the caller")
    return None, app


def backend_env(backend):
    """Environment that points the herdr CLI at the chosen backend.

    HERDR_SOCKET_PATH is always set or removed explicitly: run from a tuios pane
    it is already set to tuios's socket, which would send a herdr dispatch to
    tuios.
    """
    env = dict(os.environ)
    socket_path = BACKEND_SOCKETS[backend]
    if socket_path is None:
        env.pop("HERDR_SOCKET_PATH", None)
        return env
    if not os.path.exists(socket_path):
        raise DispatchError("%s is not running (no socket at %s)" % (backend, socket_path))
    env["HERDR_SOCKET_PATH"] = socket_path
    return env


def run_herdr(*args, env=None):
    try:
        proc = subprocess.run(
            [HERDR, *args], capture_output=True, text=True, timeout=30, env=env
        )
    except FileNotFoundError:
        raise DispatchError("herdr not found at %s" % HERDR)
    except subprocess.TimeoutExpired:
        raise DispatchError("herdr timed out running: %s" % " ".join(args))

    raw = (proc.stdout or "").strip()
    payload = None
    # herdr writes success JSON to stdout but error JSON to stderr, so try both.
    for candidate in (raw, (proc.stderr or "").strip()):
        if not candidate:
            continue
        try:
            payload = json.loads(candidate)
            break
        except ValueError:
            continue

    # herdr reports failures as {"error": {"code", "message"}} -- surface just the
    # first line of the message, since this ends up in a notification bubble.
    if isinstance(payload, dict) and payload.get("error"):
        err = payload["error"]
        code = err.get("code", "error")
        first = (err.get("message") or "").strip().splitlines()
        hint = first[0] if first else code
        if code == "protocol_mismatch":
            hint = "herdr server needs restarting (CLI upgraded past the running server)"
        raise DispatchError("herdr: %s" % hint)

    if proc.returncode != 0:
        detail = (proc.stderr or raw or "").strip().splitlines()
        raise DispatchError("herdr %s failed: %s" % (args[0], detail[0] if detail else "?"))

    if payload is None:
        raise DispatchError("herdr %s returned non-JSON output" % args[0])
    return payload


def transcript_header(path):
    """(session name, start timestamp) from the first rows of a transcript."""
    name = started = None
    try:
        with open(path, encoding="utf-8", errors="replace") as handle:
            for index, line in enumerate(handle):
                if index >= HEADER_SCAN_LINES:
                    break
                try:
                    row = json.loads(line)
                except ValueError:
                    continue
                if name is None and row.get("type") in NAME_ROW_TYPES:
                    name = row.get("customTitle") or row.get("agentName")
                if started is None and row.get("timestamp"):
                    started = row["timestamp"]
                if name and started:
                    break
    except OSError:
        pass
    return name, started


def latest_named_sessions(tdir):
    """{name: (started, session_id)} for the newest-started transcript per name."""
    latest = {}
    try:
        entries = os.listdir(tdir)
    except OSError:
        return latest
    for entry in entries:
        if not entry.endswith(".jsonl"):
            continue
        name, started = transcript_header(os.path.join(tdir, entry))
        if not (name and started and PANE_TITLE_RE.match(name)):
            continue
        if name not in latest or started > latest[name][0]:
            latest[name] = (started, entry[: -len(".jsonl")])
    return latest


def current_session_id(title, herdr_id, latest, tdir, live):
    """The session actually live in a pane, which herdr can lag behind.

    /clear starts a fresh session (new transcript) in the same pane, but herdr's
    agent_session can keep reporting the old id -- whose transcript still shows
    the finished job, so the pane reads "needs /clear" forever. Claude tags every
    new transcript with the pane's name up front, and a /clear always starts a
    newer session, so trust whichever of herdr's id and the newest transcript
    carrying this name started later. File mtimes are no use here: Claude
    touches old transcripts without adding turns.

    An id some running Claude process still holds (live) hasn't lagged, and is
    kept: another multiplexer can have a newer Job-Apply-N of the same name in
    the same cwd, and that one belongs to its pane, not this one.
    """
    newest = latest.get(title)
    if newest is None or newest[1] == herdr_id or herdr_id in live:
        return herdr_id
    if herdr_id:
        _, herdr_started = transcript_header(os.path.join(tdir, "%s.jsonl" % herdr_id))
        if herdr_started is None or herdr_started >= newest[0]:
            # No transcript yet (never used), or herdr's is the newer one.
            return herdr_id
    log("session_override", pane=title, herdr=herdr_id, using=newest[1])
    return newest[1]


def pane_title(agent):
    stripped = agent.get("terminal_title_stripped")
    if stripped:
        return stripped
    return TITLE_GLYPHS_RE.sub("", agent.get("terminal_title") or "").strip()


def herdr_agents(backend):
    payload = run_herdr("agent", "list", env=backend_env(backend))
    for agent in payload.get("result", {}).get("agents", []):
        yield {
            "pane_id": agent.get("pane_id"),
            "title": pane_title(agent),
            "cwd": agent.get("cwd"),
            "status": agent.get("agent_status"),
            "session_id": (agent.get("agent_session") or {}).get("value"),
        }


def run_rex(*args, parse=True):
    # --autostart=false: a missing server should fail the dispatch, not start
    # an empty one with no Job-Apply panes in it.
    try:
        proc = subprocess.run(
            [REX, "--autostart=false", *args], capture_output=True, text=True, timeout=30
        )
    except FileNotFoundError:
        raise DispatchError("rex not found at %s" % REX)
    except subprocess.TimeoutExpired:
        raise DispatchError("rex timed out running: %s" % " ".join(args[:3]))
    if proc.returncode != 0:
        detail = (proc.stderr or proc.stdout or "").strip().splitlines()
        raise DispatchError("rex %s failed: %s" % (args[0], detail[0] if detail else "?"))
    if not parse:
        return None
    try:
        return json.loads(proc.stdout)
    except ValueError:
        raise DispatchError("rex %s returned non-JSON output" % args[0])


def claude_processes():
    """Running Claude processes' session records, and every process's parent.

    Claude keeps ~/.claude/sessions/<pid>.json per process, holding the live
    sessionId (it changes the moment /clear starts a new session) and the cwd.
    Files of exited processes linger, so keep only pids still running.
    """
    try:
        out = subprocess.run(
            ["/bin/ps", "-axo", "pid=,ppid="], capture_output=True, text=True, timeout=5
        ).stdout
    except (OSError, subprocess.TimeoutExpired):
        out = ""
    parents = {}
    for line in out.splitlines():
        fields = line.split()
        if len(fields) == 2:
            parents[int(fields[0])] = int(fields[1])
    records = {}
    try:
        entries = os.listdir(CLAUDE_SESSIONS_DIR)
    except OSError:
        entries = []
    for entry in entries:
        if not entry.endswith(".json"):
            continue
        try:
            with open(os.path.join(CLAUDE_SESSIONS_DIR, entry), encoding="utf-8") as handle:
                record = json.load(handle)
        except (OSError, ValueError):
            continue
        if record.get("pid") in parents:
            records[record["pid"]] = record
    return records, parents


def pane_claude(root_pid, records, parents):
    """Session record of the newest Claude process under a pane's root process."""
    found = []
    for pid, record in records.items():
        while pid and pid != 1:
            if pid == root_pid:
                found.append(record)
                break
            pid = parents.get(pid)
    return max(found, key=lambda r: r.get("startedAt") or 0, default=None)


def rex_agents():
    """Rex terminal blocks, shaped like herdr's agent list.

    Rex reports Claude's state but not its session, so each pane is tied to
    its own session through the Claude process running under it. Matching
    transcripts by the pane's name instead could pick up another
    multiplexer's Job-Apply-N in the same cwd. Title is read first so only
    Job-Apply panes pay for the other calls.
    """
    records, parents = claude_processes()
    for session in run_rex("ls", "--json").get("sessions", []):
        sid = session["session_id"]
        for block in run_rex("block", "ls", "-s", sid, "--json").get("blocks", []):
            bid = block["block_id"]
            if not block.get("flavor", "").startswith("com.superlogical.terminal"):
                continue
            title = run_rex("block", "call", "-s", sid, "-b", bid, "title").get("title") or ""
            title = TITLE_GLYPHS_RE.sub("", title).strip()
            if not PANE_TITLE_RE.match(title):
                continue
            root = (run_rex("block", "call", "-s", sid, "-b", bid, "process").get("child") or {}).get("pid")
            claude = pane_claude(root, records, parents)
            if claude is None:
                # Still list the pane, with a reason, rather than hide it.
                # The shell's OSC 7 location is a file:// URL with the host in it.
                pwd = run_rex("block", "call", "-s", sid, "-b", bid, "pwd").get("pwd") or ""
                yield {
                    "pane_id": bid,
                    "rex_session": sid,
                    "title": title,
                    "cwd": unquote(urlparse(pwd).path) or None,
                    "status": "no Claude session found",
                    "session_id": None,
                }
                continue
            status = run_rex("block", "call", "-s", sid, "-b", bid, "program_status")
            state = next(
                (r for r in status.get("records", []) if r.get("effective_app") == "claude-code"),
                {},
            )
            yield {
                "pane_id": bid,
                "rex_session": sid,
                "title": title,
                "cwd": claude.get("cwd"),
                "status": state.get("state"),
                "session_id": claude.get("sessionId"),
            }


def list_sessions(cwd, backend):
    """Job-Apply-N agents in cwd, ordered by their numeric suffix."""
    agents = rex_agents() if backend == "rex" else herdr_agents(backend)
    tdir = transcript_dir(cwd)
    latest = latest_named_sessions(tdir)
    live = set() if backend == "rex" else {
        r.get("sessionId") for r in claude_processes()[0].values()
    }
    found = []
    for agent in agents:
        if agent["cwd"] != cwd:
            continue
        title = agent["title"]
        match = PANE_TITLE_RE.match(title)
        if not match:
            continue
        session_id = agent["session_id"]
        if backend != "rex":
            # Rex's id comes from Claude itself and never lags; herdr's can.
            session_id = current_session_id(title, session_id, latest, tdir, live)
        found.append(
            {
                "n": int(match.group(1)),
                "pane_id": agent["pane_id"],
                "rex_session": agent.get("rex_session"),
                "title": title,
                "status": agent["status"],
                "session_id": session_id,
            }
        )
    return sorted(found, key=lambda s: s["n"])


def send_prompt(backend, session, message):
    if backend != "rex":
        run_herdr("agent", "prompt", session["pane_id"], message, env=backend_env(backend))
        return
    target = ["-s", session["rex_session"], "-b", session["pane_id"]]
    run_rex("send", *target, PASTE_START + message + PASTE_END, parse=False)
    time.sleep(PASTE_SETTLE_SEC)
    run_rex("send-key", *target, "enter", parse=False)


def is_cleared(session_id, tdir):
    """True if the transcript shows no assistant turn after the last /clear.

    Counting *assistant* entries matters: a cleared transcript still holds
    user-role entries for the /clear command itself and its
    <local-command-caveat> meta wrapper, so "any user message" would mark every
    session as used. Position matters too, not just totals -- a session that ran
    a job has its /clear near the top with all its turns after it.
    """
    if not session_id:
        return False
    path = os.path.join(tdir, "%s.jsonl" % session_id)
    if not os.path.exists(path):
        # No transcript yet: launched but never used.
        return True

    last_clear = -1
    assistant_at = []
    try:
        with open(path, encoding="utf-8", errors="replace") as handle:
            for index, line in enumerate(handle):
                line = line.strip()
                if not line:
                    continue
                try:
                    row = json.loads(line)
                except ValueError:
                    continue
                rtype = row.get("type")
                if rtype == "assistant":
                    assistant_at.append(index)
                elif rtype == "user" and CLEAR_MARKER in json.dumps(row.get("message", "")):
                    last_clear = index
    except OSError as exc:
        raise DispatchError("cannot read transcript for %s: %s" % (session_id[:8], exc))

    return not any(index > last_clear for index in assistant_at)


def read_state():
    try:
        with open(STATE_PATH, encoding="utf-8") as handle:
            return json.load(handle)
    except (OSError, ValueError):
        return {}


def write_state(pane_id, url=None):
    os.makedirs(os.path.dirname(STATE_PATH), exist_ok=True)
    tmp = STATE_PATH + ".tmp"
    with open(tmp, "w", encoding="utf-8") as handle:
        json.dump({"pane_id": pane_id, "at": time.time(), "url": url}, handle)
    os.replace(tmp, STATE_PATH)


def in_cooldown(session, state):
    """Guard against two rapid dispatches landing in the same session."""
    if state.get("pane_id") != session["pane_id"]:
        return False
    if session["status"] == "working":
        return False  # already picked the job up
    return (time.time() - state.get("at", 0)) < DISPATCH_COOLDOWN_SEC


def choose(sessions, tdir, state):
    """First eligible session, or a per-session explanation of why not."""
    reasons = []
    for session in sessions:
        if session["status"] not in FREE_STATUSES:
            reasons.append((session["title"], session["status"]))
        elif in_cooldown(session, state):
            reasons.append((session["title"], "just dispatched"))
        elif not is_cleared(session["session_id"], tdir):
            reasons.append((session["title"], "needs /clear"))
        else:
            return session, reasons
    return None, reasons


def summarise(reasons):
    counts = {}
    for _, reason in reasons:
        counts[reason] = counts.get(reason, 0) + 1
    return ", ".join("%d %s" % (n, reason) for reason, n in sorted(counts.items()))


def backend_order(pinned):
    """Backends to try: just the one asked for, else the preferred one first."""
    if pinned:
        return [pinned]
    preferred = os.environ.get("JOB_DISPATCH_BACKEND") or DEFAULT_BACKEND
    return [preferred] + [b for b in FALLBACK_ORDER if b != preferred]


def pick(backend, cwd, target, state):
    """(session, skipped reasons) in one backend; DispatchError says why none."""
    sessions = list_sessions(cwd, backend)
    if not sessions:
        raise DispatchError("no Job-Apply sessions in %s" % cwd)
    tdir = transcript_dir(cwd)

    if not target:
        chosen, reasons = choose(sessions, tdir, state)
        if chosen is None:
            raise DispatchError("no free session: %s" % summarise(reasons))
        return chosen, reasons

    chosen = next((s for s in sessions if s["pane_id"] == target), None)
    if chosen is None:
        raise DispatchError("no such pane: %s" % target)
    # An explicit target still gets tested, so it can't clobber unactioned work.
    if chosen["status"] not in FREE_STATUSES:
        raise DispatchError("%s is %s" % (chosen["title"], chosen["status"]))
    if not is_cleared(chosen["session_id"], tdir):
        raise DispatchError("%s needs /clear" % chosen["title"])
    return chosen, []


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("url", nargs="?")
    parser.add_argument("recruiter", choices=["y", "n"], nargs="?", default="n")
    parser.add_argument("easyapply", choices=["y", "n"], nargs="?", default="n")
    parser.add_argument("--source", help="override the auto-detected source letter")
    parser.add_argument("--cwd", default=JOB_SEARCH_DIR)
    parser.add_argument("--target", help="dispatch to this pane_id instead of the first free one")
    parser.add_argument(
        "--backend", choices=BACKENDS,
        help="use only this multiplexer's Job-Apply sessions (default: try "
        "$JOB_DISPATCH_BACKEND, else %s, then the others)" % DEFAULT_BACKEND,
    )
    parser.add_argument("--dry-run", action="store_true")
    parser.add_argument(
        "--resolve-only", action="store_true",
        help="print the URL that would be dispatched, then exit (for the menu prompt)",
    )
    parser.add_argument(
        "--no-notify", action="store_true", help="suppress the macOS notification",
    )
    parser.add_argument(
        "--no-live", action="store_true",
        help="ignore the frontmost browser; use the passed URL (for testing)",
    )
    parser.add_argument(
        "--force", action="store_true",
        help="dispatch even if this URL matches the previous one",
    )
    args = parser.parse_args()
    # argparse checks choices only for values given on the command line, not
    # for a preference taken from the environment, which --backend overrides.
    preferred = os.environ.get("JOB_DISPATCH_BACKEND")
    if not args.backend and preferred and preferred not in BACKENDS:
        parser.error("JOB_DISPATCH_BACKEND=%r is not one of: %s"
                     % (preferred, ", ".join(BACKENDS)))
    log("invoked", argv=sys.argv[1:])

    raw = args.url
    if raw is None or not raw.strip():
        # Fall back to stdin, in case the action is set to pass input that way.
        if not sys.stdin.isatty():
            raw = sys.stdin.read()
    shared = None
    try:
        shared = extract_url(raw)
    except DispatchError:
        shared = None  # may still be recoverable from the live browser

    live, front_app = (None, None) if args.no_live else live_url()
    if live and shared and live != shared:
        log("stale_share_input", front=front_app, shared=shared, live=live)
    url = live or shared
    if not url:
        raise DispatchError(
            "no URL: share input empty and %s is not scriptable"
            % (front_app or "the frontmost app")
        )
    log("resolved", url=url, front=front_app, via="live" if live else "share")

    if args.resolve_only:
        print(url)
        return

    source = (args.source or detect_source(url)).upper()
    if source not in VALID_SOURCES:
        raise DispatchError("invalid source %r (expected one of LISGMO)" % source)

    message = "\n".join([url, source, args.recruiter, args.easyapply])

    previous = read_state().get("url")
    if url == previous and not args.force and not args.dry_run:
        raise DispatchError(
            "same URL as the last dispatch - share sheet likely returned a stale "
            "page. Re-share, or pass --force if this repeat is deliberate."
        )

    state = read_state()
    failures = []
    for backend in backend_order(args.backend):
        try:
            chosen, reasons = pick(backend, args.cwd, args.target, state)
            break
        except DispatchError as exc:
            failures.append("%s: %s" % (backend, exc))
            log("backend_skipped", backend=backend, reason=str(exc))
    else:
        raise DispatchError("no free Job-Apply session - " + "; ".join(failures))

    if args.dry_run:
        print("would send to %s %s (%s):" % (backend, chosen["title"], chosen["pane_id"]))
        for line in message.splitlines():
            print("  | %s" % line)
        for line in failures:
            print("skipped %s" % line)
        if reasons:
            print("skipped: %s" % summarise(reasons))
        return

    log("dispatch", url=url, source=source, pane=chosen["pane_id"], backend=backend)
    send_prompt(backend, chosen, message)
    write_state(chosen["pane_id"], url)
    origin = "live" if live else "share sheet"
    flags = "".join([" recruiter" if args.recruiter == "y" else "",
                     " easy-apply" if args.easyapply == "y" else ""])
    print("Sent to %s (%s, via %s)\n%s" % (chosen["title"], source, origin, url))
    if not args.no_notify:
        notify(
            "Sent to %s" % chosen["title"],
            url,
            subtitle="source %s - via %s%s" % (source, origin, flags),
        )


if __name__ == "__main__":
    try:
        main()
    except DispatchError as exc:
        print(str(exc), file=sys.stderr)
        log("failed", reason=str(exc))
        if "--no-notify" not in sys.argv:
            notify("Job dispatch failed", str(exc))
        sys.exit(1)
