import { useState } from 'react';

const SITES = [
  { value: 'indeed', label: 'Indeed' },
  { value: 'linkedin', label: 'LinkedIn' },
  { value: 'zip_recruiter', label: 'ZipRecruiter' },
  { value: 'glassdoor', label: 'Glassdoor' },
  { value: 'google', label: 'Google Jobs' },
];

const JOB_TYPES = [
  { value: '', label: 'Any' },
  { value: 'fulltime', label: 'Full-time' },
  { value: 'parttime', label: 'Part-time' },
  { value: 'contract', label: 'Contract' },
  { value: 'internship', label: 'Internship' },
];

const COUNTRIES = [
  { value: 'australia', label: 'Australia' },
  { value: 'singapore', label: 'Singapore' },
  { value: 'usa', label: 'USA' },
];

export default function JobSearch({ onSearch, loading }) {
  const [formData, setFormData] = useState({
    searchTerm: 'software engineer',
    location: '',
    siteNames: {
      indeed: true,
      linkedin: true,
      zip_recruiter: false,
      glassdoor: false,
      google: false,
    },
    jobType: '',
    resultsWanted: 20,
    hoursOld: 72,
    isRemote: false,
    country: 'usa',
  });

  const handleChange = (e) => {
    const { name, value, type, checked } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: type === 'checkbox' ? checked : value,
    }));
  };

  const handleSiteChange = (site) => {
    setFormData((prev) => ({
      ...prev,
      siteNames: {
        ...prev.siteNames,
        [site]: !prev.siteNames[site],
      },
    }));
  };

  const handleSubmit = (e) => {
    e.preventDefault();
    const params = {
      site_names: Object.entries(formData.siteNames)
        .filter(([, selected]) => selected)
        .map(([site]) => site)
        .join(','),
      search_term: formData.searchTerm,
      location: formData.location,
      job_type: formData.jobType || undefined,
      results_wanted: parseInt(formData.resultsWanted),
      hours_old: parseInt(formData.hoursOld),
      is_remote: formData.isRemote,
      country_indeed: formData.country || undefined,
      format: 'json',
    };
    onSearch(params);
  };

  return (
    <form onSubmit={handleSubmit} style={styles.form}>
      <div style={styles.row}>
        <div style={styles.field}>
          <label style={styles.label}>Search Term</label>
          <input
            type="text"
            name="searchTerm"
            value={formData.searchTerm}
            onChange={handleChange}
            style={styles.input}
            required
          />
        </div>
        <div style={styles.field}>
          <label style={styles.label}>Location</label>
          <input
            type="text"
            name="location"
            value={formData.location}
            onChange={handleChange}
            style={styles.input}
            placeholder="e.g. San Francisco, CA or Remote"
          />
        </div>
        <div style={styles.field}>
          <label style={styles.label}>Country</label>
          <select
            name="country"
            value={formData.country}
            onChange={handleChange}
            style={styles.select}
          >
            {COUNTRIES.map((c) => (
              <option key={c.value} value={c.value}>
                {c.label}
              </option>
            ))}
          </select>
        </div>
        <div style={styles.field}>
          <label style={styles.label}>Sites</label>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
            {SITES.map((site) => (
              <label key={site.value} style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                <input
                  type="checkbox"
                  checked={formData.siteNames[site.value] || false}
                  onChange={() => handleSiteChange(site.value)}
                />
                {site.label}
              </label>
            ))}
          </div>
        </div>
      </div>
      <div style={styles.row}>
        <div style={styles.field}>
          <label style={styles.label}>Job Type</label>
          <select
            name="jobType"
            value={formData.jobType}
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
        <div style={styles.field}>
          <label style={styles.label}>Results per Site</label>
          <input
            type="number"
            name="resultsWanted"
            value={formData.resultsWanted}
            onChange={handleChange}
            style={styles.input}
            min="1"
            max="100"
          />
        </div>
        <div style={styles.field}>
          <label style={styles.label}>Hours Old</label>
          <input
            type="number"
            name="hoursOld"
            value={formData.hoursOld}
            onChange={handleChange}
            style={styles.input}
            min="1"
            max="720"
          />
        </div>
        <div style={styles.fieldCheckbox}>
          <label style={styles.checkboxLabel}>
            <input
              type="checkbox"
              name="isRemote"
              checked={formData.isRemote}
              onChange={handleChange}
            />
            Remote Only
          </label>
        </div>
      </div>
      <button type="submit" style={styles.button} disabled={loading}>
        {loading ? 'Searching...' : 'Search Jobs'}
      </button>
    </form>
  );
}

const styles = {
  form: {
    backgroundColor: '#f8f9fa',
    padding: '20px',
    borderRadius: '8px',
    marginBottom: '20px',
    color: '#212529',
  },
  row: {
    display: 'flex',
    gap: '15px',
    marginBottom: '15px',
    flexWrap: 'wrap',
  },
  field: {
    flex: '1 1 200px',
    display: 'flex',
    flexDirection: 'column',
  },
  fieldCheckbox: {
    flex: '1 1 200px',
    display: 'flex',
    alignItems: 'flex-end',
    paddingBottom: '8px',
  },
  label: {
    fontSize: '14px',
    fontWeight: '500',
    marginBottom: '5px',
    color: '#495057',
  },
  checkbox: {
    width: '18px',
    height: '18px',
    cursor: 'pointer',
    accentColor: '#0d6efd',
    marginRight: '8px',
  },
  input: {
    padding: '8px 12px',
    border: '1px solid #ced4da',
    borderRadius: '4px',
    fontSize: '14px',
    backgroundColor: 'white',
    color: '#212529',
  },
  select: {
    padding: '8px 12px',
    border: '1px solid #ced4da',
    borderRadius: '4px',
    fontSize: '14px',
    backgroundColor: 'white',
    color: '#212529',
  },
  checkboxLabel: {
    display: 'flex',
    alignItems: 'center',
    gap: '5px',
    fontSize: '14px',
    color: '#212529',
    cursor: 'pointer',
  },
  button: {
    padding: '10px 24px',
    backgroundColor: '#0d6efd',
    color: 'white',
    border: 'none',
    borderRadius: '4px',
    fontSize: '14px',
    fontWeight: '500',
    cursor: 'pointer',
    width: '100%',
  },
};
