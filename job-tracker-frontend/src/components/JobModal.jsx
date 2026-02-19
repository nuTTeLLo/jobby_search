import { useState, useEffect } from 'react';

const JOB_TYPES = [
  { value: '', label: 'Select type...' },
  { value: 'fulltime', label: 'Full-time' },
  { value: 'parttime', label: 'Part-time' },
  { value: 'contract', label: 'Contract' },
  { value: 'internship', label: 'Internship' },
];

export default function JobModal({ job, onSave, onClose }) {
  const [formData, setFormData] = useState({
    job_title: '',
    company_name: '',
    location: '',
    job_url: '',
    description: '',
    salary: '',
    job_type: '',
    is_remote: false,
    notes: '',
  });
  const [error, setError] = useState('');

  useEffect(() => {
    if (job) {
      setFormData({
        job_title: job.job_title || '',
        company_name: job.company_name || '',
        location: job.location || '',
        job_url: job.job_url || '',
        description: job.description || '',
        salary: job.salary || '',
        job_type: job.job_type || '',
        is_remote: job.is_remote || false,
        notes: job.notes || '',
      });
    }
  }, [job]);

  const handleChange = (e) => {
    const { name, value, type, checked } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: type === 'checkbox' ? checked : value,
    }));
  };

  const handleSubmit = (e) => {
    e.preventDefault();
    if (!formData.job_title.trim()) {
      setError('Job title is required');
      return;
    }
    if (!formData.job_url.trim()) {
      setError('Job URL is required');
      return;
    }
    onSave(formData);
  };

  return (
    <div style={styles.overlay}>
      <div style={styles.modal}>
        <div style={styles.header}>
          <h2 style={styles.title}>{job ? 'Edit Job' : 'Add Job'}</h2>
          <button onClick={onClose} style={styles.closeBtn}>&times;</button>
        </div>
        <form onSubmit={handleSubmit}>
          {error && <div style={styles.error}>{error}</div>}
          <div style={styles.field}>
            <label style={styles.label}>Job Title *</label>
            <input
              type="text"
              name="job_title"
              value={formData.job_title}
              onChange={handleChange}
              style={styles.input}
              required
            />
          </div>
          <div style={styles.field}>
            <label style={styles.label}>Company Name</label>
            <input
              type="text"
              name="company_name"
              value={formData.company_name}
              onChange={handleChange}
              style={styles.input}
            />
          </div>
          <div style={styles.field}>
            <label style={styles.label}>Job URL *</label>
            <input
              type="url"
              name="job_url"
              value={formData.job_url}
              onChange={handleChange}
              style={styles.input}
              placeholder="https://..."
              required
            />
          </div>
          <div style={styles.row}>
            <div style={styles.field}>
              <label style={styles.label}>Location</label>
              <input
                type="text"
                name="location"
                value={formData.location}
                onChange={handleChange}
                style={styles.input}
              />
            </div>
            <div style={styles.field}>
              <label style={styles.label}>Job Type</label>
              <select
                name="job_type"
                value={formData.job_type}
                onChange={handleChange}
                style={styles.select}
              >
                {JOB_TYPES.map((type) => (
                  <option key={type.value} value={type.value}>
                    {type.label}
                  </option>
                ))}
              </select>
            </div>
          </div>
          <div style={styles.field}>
            <label style={styles.label}>Salary</label>
            <input
              type="text"
              name="salary"
              value={formData.salary}
              onChange={handleChange}
              style={styles.input}
              placeholder="e.g., $80,000 - $120,000"
            />
          </div>
          <div style={styles.field}>
            <label style={styles.label}>Notes</label>
            <textarea
              name="notes"
              value={formData.notes}
              onChange={handleChange}
              style={{ ...styles.input, minHeight: '80px', resize: 'vertical' }}
            />
          </div>
          <div style={styles.actions}>
            <button type="button" onClick={onClose} style={styles.cancelBtn}>
              Cancel
            </button>
            <button type="submit" style={styles.saveBtn}>
              {job ? 'Update' : 'Add'} Job
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

const styles = {
  overlay: {
    position: 'fixed',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    backgroundColor: 'rgba(0,0,0,0.5)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    zIndex: 1000,
  },
  modal: {
    backgroundColor: 'white',
    borderRadius: '8px',
    padding: '24px',
    width: '500px',
    maxWidth: '90%',
    maxHeight: '90vh',
    overflowY: 'auto',
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '20px',
  },
  title: {
    margin: 0,
    fontSize: '20px',
    fontWeight: '600',
  },
  closeBtn: {
    background: 'none',
    border: 'none',
    fontSize: '24px',
    cursor: 'pointer',
    padding: 0,
    lineHeight: 1,
  },
  error: {
    backgroundColor: '#f8d7da',
    color: '#721c24',
    padding: '10px',
    borderRadius: '4px',
    marginBottom: '15px',
    fontSize: '14px',
  },
  field: {
    marginBottom: '15px',
  },
  row: {
    display: 'flex',
    gap: '15px',
  },
  label: {
    display: 'block',
    fontSize: '14px',
    fontWeight: '500',
    marginBottom: '5px',
    color: '#495057',
  },
  input: {
    width: '100%',
    padding: '8px 12px',
    backgroundColor: 'white',
    border: '1px solid #ced4da',
    borderRadius: '4px',
    fontSize: '14px',
    boxSizing: 'border-box',
    color: '#212529',
  },
  select: {
    width: '100%',
    padding: '8px 12px',
    border: '1px solid #ced4da',
    borderRadius: '4px',
    fontSize: '14px',
    backgroundColor: 'white',
    boxSizing: 'border-box',
    color: '#212529',
  },
  actions: {
    display: 'flex',
    gap: '10px',
    justifyContent: 'flex-end',
    marginTop: '20px',
  },
  cancelBtn: {
    padding: '10px 20px',
    backgroundColor: '#6c757d',
    color: 'white',
    border: 'none',
    borderRadius: '4px',
    fontSize: '14px',
    cursor: 'pointer',
  },
  saveBtn: {
    padding: '10px 20px',
    backgroundColor: '#0d6efd',
    color: 'white',
    border: 'none',
    borderRadius: '4px',
    fontSize: '14px',
    cursor: 'pointer',
  },
};
