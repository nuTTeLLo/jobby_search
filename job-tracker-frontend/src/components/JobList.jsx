import { useState, useEffect } from 'react';
import StatusBadge from './StatusBadge';
import { API_BASE, downloadAttachment } from '../services/api';

const STATUSES = ['new', 'viewed', 'applied', 'rejected', 'shortlisted'];

const SOURCE_BADGES = {
  linkedin: { backgroundColor: '#0077b5', color: 'white', label: 'LinkedIn' },
  seek: { backgroundColor: '#ff4e00', color: 'white', label: 'Seek' },
  indeed: { backgroundColor: '#2164f3', color: 'white', label: 'Indeed' },
  glassdoor: { backgroundColor: '#0caa41', color: 'white', label: 'Glassdoor' },
  other: { backgroundColor: '#6c757d', color: 'white', label: 'Other' },
};

export default function JobList({ jobs, onStatusChange, onEdit, onDelete }) {
  const [statusMenu, setStatusMenu] = useState(null); // { jobId, position: { top, left } }
  const [sortColumn, setSortColumn] = useState(null);
  const [sortDirection, setSortDirection] = useState('asc');

  const handleSort = (column) => {
    if (sortColumn === column) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
    } else {
      setSortColumn(column);
      setSortDirection('asc');
    }
  };

  const getSortIndicator = (column) => {
    if (sortColumn !== column) return null;
    return sortDirection === 'asc' ? '▲' : '▼';
  };

  const sortedJobs = [...jobs].sort((a, b) => {
    if (!sortColumn) return 0;

    let aVal, bVal;

    switch (sortColumn) {
      case 'Job Title':
        aVal = a.job_title || '';
        bVal = b.job_title || '';
        break;
      case 'Company':
        aVal = a.company_name || '';
        bVal = b.company_name || '';
        break;
      case 'Location':
        aVal = a.location || '';
        bVal = b.location || '';
        break;
      case 'Status':
        aVal = a.status || '';
        bVal = b.status || '';
        break;
      case 'Updated':
        aVal = a.updated ? new Date(a.updated).getTime() : 0;
        bVal = b.updated ? new Date(b.updated).getTime() : 0;
        break;
      case 'Source':
        aVal = a.source || '';
        bVal = b.source || '';
        break;
      default:
        return 0;
    }

    if (aVal < bVal) return sortDirection === 'asc' ? -1 : 1;
    if (aVal > bVal) return sortDirection === 'asc' ? 1 : -1;
    return 0;
  });

  const handleStatusClick = (job, e) => {
    const rect = e.currentTarget.getBoundingClientRect();
    setStatusMenu({
      jobId: job.id,
      position: {
        top: rect.bottom + 5,
        left: rect.left,
      },
    });
  };

  const handleStatusChange = (jobId, status) => {
    onStatusChange(jobId, status);
    setStatusMenu(null);
  };

  const closeStatusMenu = () => setStatusMenu(null);

  useEffect(() => {
    if (!statusMenu) return;

    const handleKeyDown = (e) => {
      if (e.key === 'Escape') {
        setStatusMenu(null);
      }
    };

    const handleClickOutside = (e) => {
      // Don't close if clicking on the status badge that opened the menu
      if (e.target.closest('.status-badge')) {
        return;
      }
      // Don't close if clicking inside the menu
      if (e.target.closest('#status-menu')) {
        return;
      }
      setStatusMenu(null);
    };

    document.addEventListener('keydown', handleKeyDown);
    // Use setTimeout to delay adding the click listener so the current click doesn't immediately close it
    setTimeout(() => {
      document.addEventListener('click', handleClickOutside);
    }, 0);

    return () => {
      document.removeEventListener('keydown', handleKeyDown);
      document.removeEventListener('click', handleClickOutside);
    };
  }, [statusMenu]);

  if (jobs.length === 0) {
    return (
      <div style={styles.empty}>
        <p>No jobs found. Search for jobs or add them manually.</p>
      </div>
    );
  }

  return (
    <div style={styles.tableWrapper} onClick={closeStatusMenu}>
      <table style={styles.table}>
          <thead>
          <tr>
            <th style={styles.th} onClick={() => handleSort('Job Title')} title="Sort by Job Title">
              Job Title
              {getSortIndicator('Job Title') && <span style={styles.sortIndicator}>{getSortIndicator('Job Title')}</span>}
            </th>
            <th style={styles.th} onClick={() => handleSort('Company')} title="Sort by Company">
              Company
              {getSortIndicator('Company') && <span style={styles.sortIndicator}>{getSortIndicator('Company')}</span>}
            </th>
            <th style={styles.th} onClick={() => handleSort('Location')} title="Sort by Location">
              Location
              {getSortIndicator('Location') && <span style={styles.sortIndicator}>{getSortIndicator('Location')}</span>}
            </th>
            <th style={styles.th} onClick={() => handleSort('Status')} title="Sort by Status">
              Status
              {getSortIndicator('Status') && <span style={styles.sortIndicator}>{getSortIndicator('Status')}</span>}
            </th>
            <th style={styles.th} onClick={() => handleSort('Updated')} title="Sort by Updated">
              Updated
              {getSortIndicator('Updated') && <span style={styles.sortIndicator}>{getSortIndicator('Updated')}</span>}
            </th>
            <th style={styles.th} onClick={() => handleSort('Source')} title="Sort by Source">
              Source
              {getSortIndicator('Source') && <span style={styles.sortIndicator}>{getSortIndicator('Source')}</span>}
            </th>
            <th style={styles.thNonSortable}>Attachments</th>
            <th style={styles.thNonSortable}>Actions</th>
          </tr>
        </thead>
        <tbody>
          {sortedJobs.map((job) => (
            <tr key={job.id} style={styles.tr}>
              <td style={styles.td}>
                <a
                  href={job.job_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  style={styles.link}
                >
                  {job.job_title}
                </a>
                {job.is_remote && (
                  <span style={styles.remoteBadge}>Remote</span>
                )}
                {job.easy_apply && (
                  <span style={{...styles.remoteBadge, backgroundColor: '#28a745', color: 'white', marginLeft: '6px'}}>Easy Apply</span>
                )}
              </td>
              <td style={styles.td}>{job.company_name || '-'}</td>
              <td style={styles.td}>{job.location || '-'}</td>
              <td style={styles.td}>
                <StatusBadge
                  status={job.status}
                  className="status-badge"
                  onClick={(e) => {
                    e.stopPropagation();
                    handleStatusClick(job, e);
                  }}
                />
              </td>
              <td style={styles.td}>
                {job.updated
                  ? new Date(job.updated).toLocaleDateString('en-US', {
                      month: 'short',
                      day: 'numeric',
                      year: 'numeric',
                    })
                  : '-'}
              </td>
              <td style={styles.td}>
                {(() => {
                  const sourceKey = (job.source || '').toLowerCase();
                  const badgeConfig = SOURCE_BADGES[sourceKey];
                  if (badgeConfig) {
                    return (
                      <span
                        style={{
                          display: 'inline-block',
                          padding: '2px 8px',
                          borderRadius: '4px',
                          fontSize: '11px',
                          fontWeight: 600,
                          backgroundColor: badgeConfig.backgroundColor,
                          color: badgeConfig.color,
                        }}
                      >
                        {badgeConfig.label}
                      </span>
                    );
                  }
                  return <span style={styles.sourceBadge}>manual</span>;
                })()}
              </td>
              <td style={styles.td}>
                {job.attachments?.length > 0 && (
                  <div style={styles.attachmentsContainer}>
                    {job.attachments.map((attachment, idx) => (
                      <button
                        key={attachment.id || idx}
                        onClick={() => downloadAttachment(job.id, attachment.id)}
                        style={styles.attachmentIcon}
                        title={`${attachment.file_type}: ${attachment.file_name}`}
                      >
                        📄
                      </button>
                    ))}
                  </div>
                )}
              </td>
              <td style={styles.td}>
                <button
                  onClick={() => onEdit(job)}
                  style={styles.actionBtn}
                >
                  Edit
                </button>
                <button
                  onClick={() => onDelete(job.id)}
                  style={{ ...styles.actionBtn, color: '#dc3545' }}
                >
                  Delete
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      {statusMenu && (
        <div
          id="status-menu"
          style={{
            position: 'fixed',
            top: statusMenu.position.top,
            left: statusMenu.position.left,
            backgroundColor: 'white',
            border: '1px solid #dee2e6',
            borderRadius: '4px',
            boxShadow: '0 2px 8px rgba(0,0,0,0.15)',
            zIndex: 1000,
            padding: '5px 0',
            minWidth: '150px',
          }}
          onClick={(e) => e.stopPropagation()}
        >
          {STATUSES.filter((s) => s !== jobs.find(j => j.id === statusMenu.jobId)?.status).map((status) => (
            <div
              key={status}
              style={{ padding: '8px 16px', cursor: 'pointer', fontSize: '14px' }}
              onMouseOver={(e) => (e.target.style.backgroundColor = '#f8f9fa')}
              onMouseOut={(e) => (e.target.style.backgroundColor = 'white')}
              onClick={() => handleStatusChange(statusMenu.jobId, status)}
            >
              {status.charAt(0).toUpperCase() + status.slice(1)}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

const styles = {
  tableWrapper: {
    overflowX: 'auto',
    position: 'relative',
  },
  table: {
    width: '100%',
    borderCollapse: 'collapse',
    backgroundColor: 'white',
    borderRadius: '8px',
    overflow: 'hidden',
    boxShadow: '0 1px 3px rgba(0,0,0,0.1)',
  },
  th: {
    backgroundColor: '#f8f9fa',
    padding: '12px',
    textAlign: 'left',
    fontSize: '14px',
    fontWeight: '600',
    color: '#495057',
    borderBottom: '1px solid #dee2e6',
    cursor: 'pointer',
    userSelect: 'none',
  },
  thNonSortable: {
    backgroundColor: '#f8f9fa',
    padding: '12px',
    textAlign: 'left',
    fontSize: '14px',
    fontWeight: '600',
    color: '#495057',
    borderBottom: '1px solid #dee2e6',
  },
  sortIndicator: {
    marginLeft: '6px',
    fontWeight: 'bold',
  },
  tr: {
    borderBottom: '1px solid #dee2e6',
  },
  td: {
    padding: '12px',
    fontSize: '14px',
    verticalAlign: 'middle',
  },
  link: {
    color: '#0d6efd',
    textDecoration: 'none',
    fontWeight: '500',
  },
  remoteBadge: {
    display: 'inline-block',
    marginLeft: '8px',
    padding: '2px 8px',
    backgroundColor: '#d4edda',
    color: '#155724',
    borderRadius: '4px',
    fontSize: '11px',
    fontWeight: '500',
  },
  sourceBadge: {
    display: 'inline-block',
    padding: '2px 8px',
    backgroundColor: '#e7f1ff',
    color: '#0d6efd',
    borderRadius: '4px',
    fontSize: '11px',
    fontWeight: '500',
    textTransform: 'capitalize',
  },
  attachmentsContainer: {
    display: 'flex',
    gap: '4px',
    alignItems: 'center',
    flexWrap: 'wrap',
  },
  attachmentIcon: {
    textDecoration: 'none',
    fontSize: '16px',
    cursor: 'pointer',
    padding: '2px',
    borderRadius: '3px',
    transition: 'background-color 0.2s',
    display: 'inline-flex',
    alignItems: 'center',
    justifyContent: 'center',
  },
  actionBtn: {
    background: 'none',
    border: 'none',
    padding: '4px 8px',
    fontSize: '13px',
    cursor: 'pointer',
    color: '#0d6efd',
    marginRight: '8px',
  },
  empty: {
    textAlign: 'center',
    padding: '40px',
    color: '#6c757d',
    backgroundColor: 'white',
    borderRadius: '8px',
    boxShadow: '0 1px 3px rgba(0,0,0,0.1)',
  },
};
