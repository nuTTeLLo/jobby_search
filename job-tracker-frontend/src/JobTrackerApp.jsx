import { useState, useEffect, useCallback } from 'react';
import JobSearch from './components/JobSearch';
import JobList from './components/JobList';
import JobModal from './components/JobModal';
import AppHeader from './components/AppHeader';
import { getJobs, createJob, updateJob, deleteJob, updateJobStatus, searchJobs } from './services/api';
import './App.css';

const PAGE_SIZE = 25;
const FILTER_DEBOUNCE_MS = 300;

const STATUS_TABS = [
  { value: '', label: 'All' },
  { value: 'new', label: 'New' },
  { value: 'viewed', label: 'Viewed' },
  { value: 'applied', label: 'Applied' },
  { value: 'rejected', label: 'Rejected' },
  { value: 'shortlisted', label: 'Shortlisted' },
];

function JobTrackerApp() {
  const [jobs, setJobs] = useState([]);
  const [searchResults, setSearchResults] = useState([]);
  const [loading, setLoading] = useState(false);
  const [searching, setSearching] = useState(false);
  const [statusFilter, setStatusFilter] = useState('');
  const [modalOpen, setModalOpen] = useState(false);
  const [editingJob, setEditingJob] = useState(null);
  const [message, setMessage] = useState(null);
  const [hoveredJob, setHoveredJob] = useState(null);
  const [filterText, setFilterText] = useState('');
  // The filter runs server-side, so debounce it rather than firing per keystroke.
  const [appliedFilter, setAppliedFilter] = useState('');
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);

  useEffect(() => {
    const timer = setTimeout(() => setAppliedFilter(filterText.trim()), FILTER_DEBOUNCE_MS);
    return () => clearTimeout(timer);
  }, [filterText]);

  // Any change to what's being listed starts again from the first page.
  useEffect(() => {
    setPage(1);
  }, [statusFilter, appliedFilter]);

  const fetchJobs = useCallback(async () => {
    setLoading(true);
    try {
      const data = await getJobs({
        status: statusFilter,
        q: appliedFilter,
        page,
        pageSize: PAGE_SIZE,
      });
      setJobs(data.jobs);
      setTotal(data.total);
      // Deleting the last row of the last page can leave us past the end.
      if (data.jobs.length === 0 && page > 1) {
        setPage(page - 1);
      }
    } catch (error) {
      showMessage('Failed to fetch jobs: ' + error.message, 'error');
    } finally {
      setLoading(false);
    }
  }, [statusFilter, appliedFilter, page]);

  useEffect(() => {
    fetchJobs();
  }, [fetchJobs]);

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  // Whether a job belongs in the currently visible list given the active tab.
  const belongsToFilter = (job) => statusFilter === '' || job.status === statusFilter;

  // Mirrors the backend's free-text match, so an edited job that no longer
  // matches the filter box triggers a refetch instead of lingering on the page.
  const matchesFilter = (job) => {
    const terms = appliedFilter.toLowerCase().split(/\s+/).filter(Boolean);
    if (terms.length === 0) return true;
    const haystack = [
      job.job_title,
      job.company_name,
      job.location,
      job.job_type,
      job.source,
      job.status,
    ]
      .filter(Boolean)
      .join(' ')
      .toLowerCase();
    return terms.every((term) => haystack.includes(term));
  };

  // Merge an updated job into the current page without a reload. Anything that
  // changes which rows the page holds (a new job, or one that no longer matches
  // the active tab or filter) needs a refetch so the page stays full and the
  // total stays accurate.
  const upsertJob = (job) => {
    const onPage = jobs.some((j) => j.id === job.id);
    if (!onPage || !belongsToFilter(job) || !matchesFilter(job)) {
      fetchJobs();
      return;
    }
    // Status/update responses don't preload attachments (they come back null),
    // so keep the ones already in state rather than clobbering them.
    setJobs((prev) =>
      prev.map((j) =>
        j.id === job.id ? { ...j, ...job, attachments: job.attachments ?? j.attachments } : j
      )
    );
  };

  const handleSearch = async (params) => {
    setSearching(true);
    try {
      const result = await searchJobs(params);
      setSearchResults(result.jobs || []);
      showMessage(`Found ${result.count} jobs`, 'success');
    } catch (error) {
      showMessage('Search failed: ' + error.message, 'error');
    } finally {
      setSearching(false);
    }
  };

  const handleAddFromSearch = async (job) => {
    try {
      const jobData = {
        job_title: job.job_title,
        company_name: job.company_name,
        location: job.location,
        job_url: job.job_url,
        description: job.description,
        salary: job.salary,
        job_type: job.job_type,
        is_remote: job.is_remote,
        source: job.source || 'mcp',
      };
      const created = await createJob(jobData);
      showMessage('Job added to tracker', 'success');
      setSearchResults(prev => prev.map(j =>
        j.job_url === job.job_url ? { ...j, is_saved: true } : j
      ));
      upsertJob(created);
    } catch (error) {
      if (error.response?.data?.error) {
        showMessage(error.response.data.error, 'error');
      } else {
        showMessage('Failed to add job: ' + error.message, 'error');
      }
    }
  };

  const handleSaveJob = async (jobData) => {
    try {
      let saved;
      if (editingJob) {
        saved = await updateJob(editingJob.id, jobData);
        showMessage('Job updated successfully', 'success');
      } else {
        saved = await createJob(jobData);
        showMessage('Job added successfully', 'success');
      }
      setModalOpen(false);
      setEditingJob(null);
      upsertJob(saved);
    } catch (error) {
      if (error.response?.data?.error) {
        showMessage(error.response.data.error, 'error');
      } else {
        showMessage('Failed to save job: ' + error.message, 'error');
      }
    }
  };

  const handleEdit = (job) => {
    setEditingJob(job);
    setModalOpen(true);
  };

  const handleDelete = async (id) => {
    if (!window.confirm('Are you sure you want to delete this job?')) {
      return;
    }
    try {
      await deleteJob(id);
      showMessage('Job deleted successfully', 'success');
      fetchJobs();
    } catch (error) {
      showMessage('Failed to delete job: ' + error.message, 'error');
    }
  };

  const handleStatusChange = async (id, newStatus) => {
    try {
      const updated = await updateJobStatus(id, newStatus);
      upsertJob(updated);
    } catch (error) {
      showMessage('Failed to update status: ' + error.message, 'error');
    }
  };

  const showMessage = (text, type) => {
    setMessage({ text, type });
    setTimeout(() => setMessage(null), 3000);
  };

  return (
    <div style={styles.container}>
      <AppHeader />

      <main style={styles.main}>
        <JobSearch onSearch={handleSearch} loading={searching} />

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

        {/* Search Results Section */}
        {searchResults.length > 0 && (
          <div style={styles.searchResults}>
            <h2 style={styles.sectionTitle}>
              Search Results ({searchResults.length})
              <button
                onClick={() => setSearchResults([])}
                style={styles.clearBtn}
              >
                Clear
              </button>
            </h2>
            <div style={styles.resultsGrid}>
              {searchResults.map((job, index) => (
                <div
                  key={index}
                  style={{
                    ...styles.resultCard,
                    ...(hoveredJob === job.job_url ? styles.resultCardHover : {})
                  }}
                  onMouseEnter={() => setHoveredJob(job.job_url)}
                  onMouseLeave={() => setHoveredJob(null)}
                >
                  <div style={styles.resultHeader}>
                    <a
                      href={job.job_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      style={styles.resultTitle}
                    >
                      {job.job_title}
                    </a>
                    {job.is_saved && (
                      <span style={styles.savedBadge}>Saved</span>
                    )}
                  </div>
                  <div style={styles.resultCompany}>{job.company_name}</div>
                  <div style={styles.resultLocation}>{job.location}</div>
                  <div style={styles.resultMeta}>
                    {job.job_type && <span style={styles.resultTag}>{job.job_type}</span>}
                    {job.is_remote && <span style={styles.resultTag}>Remote</span>}
                    {job.easy_apply && (
                      <span style={{...styles.resultTag, backgroundColor: '#28a745', color: 'white'}}>Easy Apply</span>
                    )}
                    {job.source && (
                      <span style={styles.sourceBadge}>{job.source}</span>
                    )}
                  </div>
                  {job.salary && <div style={styles.resultSalary}>{job.salary}</div>}
                  {job.description && (
                    <div style={{
                      ...styles.resultDescription,
                      ...(hoveredJob === job.job_url ? styles.resultDescriptionExpanded : {})
                    }}>
                      {job.description}
                    </div>
                  )}
                  <button
                    onClick={() => handleAddFromSearch(job)}
                    disabled={job.is_saved}
                    style={job.is_saved ? styles.addedBtn : styles.addBtn}
                  >
                    {job.is_saved ? 'Added' : 'Add to Tracker'}
                  </button>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Tracked Jobs Section */}
        <div style={styles.tabs}>
          {STATUS_TABS.map((tab) => (
            <button
              key={tab.value}
              onClick={() => setStatusFilter(tab.value)}
              style={{
                ...styles.tab,
                ...(statusFilter === tab.value ? styles.activeTab : {}),
              }}
            >
              {tab.label}
            </button>
          ))}
          <button
            onClick={() => {
              setEditingJob(null);
              setModalOpen(true);
            }}
            style={styles.addJobBtn}
          >
            + Add Job
          </button>
        </div>

        <div style={styles.filterBar}>
          <input
            type="text"
            value={filterText}
            onChange={(e) => setFilterText(e.target.value)}
            placeholder="Filter jobs by title, company, location, type or source..."
            style={styles.filterInput}
          />
          {filterText && (
            <button onClick={() => setFilterText('')} style={styles.filterClearBtn}>
              Clear
            </button>
          )}
          <span style={styles.filterCount}>
            {total === 0
              ? 'No jobs'
              : `${total} job${total === 1 ? '' : 's'}${appliedFilter ? ' matched' : ''}`}
          </span>
        </div>

        {loading ? (
          <div style={styles.loading}>Loading...</div>
        ) : (
          <>
            <JobList
              jobs={jobs}
              onStatusChange={handleStatusChange}
              onEdit={handleEdit}
              onDelete={handleDelete}
            />
            {totalPages > 1 && (
              <div style={styles.pagination}>
                <button
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  disabled={page <= 1}
                  style={page <= 1 ? styles.pageBtnDisabled : styles.pageBtn}
                >
                  Previous
                </button>
                <span style={styles.pageStatus}>
                  Page {page} of {totalPages}
                </span>
                <button
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                  disabled={page >= totalPages}
                  style={page >= totalPages ? styles.pageBtnDisabled : styles.pageBtn}
                >
                  Next
                </button>
              </div>
            )}
          </>
        )}
      </main>

      {modalOpen && (
        <JobModal
          job={editingJob}
          onSave={handleSaveJob}
          onClose={() => {
            setModalOpen(false);
            setEditingJob(null);
          }}
          onRefresh={fetchJobs}
        />
      )}
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
    padding: '20px',
  },
  message: {
    padding: '12px',
    borderRadius: '4px',
    marginBottom: '20px',
    fontSize: '14px',
  },
  searchResults: {
    marginBottom: '30px',
  },
  sectionTitle: {
    fontSize: '18px',
    fontWeight: '600',
    marginBottom: '15px',
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  clearBtn: {
    padding: '4px 12px',
    backgroundColor: '#6c757d',
    color: 'white',
    border: 'none',
    borderRadius: '4px',
    fontSize: '12px',
    cursor: 'pointer',
  },
  resultsGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))',
    alignItems: 'start',
    gap: '15px',
  },
  resultCard: {
    backgroundColor: 'white',
    borderRadius: '8px',
    padding: '15px',
    boxShadow: '0 1px 3px rgba(0,0,0,0.1)',
    position: 'relative',
  },
  resultHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
    marginBottom: '8px',
  },
  resultTitle: {
    fontSize: '16px',
    fontWeight: '600',
    color: '#0d6efd',
    textDecoration: 'none',
    flex: 1,
  },
  savedBadge: {
    fontSize: '11px',
    backgroundColor: '#198754',
    color: 'white',
    padding: '2px 8px',
    borderRadius: '10px',
    marginLeft: '8px',
  },
  resultCompany: {
    fontSize: '14px',
    color: '#495057',
    marginBottom: '4px',
  },
  resultLocation: {
    fontSize: '13px',
    color: '#6c757d',
    marginBottom: '8px',
  },
  resultMeta: {
    display: 'flex',
    gap: '8px',
    marginBottom: '8px',
  },
  resultTag: {
    fontSize: '11px',
    backgroundColor: '#e9ecef',
    color: '#495057',
    padding: '2px 8px',
    borderRadius: '4px',
  },
  sourceBadge: {
    display: 'inline-block',
    padding: '2px 8px',
    backgroundColor: '#e7f1ff',
    color: '#0d6efd',
    borderRadius: '4px',
    fontSize: '10px',
    fontWeight: '500',
  },
  resultSalary: {
    fontSize: '13px',
    color: '#198754',
    fontWeight: '500',
    marginBottom: '12px',
  },
  resultDescription: {
    fontSize: '12px',
    color: '#666',
    marginTop: '6px',
    lineHeight: '1.4',
    whiteSpace: 'nowrap',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    maxHeight: '30px',
    transition: 'all 0.2s ease',
  },
  resultDescriptionExpanded: {
    whiteSpace: 'normal',
    maxHeight: '200px',
    overflowY: 'auto',
  },
  resultCardHover: {
    boxShadow: '0 4px 12px rgba(0,0,0,0.15)',
  },
  addBtn: {
    width: '100%',
    padding: '8px',
    backgroundColor: '#0d6efd',
    color: 'white',
    border: 'none',
    borderRadius: '4px',
    fontSize: '13px',
    cursor: 'pointer',
  },
  addedBtn: {
    width: '100%',
    padding: '8px',
    backgroundColor: '#198754',
    color: 'white',
    border: 'none',
    borderRadius: '4px',
    fontSize: '13px',
    cursor: 'default',
  },
  tabs: {
    display: 'flex',
    gap: '10px',
    marginBottom: '20px',
    flexWrap: 'wrap',
  },
  tab: {
    padding: '8px 16px',
    backgroundColor: 'white',
    border: '1px solid #dee2e6',
    borderRadius: '4px',
    fontSize: '14px',
    cursor: 'pointer',
    transition: 'all 0.2s',
    color: '#333',
  },
  activeTab: {
    backgroundColor: '#0d6efd',
    color: 'white',
    borderColor: '#0d6efd',
  },
  addJobBtn: {
    marginLeft: 'auto',
    padding: '8px 16px',
    backgroundColor: '#198754',
    color: 'white',
    border: 'none',
    borderRadius: '4px',
    fontSize: '14px',
    cursor: 'pointer',
  },
  filterBar: {
    display: 'flex',
    alignItems: 'center',
    gap: '10px',
    marginBottom: '15px',
  },
  filterInput: {
    flex: 1,
    padding: '8px 12px',
    border: '1px solid #dee2e6',
    borderRadius: '4px',
    fontSize: '14px',
    color: '#333',
    backgroundColor: 'white',
  },
  filterClearBtn: {
    padding: '8px 14px',
    backgroundColor: '#6c757d',
    color: 'white',
    border: 'none',
    borderRadius: '4px',
    fontSize: '13px',
    cursor: 'pointer',
  },
  filterCount: {
    fontSize: '13px',
    color: '#6c757d',
    whiteSpace: 'nowrap',
  },
  pagination: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    gap: '12px',
    marginTop: '20px',
  },
  pageBtn: {
    padding: '8px 16px',
    backgroundColor: 'white',
    border: '1px solid #dee2e6',
    borderRadius: '4px',
    fontSize: '14px',
    color: '#333',
    cursor: 'pointer',
  },
  pageBtnDisabled: {
    padding: '8px 16px',
    backgroundColor: '#f1f3f5',
    border: '1px solid #dee2e6',
    borderRadius: '4px',
    fontSize: '14px',
    color: '#adb5bd',
    cursor: 'not-allowed',
  },
  pageStatus: {
    fontSize: '13px',
    color: '#6c757d',
  },
  loading: {
    textAlign: 'center',
    padding: '40px',
    color: '#6c757d',
    backgroundColor: 'white',
    borderRadius: '8px',
    boxShadow: '0 1px 3px rgba(0,0,0,0.1)',
  },
};

export default JobTrackerApp;
