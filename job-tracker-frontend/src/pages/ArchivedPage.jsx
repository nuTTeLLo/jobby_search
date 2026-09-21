import { useState, useEffect, useCallback } from 'react';
import AppHeader from '../components/AppHeader';
import JobList from '../components/JobList';
import JobModal from '../components/JobModal';
import Pagination from '../components/Pagination';
import { getJobs, updateJob, updateJobStatus, deleteJob } from '../services/api';

const PAGE_SIZE = 25;
const FILTER_DEBOUNCE_MS = 300;

// Applications that went unanswered for six months, retired by the weekly sweep.
// Their own page rather than a tab on the tracker: the tracker is the list of
// things still in play, and the API leaves archived jobs out of it entirely.
export default function ArchivedPage() {
  const [jobs, setJobs] = useState([]);
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState(null);
  const [filterText, setFilterText] = useState('');
  const [appliedFilter, setAppliedFilter] = useState('');
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [editingJob, setEditingJob] = useState(null);
  const [sort, setSort] = useState('');
  const [order, setOrder] = useState('asc');

  useEffect(() => {
    const timer = setTimeout(() => setAppliedFilter(filterText.trim()), FILTER_DEBOUNCE_MS);
    return () => clearTimeout(timer);
  }, [filterText]);

  useEffect(() => {
    setPage(1);
  }, [appliedFilter, sort, order]);

  const fetchJobs = useCallback(async () => {
    setLoading(true);
    try {
      const data = await getJobs({
        status: 'archived',
        q: appliedFilter,
        page,
        pageSize: PAGE_SIZE,
        sort,
        order,
      });
      setJobs(data.jobs);
      setTotal(data.total);
      // Restoring the last row of the last page can leave us past the end.
      if (data.jobs.length === 0 && page > 1) {
        setPage(page - 1);
      }
    } catch (error) {
      showMessage('Failed to fetch archived jobs: ' + error.message, 'error');
    } finally {
      setLoading(false);
    }
  }, [appliedFilter, page, sort, order]);

  useEffect(() => {
    fetchJobs();
  }, [fetchJobs]);

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  const showMessage = (text, type) => {
    setMessage({ text, type });
    setTimeout(() => setMessage(null), 3000);
  };

  // Any status change takes the job off this page, so always refetch.
  const handleStatusChange = async (id, newStatus) => {
    try {
      await updateJobStatus(id, newStatus);
      if (newStatus !== 'archived') {
        showMessage(`Moved back to ${newStatus}`, 'success');
      }
      fetchJobs();
    } catch (error) {
      showMessage('Failed to update status: ' + error.message, 'error');
    }
  };

  const handleDelete = async (id) => {
    if (!window.confirm('Delete this archived job for good?')) return;
    try {
      await deleteJob(id);
      showMessage('Job deleted', 'success');
      fetchJobs();
    } catch (error) {
      showMessage('Failed to delete job: ' + error.message, 'error');
    }
  };

  const handleSaveJob = async (jobData) => {
    try {
      await updateJob(editingJob.id, jobData);
      showMessage('Job updated', 'success');
      setEditingJob(null);
      fetchJobs();
    } catch (error) {
      showMessage('Failed to save job: ' + error.message, 'error');
    }
  };

  return (
    <div style={styles.container}>
      <AppHeader />

      <main style={styles.main}>
        <div style={styles.intro}>
          <h2 style={styles.heading}>Archived</h2>
          <p style={styles.subheading}>
            Applications still unanswered six months on, retired by the weekly sweep.
            Their documents live in the <code>Archived/</code> folder. Change the status
            of one to put it back in the tracker.
          </p>
        </div>

        {message && (
          <div
            style={{
              ...styles.message,
              backgroundColor: message.type === 'error' ? '#f8d7da' : '#d4edda',
              color: message.type === 'error' ? '#721c24' : '#155724',
            }}
          >
            {message.text}
          </div>
        )}

        <div style={styles.filterBar}>
          <input
            type="text"
            value={filterText}
            onChange={(e) => setFilterText(e.target.value)}
            placeholder="Filter by title, company, location, type or source..."
            style={styles.filterInput}
          />
          {filterText && (
            <button onClick={() => setFilterText('')} style={styles.filterClearBtn}>
              Clear
            </button>
          )}
          <span style={styles.filterCount}>
            {total === 0
              ? 'Nothing archived'
              : `${total} job${total === 1 ? '' : 's'}${appliedFilter ? ' matched' : ''}`}
          </span>
        </div>

        {loading ? (
          <div style={styles.loading}>Loading...</div>
        ) : (
          <>
            <JobList
              jobs={jobs}
              sort={sort}
              order={order}
              onSort={(key, direction) => {
                setSort(key);
                setOrder(direction);
              }}
              onStatusChange={handleStatusChange}
              onEdit={setEditingJob}
              onDelete={handleDelete}
            />
            <Pagination page={page} totalPages={totalPages} onChange={setPage} />
          </>
        )}
      </main>

      {editingJob && (
        <JobModal
          job={editingJob}
          onSave={handleSaveJob}
          onClose={() => setEditingJob(null)}
          onRefresh={fetchJobs}
        />
      )}
    </div>
  );
}

// Matches the Discovered page, which this is a sibling of.
const styles = {
  container: {
    minHeight: '100vh',
    backgroundColor: '#f5f5f5',
  },
  main: {
    maxWidth: '1200px',
    margin: '0 auto',
    padding: '20px',
  },
  intro: {
    marginBottom: '16px',
  },
  heading: {
    margin: '0 0 4px',
    fontSize: '20px',
    fontWeight: 600,
  },
  subheading: {
    margin: 0,
    fontSize: '14px',
    color: '#6c757d',
  },
  message: {
    padding: '12px',
    borderRadius: '4px',
    marginBottom: '20px',
    fontSize: '14px',
  },
  filterBar: {
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    marginBottom: '12px',
  },
  filterInput: {
    flex: 1,
    padding: '8px 12px',
    fontSize: '14px',
    border: '1px solid #ced4da',
    borderRadius: '4px',
  },
  filterClearBtn: {
    padding: '8px 12px',
    fontSize: '13px',
    border: '1px solid #ced4da',
    borderRadius: '4px',
    backgroundColor: 'white',
    cursor: 'pointer',
  },
  filterCount: {
    fontSize: '13px',
    color: '#6c757d',
    whiteSpace: 'nowrap',
  },
  loading: {
    padding: '40px',
    textAlign: 'center',
    color: '#6c757d',
  },
};
