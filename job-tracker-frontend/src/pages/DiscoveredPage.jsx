import { useEffect, useRef, useState } from 'react';
import AppHeader from '../components/AppHeader';
import DiscoveredList from '../components/DiscoveredList';
import { getDiscoveredJobs, dismissDiscoveredJob } from '../services/api';

// Read-only feed of postings found by the daily job board scrapes. Deliberately has
// no "add to tracker" action: application status is tracked on the boards themselves,
// and scraped rows never enter the jobs table.
export default function DiscoveredPage() {
  const [jobs, setJobs] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  // Rows dismissed in this session, including ones whose request is still in flight.
  // Every list we receive is filtered through it, so a reload that races a dismiss
  // cannot bring back a row the user has just hidden.
  const hiddenIds = useRef(new Set());

  useEffect(() => {
    fetchDiscovered();
  }, []);

  const fetchDiscovered = async ({ quiet = false } = {}) => {
    if (!quiet) setLoading(true);
    try {
      const fetched = await getDiscoveredJobs();
      setJobs(fetched.filter((job) => !hiddenIds.current.has(job.id)));
      setError(null);
    } catch (err) {
      setError('Failed to load discovered jobs: ' + err.message);
    } finally {
      if (!quiet) setLoading(false);
    }
  };

  // A posting seen on several boards is several rows; dismissing it hides them all.
  const handleDismiss = async (ids) => {
    // Drop it locally straight away; it is hidden server-side either way.
    ids.forEach((id) => hiddenIds.current.add(id));
    setJobs((prev) => prev.filter((job) => !hiddenIds.current.has(job.id)));

    const results = await Promise.allSettled(ids.map((id) => dismissDiscoveredJob(id)));
    const failures = results.filter((result) => result.status === 'rejected');
    if (!failures.length) return;

    // Only the rows whose request failed come back; the server says what else is left.
    results.forEach((result, i) => {
      if (result.status === 'rejected') hiddenIds.current.delete(ids[i]);
    });
    await fetchDiscovered({ quiet: true });
    setError('Failed to dismiss: ' + failures[0].reason.message);
  };

  return (
    <div style={styles.container}>
      <AppHeader />

      <main style={styles.main}>
        <div style={styles.intro}>
          <h2 style={styles.heading}>Discovered</h2>
          <p style={styles.subheading}>
            Roles found by the daily job board scrapes, newest first. Kept for 7 days.
          </p>
        </div>

        {error && <div style={styles.error}>{error}</div>}

        <DiscoveredList jobs={jobs} onDismiss={handleDismiss} loading={loading} />
      </main>
    </div>
  );
}

const styles = {
  container: {
    minHeight: '100vh',
    backgroundColor: '#f5f5f5',
  },
  main: {
    maxWidth: '1200px',
    margin: '0 auto',
    padding: '24px',
  },
  intro: {
    marginBottom: '20px',
  },
  heading: {
    margin: 0,
    fontSize: '20px',
    color: '#333',
  },
  subheading: {
    margin: '4px 0 0 0',
    color: '#6c757d',
    fontSize: '14px',
  },
  error: {
    padding: '12px 16px',
    borderRadius: '4px',
    marginBottom: '16px',
    backgroundColor: '#f8d7da',
    color: '#721c24',
  },
};
