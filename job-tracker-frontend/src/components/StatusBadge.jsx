const statusColors = {
  new: '#6c757d',
  viewed: '#0dcaf0',
  applied: '#0d6efd',
  rejected: '#dc3545',
  shortlisted: '#198754',
};

const statusLabels = {
  new: 'New',
  viewed: 'Viewed',
  applied: 'Applied',
  rejected: 'Rejected',
  shortlisted: 'Shortlisted',
};

export default function StatusBadge({ status, onClick }) {
  const color = statusColors[status] || statusColors.new;
  const label = statusLabels[status] || status;

  return (
    <span
      onClick={onClick}
      style={{
        backgroundColor: color,
        color: 'white',
        padding: '4px 12px',
        borderRadius: '12px',
        fontSize: '12px',
        fontWeight: '500',
        cursor: onClick ? 'pointer' : 'default',
        display: 'inline-block',
      }}
    >
      {label}
    </span>
  );
}
