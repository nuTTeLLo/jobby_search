#!/usr/bin/env python3
"""Daily LinkedIn scrape -> job tracker "Discovered" feed.

Queries LinkedIn's public (guest) job-search endpoint for roles posted in the last
24 hours, filters to the role families being targeted, classifies each surviving
posting as Easy Apply or external, and POSTs the batch to the job tracker.

The tracker owns de-duplication and retention: rows are upserted on
(user_id, external_id) and pruned to a rolling window, so this script keeps no
local state and a re-run is harmless.

Stdlib only, plus PyJWT to mint the tracker token.
"""

import argparse
import json
import os
import random
import re
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from datetime import datetime, timedelta
from html.parser import HTMLParser

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
GUEST_SEARCH_URL = (
    "https://www.linkedin.com/jobs-guest/jobs/api/seeMoreJobPostings/search"
)
GUEST_POSTING_URL = "https://www.linkedin.com/jobs-guest/jobs/api/jobPosting"
USER_AGENT = (
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 "
    "(KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
)
PAGE_SIZE = 10
SOURCE = "linkedin"

APPLY_EASY = "easy_apply"
APPLY_EXTERNAL = "external"
APPLY_UNKNOWN = "unknown"


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
        ("SEARCH_LOCATION", "location"),
    ):
        value = os.environ.get(env_key)
        if value:
            cfg[cfg_key] = value

    if os.environ.get("HOURS_OLD"):
        cfg["hours_old"] = int(os.environ["HOURS_OLD"])

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


# ---------------------------------------------------------------------------- parse


class JobCardParser(HTMLParser):
    """Extracts job cards from the guest endpoint's HTML fragment."""

    FIELD_CLASSES = {
        "base-search-card__title": "title",
        "base-search-card__subtitle": "company",
        "job-search-card__location": "location",
    }

    def __init__(self):
        super().__init__(convert_charrefs=True)
        self.cards = []
        self._card = None
        self._field = None
        self._field_tag = None
        self._buf = []

    def _flush_card(self):
        if self._card and self._card.get("id"):
            self.cards.append(self._card)
        self._card = None

    def handle_starttag(self, tag, attrs):
        attrs = dict(attrs)
        urn = attrs.get("data-entity-urn", "")
        if urn.startswith("urn:li:jobPosting:"):
            self._flush_card()
            self._card = {
                "id": urn.rsplit(":", 1)[-1],
                "title": "",
                "company": "",
                "location": "",
                "posted": "",
            }
            return

        if self._card is None:
            return

        if tag == "time" and not self._card["posted"]:
            self._card["posted"] = attrs.get("datetime", "").strip()
            return

        if self._field is not None:
            return

        for cls in attrs.get("class", "").split():
            field = self.FIELD_CLASSES.get(cls)
            if field and not self._card[field]:
                self._field = field
                self._field_tag = tag
                self._buf = []
                return

    def handle_data(self, data):
        if self._field is not None:
            self._buf.append(data)

    def handle_endtag(self, tag):
        if self._field is not None and tag == self._field_tag:
            self._card[self._field] = " ".join("".join(self._buf).split())
            self._field = None
            self._field_tag = None
            self._buf = []

    def close(self):
        super().close()
        self._flush_card()


def parse_cards(html):
    parser = JobCardParser()
    parser.feed(html)
    parser.close()
    return parser.cards


# ---------------------------------------------------------------------------- fetch


def http_get(url, timeout=25):
    req = urllib.request.Request(
        url,
        headers={
            "User-Agent": USER_AGENT,
            "Accept": "text/html,application/xhtml+xml",
            "Accept-Language": "en-AU,en;q=0.9",
        },
    )
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        return resp.read().decode("utf-8", errors="replace")


def get_with_retry(url, label, warnings):
    """Fetch, retrying transport errors. Returns the body or None."""
    for attempt in range(3):
        try:
            return http_get(url)
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


def search_page_url(search_term, location, start):
    return GUEST_SEARCH_URL + "?" + urllib.parse.urlencode({
        "keywords": search_term,
        "location": location,
        "f_TPR": "r86400",
        "start": str(start),
    })


def fetch_term(search_term, location, max_pages, warnings, verbose=False):
    """Page through one search term. Returns a list of raw cards."""
    cards = []
    for page in range(max_pages):
        start = page * PAGE_SIZE
        label = "'%s' start=%d" % (search_term, start)
        html = get_with_retry(search_page_url(search_term, location, start), label, warnings)
        if html is None:
            break

        page_cards = parse_cards(html)
        # LinkedIn soft-throttles by serving an empty (HTTP 200) page; on the first
        # page that is indistinguishable from "no results", so back off and re-ask.
        for backoff in (8, 20):
            if page_cards or page != 0:
                break
            time.sleep(backoff + random.uniform(0, 4))
            retry_html = get_with_retry(search_page_url(search_term, location, start), label, warnings)
            if retry_html is not None:
                page_cards = parse_cards(retry_html)

        if not page_cards and page == 0:
            warnings.append(
                "'%s' returned no cards after retries - LinkedIn is likely "
                "throttling; this term may be under-reported today." % search_term
            )

        if verbose:
            print("  %s -> %d cards" % (label, len(page_cards)), file=sys.stderr)
        if not page_cards:
            break
        cards.extend(page_cards)
        time.sleep(random.uniform(2.0, 4.0))

    return cards


def detect_apply_type(job_id, warnings):
    """Classify how a posting is applied to.

    The guest posting page marks the apply button's destination: `apply-link-onsite`
    for LinkedIn's own Easy Apply flow, `apply-link-offsite` when it hands off to an
    external ATS. Some postings render neither, hence the unknown case.
    """
    html = get_with_retry(
        "%s/%s" % (GUEST_POSTING_URL, job_id), "apply-type for %s" % job_id, warnings
    )
    if html is None:
        return APPLY_UNKNOWN
    if "apply-link-onsite" in html:
        return APPLY_EASY
    if "apply-link-offsite" in html:
        return APPLY_EXTERNAL
    return APPLY_UNKNOWN


# --------------------------------------------------------------------------- filter


def within_window(posted, hours_old):
    """The guest endpoint only exposes a date, so compare on whole days."""
    if not posted:
        return True  # f_TPR already constrained the query; keep undated cards
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


def format_digest(jobs, per_term_counts, warnings, cfg, run_time):
    """Human-readable summary for the job log."""
    lines = [
        "LinkedIn daily scrape - %s" % run_time.strftime("%Y-%m-%d %H:%M %Z"),
        "%s | last %d hours | terms: %s"
        % (cfg["location"], cfg["hours_old"], ", ".join(cfg["search_terms"])),
        "",
    ]
    for term in cfg["search_terms"]:
        lines.append("  %-28s %d cards" % (term, per_term_counts.get(term, 0)))
    lines.append("")

    if warnings:
        lines.append("Warnings:")
        lines.extend("  - %s" % warning for warning in warnings)
        lines.append("")

    lines.append("Matched %d job(s):" % len(jobs))
    for job in jobs:
        marker = {
            APPLY_EASY: "[easy apply]",
            APPLY_EXTERNAL: "[external]  ",
            APPLY_UNKNOWN: "[unknown]   ",
        }[job["apply_type"]]
        lines.append("  %s %s" % (marker, job["job_title"]))
        lines.append("      %s | %s" % (job["company_name"], job["job_url"]))

    return "\n".join(lines)


# ----------------------------------------------------------------------------- main


def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--config", default=os.path.join(SCRIPT_DIR, "config.json"))
    ap.add_argument("--hours", type=int, help="override hours_old from config")
    ap.add_argument(
        "--dry-run",
        action="store_true",
        help="scrape and print the digest without posting to the tracker",
    )
    ap.add_argument("--verbose", action="store_true", help="per-page progress on stderr")
    args = ap.parse_args()

    try:
        cfg = load_config(args.config)
    except (OSError, ValueError) as exc:
        print("FATAL: cannot read config %s: %s" % (args.config, exc), file=sys.stderr)
        return 2

    if args.hours:
        cfg["hours_old"] = args.hours

    run_time = datetime.now().astimezone()
    warnings = []
    include_re = re.compile(cfg["title_include"], re.I)
    exclude_re = re.compile(cfg["title_exclude"], re.I) if cfg.get("title_exclude") else None

    # 1. scrape each term
    per_term_counts = {}
    candidates = {}
    for term in cfg["search_terms"]:
        cards = fetch_term(
            term,
            cfg["location"],
            cfg.get("max_pages_per_term", 10),
            warnings,
            verbose=args.verbose,
        )
        per_term_counts[term] = len(cards)

        for card in cards:
            job_id = card["id"]
            if job_id in candidates:
                continue
            if not include_re.search(card["title"]):
                continue
            if exclude_re and exclude_re.search(card["title"]):
                continue
            if not within_window(card["posted"], cfg["hours_old"]):
                continue
            candidates[job_id] = card

        time.sleep(random.uniform(5.0, 9.0))

    if not any(per_term_counts.values()):
        print(
            "FATAL: every search term returned zero cards (%s)"
            % ("; ".join(warnings) or "no HTTP errors reported"),
            file=sys.stderr,
        )
        return 1

    # 2. classify how each survivor is applied to
    jobs = []
    for job_id, card in candidates.items():
        apply_type = detect_apply_type(job_id, warnings)
        jobs.append({
            "external_id": job_id,
            "job_title": card["title"],
            "company_name": card["company"],
            "location": card["location"],
            "job_url": "https://www.linkedin.com/jobs/view/%s/" % job_id,
            "source": SOURCE,
            "posted_date": card["posted"],
            "apply_type": apply_type,
        })
        time.sleep(random.uniform(1.5, 3.0))

    jobs.sort(key=lambda j: (j["company_name"].lower(), j["job_title"].lower()))

    print(format_digest(jobs, per_term_counts, warnings, cfg, run_time))

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
