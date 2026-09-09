const APPLY_BADGES = {
  easy_apply: { label: 'Easy Apply', color: '#198754' },
  external: { label: 'External', color: '#6c757d' },
  unknown: { label: 'Apply type unknown', color: '#adb5bd' },
};

// Within a day, postings are ordered by how much attention they deserve: companies we
// have applied to before come first whatever their apply type, then fresh external
// applications, then easy-apply postings. Inside the applied-before tier the easy-apply
// postings sink to the bottom, so the ones worth a real application read first.
const isEasyApply = (job) => job.apply_type === 'easy_apply';

const TIERS = [
  {
    key: 'applied_before',
    label: 'Applied here before',
    match: (job) => job.applied_before,
    sortWithin: (a, b) => isEasyApply(a) - isEasyApply(b),
  },
  { key: 'new', label: 'New application', match: (job) => !isEasyApply(job) },
  { key: 'easy_apply', label: 'Easy apply', match: () => true },
];

function tierIndex(job) {
  return TIERS.findIndex((tier) => tier.match(job));
}

// Group by the day a posting was first seen, newest day first, then into tiers within
// the day. The backend already orders rows by discovered_at DESC, so insertion order
// gives us both the day grouping and the ordering inside each tier for free.
function groupByDay(jobs) {
  const days = new Map();
  for (const job of jobs) {
    const day = (job.discovered_at || '').slice(0, 10);
    if (!days.has(day)) days.set(day, TIERS.map(() => []));
    days.get(day)[tierIndex(job)].push(job);
  }

  return [...days.entries()].map(([day, buckets]) => ({
    day,
    count: buckets.reduce((total, bucket) => total + bucket.length, 0),
    tiers: TIERS.map((tier, i) => ({
      ...tier,
      // Array#sort is stable, so discovered_at DESC still holds within each half.
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

export default function DiscoveredList({ jobs, onDismiss, loading }) {
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

  return (
    <div>
      {groupByDay(jobs).map(({ day, count, tiers }) => (
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
                    </div>

                    <div style={styles.badges}>
                      <span style={{ ...styles.badge, backgroundColor: badge.color }}>
                        {badge.label}
                      </span>
                      {job.applied_before && (
                        <span style={{ ...styles.badge, backgroundColor: '#fd7e14' }}>
                          Applied here before
                        </span>
                      )}
                      <button
                        onClick={() => onDismiss(job.id)}
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
  badges: {
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    flexWrap: 'wrap',
  },
  badge: {
    color: 'white',
    padding: '4px 12px',
    borderRadius: '12px',
    fontSize: '12px',
    fontWeight: 500,
    display: 'inline-block',
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
