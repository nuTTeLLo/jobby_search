import axios from 'axios';

export const API_BASE = import.meta.env.VITE_API_BASE || 'http://localhost:8080';

const api = axios.create({
  baseURL: API_BASE,
  headers: {
    'Content-Type': 'application/json',
  },
});

export const getJobs = async (status = '') => {
  const params = status ? { status } : {};
  const response = await api.get('/api/jobs', { params });
  return response.data.data || [];
};

export const getJob = async (id) => {
  const response = await api.get(`/api/jobs/${id}`);
  return response.data.data;
};

export const createJob = async (jobData) => {
  const response = await api.post('/api/jobs', jobData);
  return response.data.data;
};

export const updateJob = async (id, jobData) => {
  const response = await api.put(`/api/jobs/${id}`, jobData);
  return response.data.data;
};

export const deleteJob = async (id) => {
  await api.delete(`/api/jobs/${id}`);
};

export const updateJobStatus = async (id, status) => {
  const response = await api.patch(`/api/jobs/${id}/status`, { status });
  return response.data.data;
};

export const searchJobs = async (searchParams) => {
  const response = await api.post('/api/jobs/search', searchParams);
  return response.data.data;
};

export const getAttachments = async (jobId) => {
  const response = await api.get(`/api/jobs/${jobId}/attachments`);
  return response.data.data || [];
};

export const uploadAttachment = async (jobId, file, fileType) => {
  const formData = new FormData();
  formData.append('file', file);
  formData.append('file_type', fileType);
  const response = await api.post(`/api/jobs/${jobId}/attachments`, formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  });
  return response.data.data;
};

export const downloadAttachment = async (jobId, attachmentId) => {
  try {
    const response = await api.get(`/api/jobs/${jobId}/attachments/${attachmentId}/download`, {
      responseType: 'blob',
    });

    // Create a download link
    const url = window.URL.createObjectURL(new Blob([response.data]));
    const link = document.createElement('a');
    link.href = url;

    // Try to get filename from content-disposition header (handles both quoted and unquoted)
    const contentDisposition = response.headers['content-disposition'];
    let fileName = 'attachment';
    if (contentDisposition) {
      // Try quoted format first: filename="name.ext"
      let fileNameMatch = contentDisposition.match(/filename="([^"]+)"/);
      if (fileNameMatch && fileNameMatch[1]) {
        fileName = fileNameMatch[1];
      } else {
        // Try unquoted format: filename=name.ext
        fileNameMatch = contentDisposition.match(/filename=([^;\s]+)/);
        if (fileNameMatch && fileNameMatch[1]) {
          fileName = fileNameMatch[1];
        }
      }
    }

    link.setAttribute('download', fileName);
    link.style.display = 'none';
    document.body.appendChild(link);
    link.click();
    
    // Give the browser time to initiate the download
    setTimeout(() => {
      document.body.removeChild(link);
      window.URL.revokeObjectURL(url);
    }, 100);
  } catch (err) {
    console.error('Download failed:', err);
    alert('Failed to download file. Please try again.');
  }
};

export const deleteAttachment = async (attachmentId) => {
  await api.delete(`/api/jobs/attachments/${attachmentId}`);
};

export default api;
