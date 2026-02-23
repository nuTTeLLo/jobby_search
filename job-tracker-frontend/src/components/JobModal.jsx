import { useState, useEffect } from 'react';
import { getAttachments, uploadAttachment, downloadAttachment, deleteAttachment } from '../services/api';

const JOB_TYPES = [
  { value: '', label: 'Select type...' },
  { value: 'fulltime', label: 'Full-time' },
  { value: 'parttime', label: 'Part-time' },
  { value: 'contract', label: 'Contract' },
  { value: 'internship', label: 'Internship' },
];

const FILE_TYPES = [
  { value: 'resume', label: 'Resume' },
  { value: 'cover_letter', label: 'Cover Letter' },
];

export default function JobModal({ job, onSave, onClose, onRefresh }) {
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
  const [attachments, setAttachments] = useState([]);
  const [uploading, setUploading] = useState(false);
  const [selectedFile, setSelectedFile] = useState(null);
  const [selectedFileType, setSelectedFileType] = useState('resume');

  const isEditing = !!job;

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
      loadAttachments();
    }
  }, [job]);

  const loadAttachments = async () => {
    if (job?.id) {
      try {
        const data = await getAttachments(job.id);
        setAttachments(data);
      } catch (err) {
        console.error('Failed to load attachments:', err);
      }
    }
  };

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

  const handleFileChange = (e) => {
    setSelectedFile(e.target.files[0]);
  };

  const handleUpload = async () => {
    if (!selectedFile || !job?.id) return;
    
    setUploading(true);
    try {
      await uploadAttachment(job.id, selectedFile, selectedFileType);
      await loadAttachments();
      setSelectedFile(null);
      // Reset file input
      document.getElementById('file-input').value = '';
      if (onRefresh) onRefresh();
    } catch (err) {
      console.error('Failed to upload:', err);
      setError('Failed to upload file');
    } finally {
      setUploading(false);
    }
  };

  const handleDownload = async (attachment) => {
    try {
      await downloadAttachment(job.id, attachment.id);
    } catch (err) {
      console.error('Failed to download:', err);
    }
  };

  const handleDelete = async (attachmentId) => {
    if (!confirm('Are you sure you want to delete this attachment?')) return;
    try {
      await deleteAttachment(attachmentId);
      await loadAttachments();
      if (onRefresh) onRefresh();
    } catch (err) {
      console.error('Failed to delete:', err);
    }
  };

  const formatFileSize = (bytes) => {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
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
          <div style={styles.field}>
            <label style={styles.label}>Description</label>
            <textarea
              name="description"
              value={formData.description}
              onChange={handleChange}
              style={{ ...styles.input, minHeight: '100px', resize: 'vertical' }}
            />
          </div>

          {isEditing && (
            <div style={styles.attachmentsSection}>
              <label style={styles.label}>Attachments</label>
              
              {attachments.length > 0 && (
                <div style={styles.attachmentList}>
                  {attachments.map((attachment) => (
                    <div key={attachment.id} style={styles.attachmentItem}>
                      <span style={styles.attachmentIcon}>📄</span>
                      <div style={styles.attachmentInfo}>
                        <span style={styles.attachmentName}>{attachment.file_name}</span>
                        <span style={styles.attachmentMeta}>
                          {attachment.file_type === 'resume' ? 'Resume' : 'Cover Letter'} • {formatFileSize(attachment.file_size)}
                        </span>
                      </div>
                      <button
                        type="button"
                        onClick={() => handleDownload(attachment)}
                        style={styles.attachmentBtn}
                      >
                        Download
                      </button>
                      <button
                        type="button"
                        onClick={() => handleDelete(attachment.id)}
                        style={styles.deleteBtn}
                      >
                        ✕
                      </button>
                    </div>
                  ))}
                </div>
              )}

              <div style={styles.uploadSection}>
                <input
                  id="file-input"
                  type="file"
                  accept=".pdf,.doc,.docx"
                  onChange={handleFileChange}
                  style={styles.fileInput}
                />
                <select
                  value={selectedFileType}
                  onChange={(e) => setSelectedFileType(e.target.value)}
                  style={styles.fileTypeSelect}
                >
                  {FILE_TYPES.map((type) => (
                    <option key={type.value} value={type.value}>
                      {type.label}
                    </option>
                  ))}
                </select>
                <button
                  type="button"
                  onClick={handleUpload}
                  disabled={!selectedFile || uploading}
                  style={styles.uploadBtn}
                >
                  {uploading ? 'Uploading...' : 'Upload'}
                </button>
              </div>
              <div style={styles.uploadHint}>
                Accepted: PDF, DOC, DOCX (max 10MB)
              </div>
            </div>
          )}

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
    width: '550px',
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
  attachmentsSection: {
    marginBottom: '15px',
    padding: '15px',
    backgroundColor: '#f8f9fa',
    borderRadius: '4px',
    border: '1px solid #dee2e6',
  },
  attachmentList: {
    marginBottom: '12px',
  },
  attachmentItem: {
    display: 'flex',
    alignItems: 'center',
    gap: '10px',
    padding: '8px',
    backgroundColor: 'white',
    borderRadius: '4px',
    marginBottom: '8px',
    border: '1px solid #dee2e6',
  },
  attachmentIcon: {
    fontSize: '18px',
  },
  attachmentInfo: {
    flex: 1,
    minWidth: 0,
  },
  attachmentName: {
    display: 'block',
    fontSize: '14px',
    fontWeight: '500',
    color: '#212529',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  },
  attachmentMeta: {
    display: 'block',
    fontSize: '12px',
    color: '#6c757d',
  },
  attachmentBtn: {
    padding: '4px 10px',
    backgroundColor: '#0d6efd',
    color: 'white',
    border: 'none',
    borderRadius: '4px',
    fontSize: '12px',
    cursor: 'pointer',
  },
  deleteBtn: {
    padding: '4px 8px',
    backgroundColor: 'transparent',
    color: '#dc3545',
    border: '1px solid #dc3545',
    borderRadius: '4px',
    fontSize: '12px',
    cursor: 'pointer',
  },
  uploadSection: {
    display: 'flex',
    gap: '10px',
    alignItems: 'center',
  },
  fileInput: {
    flex: 1,
    fontSize: '14px',
  },
  fileTypeSelect: {
    width: '120px',
    padding: '8px',
    border: '1px solid #ced4da',
    borderRadius: '4px',
    fontSize: '14px',
    backgroundColor: 'white',
  },
  uploadBtn: {
    padding: '8px 16px',
    backgroundColor: '#28a745',
    color: 'white',
    border: 'none',
    borderRadius: '4px',
    fontSize: '14px',
    cursor: 'pointer',
  },
  uploadHint: {
    marginTop: '8px',
    fontSize: '12px',
    color: '#6c757d',
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
