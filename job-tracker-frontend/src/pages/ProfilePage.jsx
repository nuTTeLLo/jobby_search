import { useEffect, useState } from 'react';
import AppHeader from '../components/AppHeader';
import { getProfile, updateProfile } from '../services/api';

// Mirrors domain.MaxFitProfileLength in the backend.
const MAX_FIT_PROFILE_LENGTH = 20000;

// A starting point for an empty profile. The headings are only a suggestion: the
// whole text goes to the model as-is.
const FIT_PROFILE_TEMPLATE = `# Experience
- Years and kind of experience (e.g. 8 years full-stack, mostly backend)
- Strong: languages, frameworks, databases, domains
- Some: things you have used but would not lead on
- None: stacks you would not want to be hired for

# What I want
- Roles and seniority (e.g. mid-senior IC, not lead/manager)
- Location and working arrangement (e.g. Melbourne hybrid, at most 2 office days, or remote within Australia)
- Salary floor
- Employment type (permanent, contract)

# Dealbreakers (score 1 whatever the skills match)
- e.g. security clearance or citizenship required
- e.g. fully on-site, or based interstate

# Pluses
- e.g. Elixir, data platform work, product companies over consultancies
`;

// The fit profile: who the daily scrape is scoring postings for. Read by the
// board scrape before each run, so an edit here applies from the next morning.
export default function ProfilePage() {
  const [saved, setSaved] = useState(null);
  const [draft, setDraft] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState(null);

  useEffect(() => {
    (async () => {
      try {
        const profile = await getProfile();
        setSaved(profile);
        setDraft(profile.fit_profile || '');
      } catch (error) {
        setMessage({ text: 'Failed to load profile: ' + error.message, type: 'error' });
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  const dirty = saved !== null && draft !== (saved.fit_profile || '');
  const tooLong = draft.length > MAX_FIT_PROFILE_LENGTH;

  // Closing the tab with unsaved edits loses them, so ask first.
  useEffect(() => {
    if (!dirty) return undefined;
    const warn = (event) => {
      event.preventDefault();
      event.returnValue = '';
    };
    window.addEventListener('beforeunload', warn);
    return () => window.removeEventListener('beforeunload', warn);
  }, [dirty]);

  const handleSave = async () => {
    setSaving(true);
    const submitted = draft;
    try {
      const profile = await updateProfile({ fit_profile: submitted });
      setSaved(profile);
      // Take the saved (trimmed) text, unless the user kept typing while it saved.
      setDraft((current) => (current === submitted ? profile.fit_profile || '' : current));
      setMessage({ text: 'Profile saved', type: 'success' });
      setTimeout(() => setMessage(null), 3000);
    } catch (error) {
      const detail = error.response?.data?.error || error.message;
      setMessage({ text: 'Failed to save profile: ' + detail, type: 'error' });
    } finally {
      setSaving(false);
    }
  };

  return (
    <div style={styles.container}>
      <AppHeader />

      <main style={styles.main}>
        <div style={styles.intro}>
          <h2 style={styles.heading}>Fit profile</h2>
          <p style={styles.subheading}>
            The daily scrape gives this text, with each posting it finds, to a model that
            scores how well the role fits you (1–5) and notes strengths, gaps and
            dealbreakers. Describe your experience and what you want from a role. Changes
            apply from the next run. Leave it empty to turn fit scoring off.
          </p>
        </div>

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

        {loading ? (
          <div style={styles.loading}>Loading...</div>
        ) : (
          <>
            <textarea
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              placeholder="Describe your experience, the roles you want, and your dealbreakers..."
              style={styles.textarea}
              spellCheck
            />

            <div style={styles.footer}>
              <div style={styles.footerLeft}>
                {!draft && (
                  <button onClick={() => setDraft(FIT_PROFILE_TEMPLATE)} style={styles.secondaryBtn}>
                    Start from template
                  </button>
                )}
                <span style={{ ...styles.meta, ...(tooLong ? styles.metaError : {}) }}>
                  {draft.length.toLocaleString()} / {MAX_FIT_PROFILE_LENGTH.toLocaleString()} characters
                </span>
                {saved?.updated_at && (
                  <span style={styles.meta}>
                    Last saved {new Date(saved.updated_at).toLocaleString()}
                  </span>
                )}
              </div>
              <div style={styles.footerRight}>
                {dirty && (
                  <button
                    onClick={() => setDraft(saved.fit_profile || '')}
                    disabled={saving}
                    style={styles.secondaryBtn}
                  >
                    Discard changes
                  </button>
                )}
                <button
                  onClick={handleSave}
                  disabled={!dirty || tooLong || saving}
                  style={{
                    ...styles.primaryBtn,
                    ...(!dirty || tooLong || saving ? styles.primaryBtnDisabled : {}),
                  }}
                >
                  {saving ? 'Saving...' : 'Save'}
                </button>
              </div>
            </div>
          </>
        )}
      </main>
    </div>
  );
}

// Matches the Discovered and Archived pages.
const styles = {
  container: {
    minHeight: '100vh',
    backgroundColor: '#f5f5f5',
  },
  main: {
    maxWidth: '1200px',
    margin: '0 auto',
    padding: '20px',
  },
  intro: {
    marginBottom: '16px',
  },
  heading: {
    margin: '0 0 4px',
    fontSize: '20px',
    fontWeight: 600,
  },
  subheading: {
    margin: 0,
    fontSize: '14px',
    color: '#6c757d',
    maxWidth: '760px',
  },
  message: {
    padding: '12px',
    borderRadius: '4px',
    marginBottom: '16px',
    fontSize: '14px',
  },
  loading: {
    padding: '40px',
    textAlign: 'center',
    color: '#6c757d',
  },
  textarea: {
    width: '100%',
    minHeight: '480px',
    boxSizing: 'border-box',
    padding: '12px',
    fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
    fontSize: '13px',
    lineHeight: 1.5,
    border: '1px solid #ced4da',
    borderRadius: '4px',
    backgroundColor: 'white',
    // index.css sets a light text colour for dark mode; this box is always white.
    color: '#212529',
    resize: 'vertical',
  },
  footer: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    gap: '12px',
    flexWrap: 'wrap',
    marginTop: '12px',
  },
  footerLeft: {
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
    flexWrap: 'wrap',
  },
  footerRight: {
    display: 'flex',
    gap: '8px',
  },
  meta: {
    fontSize: '13px',
    color: '#6c757d',
  },
  metaError: {
    color: '#dc3545',
    fontWeight: 600,
  },
  primaryBtn: {
    padding: '8px 20px',
    backgroundColor: '#0d6efd',
    color: 'white',
    border: 'none',
    borderRadius: '4px',
    fontSize: '14px',
    cursor: 'pointer',
  },
  primaryBtnDisabled: {
    opacity: 0.5,
    cursor: 'default',
  },
  secondaryBtn: {
    padding: '8px 12px',
    fontSize: '13px',
    border: '1px solid #ced4da',
    borderRadius: '4px',
    backgroundColor: 'white',
    cursor: 'pointer',
  },
};
