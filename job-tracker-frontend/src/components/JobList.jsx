import StatusBadge from './StatusBadge';

const STATUSES = ['new', 'viewed', 'applied', 'rejected', 'shortlisted'];

export default function JobList({ jobs, onStatusChange, onEdit, onDelete }) {
  const getStatusMenuPosition = (e) => {
    const rect = e.target.getBoundingClientRect();
    return {
      top: rect.bottom + window.scrollY + 5,
      left: rect.left + window.scrollX,
    };
  };

  const handleStatusClick = (job, e) => {
    const menu = document.getElementById('status-menu');
    if (menu) {
      menu.remove();
    }

    const newMenu = document.createElement('div');
    newMenu.id = 'status-menu';
    newMenu.style.position = 'absolute';
    const pos = getStatusMenuPosition(e);
    newMenu.style.top = `${pos.top}px`;
    newMenu.style.left = `${pos.left}px}`;
    newMenu.style.backgroundColor = 'white';
    newMenu.style.border = '1px solid #dee2e6';
    newMenu.style.borderRadius = '4px';
    newMenu.style.boxShadow = '0 2px 8px rgba(0,0,0,0.15)';
    newMenu.style.zIndex = '1000';
    newMenu.style.padding = '5px 0';

    STATUSES.filter((s) => s !== job.status).forEach((status) => {
      const item = document.createElement('div');
      item.textContent = status.charAt(0).toUpperCase() + status.slice(1);
      item.style.padding = '8px 16px';
      item.style.cursor = 'pointer';
      item.style.fontSize = '14px';
      item.onmouseover = () => (item.style.backgroundColor = '#f8f9fa');
      item.onmouseout = () => (item.style.backgroundColor = 'white');
      item.onclick = () => {
        onStatusChange(job.id, status);
        document.body.removeChild(newMenu);
      };
      newMenu.appendChild(item);
    });

    document.body.appendChild(newMenu);

    const closeMenu = (e) => {
      if (!newMenu.contains(e.target)) {
        if (document.body.contains(newMenu)) {
          document.body.removeChild(newMenu);
        }
        document.removeEventListener('click', closeMenu);
      }
    };
    setTimeout(() => document.addEventListener('click', closeMenu), 0);
  };

  if (jobs.length === 0) {
    return (
      <div style={styles.empty}>
        <p>No jobs found. Search for jobs or add them manually.</p>
      </div>
    );
  }

  return (
    <div style={styles.tableWrapper}>
      <table style={styles.table}>
        <thead>
          <tr>
            <th style={styles.th}>Job Title</th>
            <th style={styles.th}>Company</th>
            <th style={styles.th}>Location</th>
            <th style={styles.th}>Status</th>
            <th style={styles.th}>Source</th>
            <th style={styles.th}>Attachments</th>
            <th style={styles.th}>Actions</th>
          </tr>
        </thead>
        <tbody>
          {jobs.map((job) => (
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
              </td>
              <td style={styles.td}>{job.company_name || '-'}</td>
              <td style={styles.td}>{job.location || '-'}</td>
              <td style={styles.td}>
                <StatusBadge
                  status={job.status}
                  onClick={(e) => handleStatusClick(job, e)}
                />
              </td>
              <td style={styles.td}>
                <span style={styles.sourceBadge}>
                  {job.source || 'manual'}
                </span>
              </td>
              <td style={styles.td}>
                {job.attachments?.length > 0 && (
                  <span style={styles.attachmentBadge}>
                    📎 {job.attachments.length}
                  </span>
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
    </div>
  );
}

const styles = {
  tableWrapper: {
    overflowX: 'auto',
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
  attachmentBadge: {
    display: 'inline-flex',
    alignItems: 'center',
    gap: '4px',
    padding: '2px 8px',
    backgroundColor: '#e7f1ff',
    color: '#0d6efd',
    borderRadius: '4px',
    fontSize: '12px',
    fontWeight: '500',
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
