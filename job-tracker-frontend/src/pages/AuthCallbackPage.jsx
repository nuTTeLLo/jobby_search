import { useEffect, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { fetchMe } from '../services/api';
import { useAuth } from '../contexts/AuthContext';

// The backend redirects here after Google/GitHub sign-in, with our own JWT in
// the URL fragment (#token=...). Fragments never reach a server, so the token
// stays out of logs; this page picks it up, saves it like a password login
// used to, and wipes it from the address bar.
export default function AuthCallbackPage() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const handled = useRef(false);

  useEffect(() => {
    // StrictMode runs effects twice in dev; the second run would find the
    // fragment already cleared.
    if (handled.current) return;
    handled.current = true;

    const token = new URLSearchParams(window.location.hash.slice(1)).get('token');
    window.history.replaceState(null, '', window.location.pathname);
    if (!token) {
      navigate('/login?error=failed', { replace: true });
      return;
    }

    fetchMe(token)
      .then((user) => {
        login(token, user);
        navigate('/', { replace: true });
      })
      .catch(() => navigate('/login?error=failed', { replace: true }));
  }, [login, navigate]);

  return <p style={styles.message}>Signing you in…</p>;
}

const styles = {
  message: {
    marginTop: '40vh',
    textAlign: 'center',
    fontSize: '14px',
    color: '#6c757d',
  },
};
