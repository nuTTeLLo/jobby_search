#!/usr/bin/env python3
"""Daily job board scrape -> job tracker "Discovered" feed.

Searches LinkedIn, Seek and Indeed through the in-cluster JobSpy MCP server for roles
posted in the last 24 hours, filters to the role families being targeted, classifies
each surviving posting as Easy Apply or external, and POSTs the batch to the job
tracker.

The tracker owns de-duplication and retention: rows are upserted on
(user_id, external_id) and pruned to a rolling window, and the same role found on
several boards is grouped on read by company and title. This script therefore keeps
no local state and a re-run is harmless.

Stdlib only, plus PyJWT to mint the tracker token.
"""

import argparse
import json
import os
import random
import re
import socket
import sys
import time
import urllib.error
import urllib.request
from datetime import datetime, timedelta

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
USER_AGENT = (
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 "
    "(KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
)
LINKEDIN_POSTING_URL = "https://www.linkedin.com/jobs-guest/jobs/api/jobPosting"
SEEK_GRAPHQL_URL = "https://www.seek.com.au/graphql"

# JobSpy prefixes its ids per board. LinkedIn rows keep the bare posting id so they
# carry on from the rows the LinkedIn-only scraper wrote; the others get a readable
# board prefix so ids from different boards can never collide.
SITES = {
    "linkedin": {"jobspy_prefix": "li-", "external_prefix": ""},
    "seek": {"jobspy_prefix": "se-", "external_prefix": "seek-"},
    "indeed": {"jobspy_prefix": "in-", "external_prefix": "indeed-"},
}

APPLY_EASY = "easy_apply"
APPLY_EXTERNAL = "external"
APPLY_UNKNOWN = "unknown"


class BoardUnavailable(Exception):
    """A board's MCP search timed out; the rest of that board is skipped."""


# --------------------------------------------------------------------------- config


def load_config(path):
    """Config file provides the search itself; env vars provide the deployment.

    The container gets tracker URL and credentials from Kubernetes, while search
    terms and filters stay baked into the image alongside the code.
    """
    with open(path, "r", encoding="utf-8") as fh:
        cfg = json.load(fh)

    for env_key, cfg_key in (
        ("TRACKER_URL", "tracker_url"),
        ("TRACKER_USER_ID", "tracker_user_id"),
        ("TRACKER_EMAIL", "tracker_email"),
        ("JWT_SECRET", "jwt_secret"),
        ("MCP_URL", "mcp_url"),
        ("SEARCH_LOCATION", "location"),
    ):
        value = os.environ.get(env_key)
        if value:
            cfg[cfg_key] = value

    if os.environ.get("HOURS_OLD"):
        cfg["hours_old"] = int(os.environ["HOURS_OLD"])
    if os.environ.get("SITES"):
        cfg["sites"] = [s.strip() for s in os.environ["SITES"].split(",") if s.strip()]

    # Local runs may still point at a .env file rather than passing the secret.
    if not cfg.get("jwt_secret") and cfg.get("jwt_secret_env_path"):
        cfg["jwt_secret"] = read_jwt_secret(cfg["jwt_secret_env_path"])

    return cfg


def read_jwt_secret(env_path):
    with open(env_path, "r", encoding="utf-8") as fh:
        for line in fh:
            line = line.strip()
            if line.startswith("JWT_SECRET="):
                return line.split("=", 1)[1].strip().strip("'\"")
    raise ValueError("JWT_SECRET not found in %s" % env_path)


# ---------------------------------------------------------------------------- fetch


def http_request(url, data=None, headers=None, timeout=25):
    req = urllib.request.Request(
        url,
        data=json.dumps(data).encode() if data is not None else None,
        method="POST" if data is not None else "GET",
        headers={"User-Agent": USER_AGENT, "Accept-Language": "en-AU,en;q=0.9", **(headers or {})},
    )
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        return resp.read().decode("utf-8", errors="replace")


def request_with_retry(url, label, warnings, **kwargs):
    """Fetch, retrying transport errors and throttling. Returns the body or None."""
    for attempt in range(3):
        try:
            return http_request(url, **kwargs)
        except urllib.error.HTTPError as exc:
            if exc.code in (429, 500, 502, 503, 504) and attempt < 2:
                time.sleep(5 * (2**attempt))
                continue
            warnings.append("%s failed: HTTP %s" % (label, exc.code))
            return None
        except (urllib.error.URLError, TimeoutError, OSError) as exc:
            if attempt < 2:
                time.sleep(5 * (2**attempt))
                continue
            warnings.append("%s failed: %s" % (label, exc))
            return None
    return None


def search_board(cfg, site, term, warnings):
    """One JobSpy search for one term on one board, via the MCP server.

    Boards are searched one at a time so a board that errors or throttles costs
    only its own results, and so each gets its own date window (see seek below).

    Not retried: the MCP kills its JobSpy child after five minutes, so a timeout
    here means the search or the server is stuck, and asking again only queues more
    work on it. Raises BoardUnavailable so the caller skips the board's other terms.
    """
    hours = cfg["hours_old"]
    if site == "seek":
        # Seek's date filter counts calendar days, and one day means "listed today",
        # which drops anything listed yesterday after this run's hour. Ask for two
        # days; within_window trims the extra back off.
        hours = max(hours, 48)

    label = "%s '%s'" % (site, term)
    try:
        body = http_request(
            cfg["mcp_url"].rstrip("/") + "/api",
            data={
                "siteNames": site,
                "searchTerm": term,
                "location": cfg["location"],
                "hoursOld": hours,
                "resultsWanted": cfg.get("results_per_search", 100),
                "countryIndeed": cfg.get("country_indeed", "australia"),
                "timeout": 300000,
            },
            headers={"Content-Type": "application/json", "Accept": "application/json"},
            timeout=330,
        )
    except (TimeoutError, socket.timeout) as exc:
        raise BoardUnavailable("%s timed out" % label) from exc
    except urllib.error.HTTPError as exc:
        warnings.append("%s failed: HTTP %s" % (label, exc.code))
        return []
    except (urllib.error.URLError, OSError) as exc:
        if isinstance(getattr(exc, "reason", None), (TimeoutError, socket.timeout)):
            raise BoardUnavailable("%s timed out" % label) from exc
        warnings.append("%s failed: %s" % (label, exc))
        return []
    try:
        return json.loads(body).get("jobs") or []
    except ValueError:
        warnings.append("%s returned invalid JSON" % label)
        return []


def linkedin_apply_type(job_id, warnings):
    """Classify how a LinkedIn posting is applied to.

    The guest posting page marks the apply button's destination: `apply-link-onsite`
    for LinkedIn's own Easy Apply flow, `apply-link-offsite` when it hands off to an
    external ATS. Some postings render neither, hence the unknown case.
    """
    html = request_with_retry(
        "%s/%s" % (LINKEDIN_POSTING_URL, job_id),
        "apply-type for linkedin %s" % job_id,
        warnings,
        headers={"Accept": "text/html,application/xhtml+xml"},
    )
    if html is None:
        return APPLY_UNKNOWN
    if "apply-link-onsite" in html:
        return APPLY_EASY
    if "apply-link-offsite" in html:
        return APPLY_EXTERNAL
    return APPLY_UNKNOWN


def seek_apply_type(job_id, warnings):
    """Classify how a Seek posting is applied to.

    Seek's job details carry `isLinkOut`: true for the "Apply" button that leaves for
    the employer's site, false for Seek's own Quick apply. The search results do not
    include it, so it costs one small GraphQL call per posting.
    """
    body = request_with_retry(
        SEEK_GRAPHQL_URL,
        "apply-type for seek %s" % job_id,
        warnings,
        data={
            "operationName": "jobDetails",
            "variables": {"jobId": job_id},
            "query": "query jobDetails($jobId: ID!) { jobDetails(id: $jobId) { job { isLinkOut } } }",
        },
        headers={
            "Content-Type": "application/json",
            "Accept": "application/json",
            "seek-request-brand": "seek",
            "seek-request-country": "AU",
        },
    )
    try:
        is_link_out = json.loads(body)["data"]["jobDetails"]["job"]["isLinkOut"]
    except (TypeError, ValueError, KeyError):
        return APPLY_UNKNOWN
    if is_link_out is None:
        return APPLY_UNKNOWN
    return APPLY_EXTERNAL if is_link_out else APPLY_EASY


def board_id(site, jobspy_id):
    """The board's own posting id, from JobSpy's prefixed one."""
    prefix = SITES[site]["jobspy_prefix"]
    return jobspy_id[len(prefix):] if jobspy_id.startswith(prefix) else jobspy_id


def apply_type_for(site, posting_id, jobspy_apply_type, warnings):
    if site == "linkedin":
        return linkedin_apply_type(posting_id, warnings)
    if site == "seek":
        return seek_apply_type(posting_id, warnings)
    # Indeed's search results already say (the fork's apply_type field).
    if jobspy_apply_type in (APPLY_EASY, APPLY_EXTERNAL):
        return jobspy_apply_type
    return APPLY_UNKNOWN


# --------------------------------------------------------------------------- filter


def within_window(posted, hours_old):
    """Boards expose a date, not a time, so compare on whole days."""
    if not posted:
        return True  # the board's own date filter already constrained the query
    try:
        posted_date = datetime.strptime(posted[:10], "%Y-%m-%d").date()
    except ValueError:
        return True
    days = max(1, -(-hours_old // 24))
    cutoff = (datetime.now().astimezone() - timedelta(days=days)).date()
    return posted_date >= cutoff


# -------------------------------------------------------------------------- tracker


def tracker_token(cfg):
    import jwt  # imported lazily so --dry-run works without PyJWT installed

    now = int(time.time())
    return jwt.encode(
        {
            "user_id": cfg["tracker_user_id"],
            "email": cfg["tracker_email"],
            "iat": now,
            "exp": now + 3600,
        },
        cfg["jwt_secret"],
        algorithm="HS256",
    )


def post_jobs(cfg, jobs):
    """Send the batch to the tracker. Returns its ingest result."""
    payload = json.dumps({"jobs": jobs}).encode()
    req = urllib.request.Request(
        cfg["tracker_url"].rstrip("/") + "/api/discovered-jobs",
        data=payload,
        method="POST",
        headers={
            "Authorization": "Bearer %s" % tracker_token(cfg),
            "Content-Type": "application/json",
        },
    )
    with urllib.request.urlopen(req, timeout=30) as resp:
        body = json.load(resp)
    return body.get("data", {})


# --------------------------------------------------------------------------- output


def format_digest(jobs, counts, warnings, cfg, run_time):
    """Human-readable summary for the job log."""
    lines = [
        "Daily board scrape - %s" % run_time.strftime("%Y-%m-%d %H:%M %Z"),
        "%s | last %d hours | boards: %s"
        % (cfg["location"], cfg["hours_old"], ", ".join(cfg["sites"])),
        "",
        "  %-28s %s" % ("", "  ".join("%9s" % site for site in cfg["sites"])),
    ]
    for term in cfg["search_terms"]:
        lines.append(
            "  %-28s %s"
            % (term, "  ".join("%9d" % counts.get((site, term), 0) for site in cfg["sites"]))
        )
    lines.append("")

    if warnings:
        lines.append("Warnings:")
        lines.extend("  - %s" % warning for warning in warnings)
        lines.append("")

    lines.append("Matched %d posting(s):" % len(jobs))
    for job in jobs:
        marker = {
            APPLY_EASY: "[easy apply]",
            APPLY_EXTERNAL: "[external]  ",
            APPLY_UNKNOWN: "[unknown]   ",
        }[job["apply_type"]]
        lines.append("  %s %-9s %s" % (marker, job["source"], job["job_title"]))
        lines.append("      %s | %s" % (job["company_name"], job["job_url"]))

    return "\n".join(lines)


# ----------------------------------------------------------------------------- main


def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--config", default=os.path.join(SCRIPT_DIR, "config.json"))
    ap.add_argument("--hours", type=int, help="override hours_old from config")
    ap.add_argument("--sites", help="comma-separated boards, overriding config (e.g. seek,indeed)")
    ap.add_argument(
        "--dry-run",
        action="store_true",
        help="scrape and print the digest without posting to the tracker",
    )
    ap.add_argument("--verbose", action="store_true", help="per-search progress on stderr")
    args = ap.parse_args()

    try:
        cfg = load_config(args.config)
    except (OSError, ValueError) as exc:
        print("FATAL: cannot read config %s: %s" % (args.config, exc), file=sys.stderr)
        return 2

    if args.hours:
        cfg["hours_old"] = args.hours
    if args.sites:
        cfg["sites"] = [s.strip() for s in args.sites.split(",") if s.strip()]
    unknown_sites = [site for site in cfg["sites"] if site not in SITES]
    if unknown_sites:
        print("FATAL: unsupported board(s): %s" % ", ".join(unknown_sites), file=sys.stderr)
        return 2
    # Checked before scraping so a misconfigured run fails in seconds, not after it.
    missing = [
        key for key in ("tracker_url", "tracker_user_id", "tracker_email", "jwt_secret")
        if not cfg.get(key)
    ]
    if missing and not args.dry_run:
        print("FATAL: missing tracker settings: %s" % ", ".join(missing), file=sys.stderr)
        return 2

    run_time = datetime.now().astimezone()
    warnings = []
    include_re = re.compile(cfg["title_include"], re.I)
    exclude_re = re.compile(cfg["title_exclude"], re.I) if cfg.get("title_exclude") else None

    # 1. search every term on every board
    counts = {}
    candidates = {}
    for site in cfg["sites"]:
        for term in cfg["search_terms"]:
            try:
                results = search_board(cfg, site, term, warnings)
            except BoardUnavailable as exc:
                warnings.append("%s; skipping %s's remaining searches." % (exc, site))
                break
            counts[(site, term)] = len(results)
            if args.verbose:
                print("  %s '%s' -> %d" % (site, term, len(results)), file=sys.stderr)

            for result in results:
                posting_id = board_id(site, result.get("id") or "")
                title = result.get("title") or ""
                if not posting_id or not title or (site, posting_id) in candidates:
                    continue
                if not include_re.search(title):
                    continue
                if exclude_re and exclude_re.search(title):
                    continue
                if not within_window(result.get("datePosted") or "", cfg["hours_old"]):
                    continue
                candidates[(site, posting_id)] = result

            time.sleep(random.uniform(3.0, 6.0))

        if not any(counts.get((site, term)) for term in cfg["search_terms"]):
            warnings.append(
                "%s returned nothing for any term - it is likely throttling or its "
                "scraper is broken; today's feed is missing it." % site
            )

    if not any(counts.values()):
        print(
            "FATAL: every search returned zero results (%s)"
            % ("; ".join(warnings) or "no errors reported"),
            file=sys.stderr,
        )
        return 1

    # 2. classify how each survivor is applied to
    jobs = []
    for (site, posting_id), result in candidates.items():
        apply_type = apply_type_for(site, posting_id, result.get("applyType"), warnings)
        jobs.append({
            "external_id": SITES[site]["external_prefix"] + posting_id,
            "job_title": result.get("title") or "",
            "company_name": result.get("company") or "",
            "location": result.get("location") or "",
            "job_url": result.get("jobUrl") or "",
            "apply_url": result.get("jobUrlDirect") or "",
            "source": site,
            "posted_date": (result.get("datePosted") or "")[:10],
            "apply_type": apply_type,
        })
        if site != "indeed":
            time.sleep(random.uniform(1.5, 3.0))

    jobs.sort(key=lambda j: (j["company_name"].lower(), j["job_title"].lower(), j["source"]))

    print(format_digest(jobs, counts, warnings, cfg, run_time))

    # 3. hand off to the tracker, which de-duplicates and prunes
    if args.dry_run:
        print("\nDry run: nothing posted to the tracker.")
        return 0

    try:
        result = post_jobs(cfg, jobs)
    except Exception as exc:  # noqa: BLE001 - any failure here must be loud
        print(
            "FATAL: posting to tracker failed (%s: %s)" % (type(exc).__name__, exc),
            file=sys.stderr,
        )
        return 1

    print(
        "\nTracker: received=%s created=%s updated=%s pruned=%s"
        % (
            result.get("received", 0),
            result.get("created", 0),
            result.get("updated", 0),
            result.get("pruned", 0),
        )
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
