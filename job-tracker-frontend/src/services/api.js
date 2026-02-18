import axios from 'axios';

const API_BASE = 'http://localhost:8080';

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

export default api;
