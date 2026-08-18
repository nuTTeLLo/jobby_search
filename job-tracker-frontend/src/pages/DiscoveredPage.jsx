import { useEffect, useState } from 'react';
import AppHeader from '../components/AppHeader';
import DiscoveredList from '../components/DiscoveredList';
import { getDiscoveredJobs, dismissDiscoveredJob } from '../services/api';

// Read-only feed of postings found by the daily LinkedIn scrape. Deliberately has
// no "add to tracker" action: application status is tracked on LinkedIn itself,
// and scraped rows never enter the jobs table.
export default function DiscoveredPage() {
  const [jobs, setJobs] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  useEffect(() => {
    fetchDiscovered();
  }, []);

  const fetchDiscovered = async () => {
    setLoading(true);
    try {
      setJobs(await getDiscoveredJobs());
      setError(null);
    } catch (err) {
      setError('Failed to load discovered jobs: ' + err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleDismiss = async (id) => {
    // Drop it locally straight away; it is hidden server-side either way.
    const previous = jobs;
    setJobs((prev) => prev.filter((job) => job.id !== id));
    try {
      await dismissDiscoveredJob(id);
    } catch (err) {
      setJobs(previous);
      setError('Failed to dismiss: ' + err.message);
    }
  };

  return (
    <div style={styles.container}>
      <AppHeader />

      <main style={styles.main}>
        <div style={styles.intro}>
          <h2 style={styles.heading}>Discovered</h2>
          <p style={styles.subheading}>
            Roles found by the daily LinkedIn scrape, newest first. Kept for 7 days.
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
