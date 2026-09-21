// Prev/next plus a page selector, so a job 8 pages back is one jump away
// rather than eight clicks. Shared by the tracker and the archive.
export default function Pagination({ page, totalPages, onChange }) {
  if (totalPages <= 1) return null;

  return (
    <div style={styles.pagination}>
      <button
        onClick={() => onChange(Math.max(1, page - 1))}
        disabled={page <= 1}
        style={page <= 1 ? styles.pageBtnDisabled : styles.pageBtn}
      >
        Previous
      </button>

      <label style={styles.pageStatus}>
        Page{' '}
        <select
          value={page}
          onChange={(e) => onChange(Number(e.target.value))}
          style={styles.pageSelect}
        >
          {Array.from({ length: totalPages }, (_, i) => i + 1).map((n) => (
            <option key={n} value={n}>
              {n}
            </option>
          ))}
        </select>{' '}
        of {totalPages}
      </label>

      <button
        onClick={() => onChange(Math.min(totalPages, page + 1))}
        disabled={page >= totalPages}
        style={page >= totalPages ? styles.pageBtnDisabled : styles.pageBtn}
      >
        Next
      </button>
    </div>
  );
}

const styles = {
  pagination: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    gap: '12px',
    padding: '16px 0',
  },
  pageBtn: {
    padding: '6px 14px',
    fontSize: '13px',
    border: '1px solid #ced4da',
    borderRadius: '4px',
    backgroundColor: 'white',
    cursor: 'pointer',
  },
  pageBtnDisabled: {
    padding: '6px 14px',
    fontSize: '13px',
    border: '1px solid #e9ecef',
    borderRadius: '4px',
    backgroundColor: '#f8f9fa',
    color: '#adb5bd',
    cursor: 'not-allowed',
  },
  pageStatus: {
    fontSize: '13px',
    color: '#6c757d',
    display: 'flex',
    alignItems: 'center',
    gap: '6px',
  },
  pageSelect: {
    padding: '4px 6px',
    fontSize: '13px',
    border: '1px solid #ced4da',
    borderRadius: '4px',
    backgroundColor: 'white',
    cursor: 'pointer',
  },
};
