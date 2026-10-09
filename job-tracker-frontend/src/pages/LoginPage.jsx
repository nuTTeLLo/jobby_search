import { useSearchParams } from 'react-router-dom';
import { oauthStartUrl } from '../services/api';

const ERROR_MESSAGES = {
  cancelled: 'Sign-in was cancelled.',
  expired: 'Sign-in took too long. Please try again.',
  state_mismatch: 'Sign-in could not be verified. Please try again.',
  not_allowed: 'That account is not allowed to use Job Tracker.',
  no_verified_email: 'Your account has no verified email address.',
  unknown_provider: 'That sign-in option is not available.',
  failed: 'Sign-in failed. Please try again.',
};

const PROVIDERS = [
  { id: 'google', label: 'Sign in with Google' },
  { id: 'github', label: 'Sign in with GitHub' },
];

export default function LoginPage() {
  const [params] = useSearchParams();
  const errorCode = params.get('error');
  const error = errorCode && (ERROR_MESSAGES[errorCode] || ERROR_MESSAGES.failed);

  return (
    <div style={styles.container}>
      <div style={styles.card}>
        <h2 style={styles.title}>Job Tracker</h2>
        <p style={styles.subtitle}>Sign in to your account</p>
        {error && <div style={styles.error}>{error}</div>}
        {PROVIDERS.map(({ id, label }) => (
          <a key={id} style={styles.button} href={oauthStartUrl(id)}>
            {label}
          </a>
        ))}
      </div>
    </div>
  );
}

const styles = {
  container: {
    minHeight: '100vh',
    backgroundColor: '#f5f5f5',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
  },
  card: {
    backgroundColor: 'white',
    borderRadius: '8px',
    padding: '40px',
    boxShadow: '0 2px 8px rgba(0,0,0,0.1)',
    width: '100%',
    maxWidth: '400px',
  },
  title: {
    margin: '0 0 4px 0',
    fontSize: '24px',
    fontWeight: '600',
    color: '#0d6efd',
    textAlign: 'center',
  },
  subtitle: {
    margin: '0 0 24px 0',
    fontSize: '14px',
    color: '#6c757d',
    textAlign: 'center',
  },
  error: {
    backgroundColor: '#f8d7da',
    color: '#721c24',
    padding: '10px 12px',
    borderRadius: '4px',
    marginBottom: '16px',
    fontSize: '14px',
  },
  button: {
    display: 'block',
    width: '100%',
    padding: '10px',
    backgroundColor: '#0d6efd',
    color: 'white',
    border: 'none',
    borderRadius: '4px',
    fontSize: '14px',
    textAlign: 'center',
    textDecoration: 'none',
    boxSizing: 'border-box',
    marginTop: '12px',
  },
};
