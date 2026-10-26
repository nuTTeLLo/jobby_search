import { isLowFit } from '../constants/fit';

const APPLY_BADGES = {
  easy_apply: { label: 'Easy Apply', color: '#198754' },
  external: { label: 'External', color: '#6c757d' },
  unknown: { label: 'Apply type unknown', color: '#adb5bd' },
};

// Which job board a scraper found the posting on. Anything unlisted still gets a
// neutral pill showing the raw source, so a new scraper is visible before it is styled.
const SOURCE_BADGES = {
  linkedin: { label: 'LinkedIn', color: '#0a66c2' },
  seek: { label: 'Seek', color: '#e60278' },
  indeed: { label: 'Indeed', color: '#2164f3' },
  glassdoor: { label: 'Glassdoor', color: '#0caa41' },
};

function sourceBadge(source) {
  const key = (source || '').toLowerCase();
  return SOURCE_BADGES[key] || { label: source || 'Unknown source', color: '#495057' };
}

// The daily scrape scores each posting 1-5 against the fit profile. Unscored
// postings (no profile, or scoring failed) show no badge.
const FIT_BADGES = {
  5: { label: 'Strong fit', color: '#146c43' },
  4: { label: 'Good fit', color: '#198754' },
  3: { label: 'Possible fit', color: '#cc8a00' },
  2: { label: 'Weak fit', color: '#868e96' },
  1: { label: 'Poor fit', color: '#adb5bd' },
};

// Higher scores first; unscored postings after scored ones, since nothing is known
// about them either way.
const byFit = (a, b) => (b.fit_score ?? 0) - (a.fit_score ?? 0);

const FIT_FIELDS = [
  'fit_score', 'fit_summary', 'fit_strengths', 'fit_gaps', 'fit_flags', 'fit_model', 'fit_scored_at',
];

// Within a day, postings are ordered by how much attention they deserve: companies we
// have applied to before come first whatever their apply type, then fresh external
// applications, then easy-apply postings. Inside the applied-before tier the easy-apply
// postings sink to the bottom, so the ones worth a real application read first. Fit
// orders postings within each tier.
const isEasyApply = (job) => job.apply_type === 'easy_apply';

const TIERS = [
  {
    key: 'applied_before',
    label: 'Applied here before',
    match: (job) => job.applied_before,
    sortWithin: (a, b) => isEasyApply(a) - isEasyApply(b) || byFit(a, b),
  },
  { key: 'new', label: 'New application', match: (job) => !isEasyApply(job), sortWithin: byFit },
  { key: 'easy_apply', label: 'Easy apply', match: () => true, sortWithin: byFit },
];

// The board whose copy of a merged posting the title links to, best first. Every
// board's copy stays one click away on its own pill.
const BOARD_PRIORITY = ['linkedin', 'seek', 'indeed'];

function boardRank(source) {
  const rank = BOARD_PRIORITY.indexOf((source || '').toLowerCase());
  return rank === -1 ? BOARD_PRIORITY.length : rank;
}

// The easiest way in wins: if any board offers Easy/Quick apply, the role is easy apply.
const APPLY_RANK = { easy_apply: 2, external: 1, unknown: 0 };

// The same role scraped from several boards arrives as one row per board sharing a
// match_key. Fold them into one posting that remembers each board's link, and file it
// under the day it was first seen on any board so a re-post elsewhere does not
// resurface it as new.
function mergeDuplicates(jobs) {
  const groups = new Map();
  for (const job of jobs) {
    const key = job.match_key || job.id;
    const group = groups.get(key);
    if (!group) {
      groups.set(key, { ...job, ids: [job.id], postings: [job] });
      continue;
    }
    group.ids.push(job.id);
    group.postings.push(job);
    group.applied_before = group.applied_before || job.applied_before;
    if (!group.last_applied_at || new Date(job.last_applied_at) > new Date(group.last_applied_at)) {
      group.last_applied_at = job.last_applied_at || group.last_applied_at;
    }
    group.apply_url = group.apply_url || job.apply_url;
    if ((APPLY_RANK[job.apply_type] || 0) > (APPLY_RANK[group.apply_type] || 0)) {
      group.apply_type = job.apply_type;
    }
    if (new Date(job.discovered_at) < new Date(group.discovered_at)) {
      group.discovered_at = job.discovered_at;
    }
    // Each board's copy is scored separately; show the most favourable assessment.
    if ((job.fit_score ?? 0) > (group.fit_score ?? 0)) {
      for (const field of FIT_FIELDS) group[field] = job[field];
    }
  }
  // Show and link the preferred board's copy; its pill comes first too.
  for (const group of groups.values()) {
    group.postings.sort((a, b) => boardRank(a.source) - boardRank(b.source));
    const preferred = group.postings[0];
    group.job_url = preferred.job_url;
    group.job_title = preferred.job_title;
    group.company_name = preferred.company_name;
    group.location = preferred.location;
  }
  // Moving a group to its earliest day can break the DESC order the backend sent.
  return [...groups.values()].sort(
    (a, b) => new Date(b.discovered_at) - new Date(a.discovered_at),
  );
}

// One pill per board, linking to that board's copy of the posting.
function boardLinks(postings) {
  const seen = new Map();
  for (const posting of postings) {
    const key = (posting.source || '').toLowerCase();
    if (!seen.has(key)) seen.set(key, posting);
  }
  return [...seen.values()];
}

function tierIndex(job) {
  return TIERS.findIndex((tier) => tier.match(job));
}

// Group by the day a posting was first seen, newest day first, then into tiers within
// the day. mergeDuplicates returns postings in discovered_at DESC order, so insertion
// order gives us both the day grouping and the ordering inside each tier for free.
function groupByDay(groups) {
  const days = new Map();
  for (const job of groups) {
    const day = (job.discovered_at || '').slice(0, 10);
    if (!days.has(day)) days.set(day, TIERS.map(() => []));
    days.get(day)[tierIndex(job)].push(job);
  }

  return [...days.entries()].map(([day, buckets]) => ({
    day,
    count: buckets.reduce((total, bucket) => total + bucket.length, 0),
    tiers: TIERS.map((tier, i) => ({
      ...tier,
      // Array#sort is stable, so discovered_at DESC still breaks ties.
      jobs: tier.sortWithin ? buckets[i].sort(tier.sortWithin) : buckets[i],
    })).filter((tier) => tier.jobs.length > 0),
  }));
}

function formatDay(day) {
  if (!day) return 'Unknown date';
  const date = new Date(day + 'T00:00:00');
  if (Number.isNaN(date.getTime())) return day;

  const today = new Date();
  const isSameDay = (a, b) => a.toDateString() === b.toDateString();
  const yesterday = new Date(today);
  yesterday.setDate(today.getDate() - 1);

  const formatted = date.toLocaleDateString(undefined, {
    weekday: 'long',
    day: 'numeric',
    month: 'short',
  });
  if (isSameDay(date, today)) return `Today · ${formatted}`;
  if (isSameDay(date, yesterday)) return `Yesterday · ${formatted}`;
  return formatted;
}

function formatAppliedDate(timestamp) {
  const date = new Date(timestamp);
  const formatted = date.toLocaleDateString(undefined, {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  });

  // Whole calendar days, so an application yesterday evening reads "1 day". Counted
  // from local date parts, so a daylight-saving change cannot skew the difference.
  const dayNumber = (d) => Date.UTC(d.getFullYear(), d.getMonth(), d.getDate()) / 86400000;
  const days = dayNumber(new Date()) - dayNumber(date);
  const ago = days <= 0 ? 'today' : days === 1 ? '1 day' : `${days} days`;
  return `${formatted} (${ago})`;
}

// The model's reasoning behind a score, folded away so the list stays scannable.
function FitDetails({ job }) {
  const sections = [
    { label: 'Strengths', items: job.fit_strengths },
    { label: 'Gaps', items: job.fit_gaps },
  ].filter((section) => section.items?.length);
  if (!sections.length) return null;

  return (
    <details style={styles.fitDetails}>
      <summary style={styles.fitDetailsToggle}>Why this score</summary>
      {sections.map((section) => (
        <div key={section.label} style={styles.fitSection}>
          <span style={styles.fitSectionLabel}>{section.label}</span>
          <ul style={styles.fitList}>
            {section.items.map((item) => <li key={item}>{item}</li>)}
          </ul>
        </div>
      ))}
    </details>
  );
}

export default function DiscoveredList({ jobs, onDismiss, loading, hideLowFit = false }) {
  if (loading) {
    return <div style={styles.empty}>Loading...</div>;
  }

  if (!jobs.length) {
    return (
      <div style={styles.empty}>
        Nothing discovered in the last 7 days. The scrape runs each morning at 9am Melbourne time.
      </div>
    );
  }

  const groups = mergeDuplicates(jobs);
  const visible = hideLowFit ? groups.filter((job) => !isLowFit(job)) : groups;

  if (!visible.length) {
    return (
      <div style={styles.empty}>
        Every posting from the last 7 days is a low fit. Untick &ldquo;Hide low fits&rdquo; to see them.
      </div>
    );
  }

  return (
    <div>
      {groupByDay(visible).map(({ day, count, tiers }) => (
        <section key={day} style={styles.daySection}>
          <h3 style={styles.dayHeading}>
            {formatDay(day)}
            <span style={styles.dayCount}>{count}</span>
          </h3>

          {tiers.map((tier) => (
            <div key={tier.key} style={styles.tierSection}>
              <h4 style={styles.tierHeading}>
                {tier.label}
                <span style={styles.tierCount}>{tier.jobs.length}</span>
              </h4>

              {tier.jobs.map((job) => {
                const badge = APPLY_BADGES[job.apply_type] || APPLY_BADGES.unknown;
                const fit = FIT_BADGES[job.fit_score];
                return (
                  <div key={job.id} style={styles.row}>
                    <div style={styles.rowMain}>
                      <a
                        href={job.job_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        style={styles.jobTitle}
                      >
                        {job.job_title}
                      </a>
                      <div style={styles.subline}>
                        {job.company_name}
                        {job.location ? ` · ${job.location}` : ''}
                      </div>
                      {fit && (
                        <div style={styles.fitBlock}>
                          {job.fit_summary && <div style={styles.fitSummary}>{job.fit_summary}</div>}
                          {job.fit_flags?.length > 0 && (
                            <div style={styles.fitFlags}>
                              {job.fit_flags.map((flag) => (
                                <span key={flag} style={styles.fitFlag}>⚠ {flag}</span>
                              ))}
                            </div>
                          )}
                          <FitDetails job={job} />
                        </div>
                      )}
                    </div>

                    <div style={styles.badges}>
                      {fit && (
                        <span
                          style={{ ...styles.badge, backgroundColor: fit.color }}
                          title={job.fit_model ? `Scored by ${job.fit_model}` : undefined}
                        >
                          {fit.label} · {job.fit_score}/5
                        </span>
                      )}
                      {boardLinks(job.postings).map((posting) => {
                        const source = sourceBadge(posting.source);
                        return (
                          <a
                            key={posting.id}
                            href={posting.job_url}
                            target="_blank"
                            rel="noopener noreferrer"
                            style={{ ...styles.badge, ...styles.link, backgroundColor: source.color }}
                            title={`Open on ${source.label}`}
                          >
                            {source.label} ↗
                          </a>
                        );
                      })}
                      {job.apply_url && (
                        <a
                          href={job.apply_url}
                          target="_blank"
                          rel="noopener noreferrer"
                          style={styles.employerLink}
                          title="Apply on the employer's own site"
                        >
                          Employer site ↗
                        </a>
                      )}
                      <span style={{ ...styles.badge, backgroundColor: badge.color }}>
                        {badge.label}
                      </span>
                      {job.applied_before && (
                        <span style={{ ...styles.badge, backgroundColor: '#fd7e14' }}>
                          {job.last_applied_at
                            ? `Last Applied ${formatAppliedDate(job.last_applied_at)}`
                            : 'Applied here before'}
                        </span>
                      )}
                      <button
                        onClick={() => onDismiss(job.ids)}
                        style={styles.dismissBtn}
                        title="Hide this posting"
                      >
                        Dismiss
                      </button>
                    </div>
                  </div>
                );
              })}
            </div>
          ))}
        </section>
      ))}
    </div>
  );
}

const styles = {
  empty: {
    padding: '40px 20px',
    textAlign: 'center',
    color: '#6c757d',
    backgroundColor: 'white',
    borderRadius: '8px',
  },
  daySection: {
    marginBottom: '24px',
  },
  dayHeading: {
    display: 'flex',
    alignItems: 'center',
    gap: '10px',
    fontSize: '14px',
    textTransform: 'uppercase',
    letterSpacing: '0.05em',
    color: '#6c757d',
    margin: '0 0 8px 0',
  },
  tierSection: {
    marginBottom: '14px',
  },
  tierHeading: {
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    fontSize: '12px',
    fontWeight: 600,
    color: '#868e96',
    margin: '0 0 6px 2px',
  },
  tierCount: {
    color: '#adb5bd',
    fontWeight: 500,
  },
  dayCount: {
    backgroundColor: '#e9ecef',
    color: '#495057',
    borderRadius: '10px',
    padding: '1px 8px',
    fontSize: '12px',
    letterSpacing: 0,
  },
  row: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    gap: '16px',
    flexWrap: 'wrap',
    backgroundColor: 'white',
    borderRadius: '8px',
    padding: '12px 16px',
    marginBottom: '6px',
  },
  rowMain: {
    minWidth: '260px',
    flex: 1,
  },
  jobTitle: {
    color: '#0d6efd',
    fontWeight: 600,
    textDecoration: 'none',
    fontSize: '15px',
  },
  subline: {
    color: '#6c757d',
    fontSize: '13px',
    marginTop: '2px',
  },
  // Pinned to the right edge, even when the row wraps it onto its own line or its
  // items wrap among themselves, so Dismiss always lands in the same spot.
  badges: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'flex-end',
    gap: '8px',
    flexWrap: 'wrap',
    marginLeft: 'auto',
  },
  badge: {
    color: 'white',
    padding: '4px 12px',
    borderRadius: '12px',
    fontSize: '12px',
    fontWeight: 500,
    display: 'inline-block',
  },
  link: {
    textDecoration: 'none',
  },
  employerLink: {
    border: '1px solid #dee2e6',
    color: '#0d6efd',
    borderRadius: '12px',
    padding: '3px 12px',
    fontSize: '12px',
    textDecoration: 'none',
  },
  fitBlock: {
    marginTop: '6px',
  },
  fitSummary: {
    color: '#495057',
    fontSize: '13px',
    lineHeight: 1.4,
  },
  fitFlags: {
    display: 'flex',
    flexWrap: 'wrap',
    gap: '6px',
    marginTop: '4px',
  },
  fitFlag: {
    backgroundColor: '#f8d7da',
    color: '#842029',
    borderRadius: '10px',
    padding: '2px 10px',
    fontSize: '12px',
  },
  fitDetails: {
    marginTop: '4px',
    fontSize: '13px',
    color: '#495057',
  },
  fitDetailsToggle: {
    cursor: 'pointer',
    color: '#6c757d',
    fontSize: '12px',
  },
  fitSection: {
    marginTop: '4px',
  },
  fitSectionLabel: {
    fontWeight: 600,
    fontSize: '12px',
    color: '#495057',
  },
  fitList: {
    margin: '2px 0 0 0',
    paddingLeft: '20px',
  },
  dismissBtn: {
    border: '1px solid #dee2e6',
    backgroundColor: 'transparent',
    color: '#6c757d',
    borderRadius: '12px',
    padding: '4px 12px',
    fontSize: '12px',
    cursor: 'pointer',
  },
};
