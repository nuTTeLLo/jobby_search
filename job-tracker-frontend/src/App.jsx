import { useState, useEffect } from 'react';
import JobSearch from './components/JobSearch';
import JobList from './components/JobList';
import JobModal from './components/JobModal';
import { getJobs, createJob, updateJob, deleteJob, updateJobStatus, searchJobs } from './services/api';
import './App.css';

const STATUS_TABS = [
  { value: '', label: 'All' },
  { value: 'new', label: 'New' },
  { value: 'viewed', label: 'Viewed' },
  { value: 'applied', label: 'Applied' },
  { value: 'rejected', label: 'Rejected' },
  { value: 'shortlisted', label: 'Shortlisted' },
];

function App() {
  const [jobs, setJobs] = useState([]);
  const [searchResults, setSearchResults] = useState([]);
  const [loading, setLoading] = useState(false);
  const [searching, setSearching] = useState(false);
  const [statusFilter, setStatusFilter] = useState('');
  const [modalOpen, setModalOpen] = useState(false);
  const [editingJob, setEditingJob] = useState(null);
  const [message, setMessage] = useState(null);

  useEffect(() => {
    fetchJobs();
  }, [statusFilter]);

  const fetchJobs = async () => {
    setLoading(true);
    try {
      const data = await getJobs(statusFilter);
      setJobs(data);
    } catch (error) {
      showMessage('Failed to fetch jobs: ' + error.message, 'error');
    } finally {
      setLoading(false);
    }
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
        source: 'mcp',
      };
      await createJob(jobData);
      showMessage('Job added to tracker', 'success');
      // Mark as saved in search results
      setSearchResults(prev => prev.map(j => 
        j.job_url === job.job_url ? { ...j, is_saved: true } : j
      ));
      fetchJobs();
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
      if (editingJob) {
        await updateJob(editingJob.id, jobData);
        showMessage('Job updated successfully', 'success');
      } else {
        await createJob(jobData);
        showMessage('Job added successfully', 'success');
      }
      setModalOpen(false);
      setEditingJob(null);
      fetchJobs();
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
      await updateJobStatus(id, newStatus);
      fetchJobs();
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
      <header style={styles.header}>
        <h1 style={styles.title}>Job Tracker</h1>
      </header>

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
              Search Results 
              <button 
                onClick={() => setSearchResults([])}
                style={styles.clearBtn}
              >
                Clear
              </button>
            </h2>
            <div style={styles.resultsGrid}>
              {searchResults.map((job, index) => (
                <div key={index} style={styles.resultCard}>
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
                  </div>
                  {job.salary && <div style={styles.resultSalary}>{job.salary}</div>}
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

        {loading ? (
          <div style={styles.loading}>Loading...</div>
        ) : (
          <JobList
            jobs={jobs}
            onStatusChange={handleStatusChange}
            onEdit={handleEdit}
            onDelete={handleDelete}
          />
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
  header: {
    backgroundColor: '#0d6efd',
    color: 'white',
    padding: '20px',
    boxShadow: '0 2px 4px rgba(0,0,0,0.1)',
  },
  title: {
    margin: 0,
    fontSize: '24px',
    fontWeight: '600',
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
    gap: '15px',
  },
  resultCard: {
    backgroundColor: 'white',
    borderRadius: '8px',
    padding: '15px',
    boxShadow: '0 1px 3px rgba(0,0,0,0.1)',
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
  resultSalary: {
    fontSize: '13px',
    color: '#198754',
    fontWeight: '500',
    marginBottom: '12px',
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
  loading: {
    textAlign: 'center',
    padding: '40px',
    color: '#6c757d',
    backgroundColor: 'white',
    borderRadius: '8px',
    boxShadow: '0 1px 3px rgba(0,0,0,0.1)',
  },
};

export default App;
