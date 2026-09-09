#!/usr/bin/python3
"""Dispatch a job posting URL into a free Job-Apply Claude session.

Invoked by the "Send to Job-Apply" Shortcut (macOS Share Sheet). Builds the
4-line trigger message that the job-search workflow expects and submits it to a
Job-Apply-N pane via herdr.

Stdlib only, and run with /usr/bin/python3 on purpose: Shortcuts' Run Shell
Script executes in a sandbox with a minimal PATH that does not source the user's
profile, so nix/mise interpreters are not reachable from there.
"""

import argparse
import json
import os
import re
import subprocess
import sys
import time
from urllib.parse import urlparse

# Stable per-user profile symlink, NOT the resolved /nix/store path, which
# changes on every herdr update.
HERDR = "/etc/profiles/per-user/adrian/bin/herdr"

JOB_SEARCH_DIR = "/Users/adrian/Remotes/OneDrive/My Job Search"
PANE_TITLE_RE = re.compile(r"^Job-Apply-(\d+)$")
CLEAR_MARKER = "<command-name>/clear</command-name>"

STATE_PATH = os.path.expanduser("~/.local/state/job-dispatch/last.json")
# Claude writes transcripts with a lag, so a just-dispatched pane can still read
# idle-and-cleared for a moment. Hold it back until it has been seen working.
DISPATCH_COOLDOWN_SEC = 10

SOURCE_BY_HOST = (
    ("linkedin.com", "L"),
    ("indeed.", "I"),
    ("seek.com.au", "S"),
    ("glassdoor.", "G"),
)
VALID_SOURCES = set("LISGMO")


class DispatchError(Exception):
    """Fatal, with a message intended for the Shortcut's notification."""


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


def run_herdr(*args):
    try:
        proc = subprocess.run(
            [HERDR, *args], capture_output=True, text=True, timeout=30
        )
    except FileNotFoundError:
        raise DispatchError("herdr not found at %s" % HERDR)
    except subprocess.TimeoutExpired:
        raise DispatchError("herdr timed out running: %s" % " ".join(args))

    if proc.returncode != 0:
        detail = (proc.stderr or proc.stdout or "").strip().splitlines()
        raise DispatchError("herdr %s failed: %s" % (args[0], detail[0] if detail else "?"))

    try:
        return json.loads(proc.stdout)
    except ValueError:
        raise DispatchError("herdr %s returned non-JSON output" % args[0])


def list_sessions(cwd):
    """Job-Apply-N agents in cwd, ordered by their numeric suffix."""
    payload = run_herdr("agent", "list")
    found = []
    for agent in payload.get("result", {}).get("agents", []):
        if agent.get("cwd") != cwd:
            continue
        match = PANE_TITLE_RE.match(agent.get("terminal_title_stripped") or "")
        if not match:
            continue
        found.append(
            {
                "n": int(match.group(1)),
                "pane_id": agent.get("pane_id"),
                "title": agent.get("terminal_title_stripped"),
                "status": agent.get("agent_status"),
                "session_id": (agent.get("agent_session") or {}).get("value"),
            }
        )
    return sorted(found, key=lambda s: s["n"])


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


def write_state(pane_id):
    os.makedirs(os.path.dirname(STATE_PATH), exist_ok=True)
    tmp = STATE_PATH + ".tmp"
    with open(tmp, "w", encoding="utf-8") as handle:
        json.dump({"pane_id": pane_id, "at": time.time()}, handle)
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
        if session["status"] != "idle":
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


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("url", nargs="?")
    parser.add_argument("recruiter", choices=["y", "n"], nargs="?", default="n")
    parser.add_argument("easyapply", choices=["y", "n"], nargs="?", default="n")
    parser.add_argument("--source", help="override the auto-detected source letter")
    parser.add_argument("--cwd", default=JOB_SEARCH_DIR)
    parser.add_argument("--target", help="dispatch to this pane_id instead of the first free one")
    parser.add_argument("--dry-run", action="store_true")
    args = parser.parse_args()

    raw = args.url
    if raw is None or not raw.strip():
        # Fall back to stdin, in case the action is set to pass input that way.
        if not sys.stdin.isatty():
            raw = sys.stdin.read()
    url = extract_url(raw)

    source = (args.source or detect_source(url)).upper()
    if source not in VALID_SOURCES:
        raise DispatchError("invalid source %r (expected one of LISGMO)" % source)

    message = "\n".join([url, source, args.recruiter, args.easyapply])

    sessions = list_sessions(args.cwd)
    if not sessions:
        raise DispatchError("no Job-Apply sessions found in %s" % args.cwd)

    tdir = transcript_dir(args.cwd)
    state = read_state()

    if args.target:
        chosen = next((s for s in sessions if s["pane_id"] == args.target), None)
        if chosen is None:
            raise DispatchError("no such pane: %s" % args.target)
        # An explicit target still gets tested, so it can't clobber unactioned work.
        if chosen["status"] != "idle":
            raise DispatchError("%s is %s" % (chosen["title"], chosen["status"]))
        if not is_cleared(chosen["session_id"], tdir):
            raise DispatchError("%s needs /clear" % chosen["title"])
        reasons = []
    else:
        chosen, reasons = choose(sessions, tdir, state)

    if chosen is None:
        raise DispatchError("no free session: %s" % summarise(reasons))

    if args.dry_run:
        print("would send to %s (%s):" % (chosen["title"], chosen["pane_id"]))
        for line in message.splitlines():
            print("  | %s" % line)
        if reasons:
            print("skipped: %s" % summarise(reasons))
        return

    run_herdr("agent", "prompt", chosen["pane_id"], message)
    write_state(chosen["pane_id"])
    print("Sent to %s (source %s)" % (chosen["title"], source))


if __name__ == "__main__":
    try:
        main()
    except DispatchError as exc:
        print(str(exc), file=sys.stderr)
        sys.exit(1)
