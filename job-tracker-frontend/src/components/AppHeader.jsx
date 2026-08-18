import { NavLink } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';

// Top-level navigation. The two pages are different jobs of work, not two filters
// of the same list: one is the tracker you act on, the other is the read-only feed
// of what the daily scrape found.
const PAGES = [
  { to: '/', label: 'Tracker', end: true },
  { to: '/discovered', label: 'Discovered' },
];

export default function AppHeader() {
  const { user, logout } = useAuth();

  return (
    <header style={styles.header}>
      <div style={styles.left}>
        <h1 style={styles.title}>Job Tracker</h1>
        <nav style={styles.nav}>
          {PAGES.map((page) => (
            <NavLink
              key={page.to}
              to={page.to}
              end={page.end}
              style={({ isActive }) => ({
                ...styles.navLink,
                ...(isActive ? styles.navLinkActive : {}),
              })}
            >
              {page.label}
            </NavLink>
          ))}
        </nav>
      </div>
      <div style={styles.right}>
        <span style={styles.userEmail}>{user?.email}</span>
        <button onClick={logout} style={styles.logoutBtn}>Sign Out</button>
      </div>
    </header>
  );
}

// Carries over the original header's look; only the nav is new.
const styles = {
  header: {
    backgroundColor: '#0d6efd',
    color: 'white',
    padding: '20px',
    boxShadow: '0 2px 4px rgba(0,0,0,0.1)',
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    gap: '16px',
    flexWrap: 'wrap',
  },
  left: {
    display: 'flex',
    alignItems: 'center',
    gap: '24px',
    flexWrap: 'wrap',
  },
  title: {
    margin: 0,
    fontSize: '24px',
    fontWeight: '600',
  },
  nav: {
    display: 'flex',
    gap: '4px',
  },
  navLink: {
    padding: '6px 14px',
    borderRadius: '4px',
    textDecoration: 'none',
    color: 'rgba(255,255,255,0.85)',
    fontSize: '14px',
    fontWeight: 500,
  },
  navLinkActive: {
    backgroundColor: 'rgba(255,255,255,0.2)',
    border: '1px solid rgba(255,255,255,0.4)',
    color: 'white',
  },
  right: {
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
  },
  userEmail: {
    fontSize: '14px',
    opacity: 0.9,
  },
  logoutBtn: {
    padding: '6px 14px',
    backgroundColor: 'rgba(255,255,255,0.2)',
    color: 'white',
    border: '1px solid rgba(255,255,255,0.4)',
    borderRadius: '4px',
    fontSize: '13px',
    cursor: 'pointer',
  },
};
