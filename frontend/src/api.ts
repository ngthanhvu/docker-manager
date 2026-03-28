import axios from 'axios';
import { watch } from 'vue';
import { appSettings } from './ui/settings';

const normalizeBase = (raw: string) => raw.replace(/\/+$/, '');
const getApiBase = () => `${normalizeBase(appSettings.runtime.apiBaseUrl)}/api`;
let authToken = '';
let unauthorizedHandler: (() => void) | null = null;

const api = axios.create({
  baseURL: getApiBase(),
});

api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error?.response?.status === 401) {
      unauthorizedHandler?.();
    }
    return Promise.reject(error);
  }
);

watch(
  () => appSettings.runtime.apiBaseUrl,
  () => {
    api.defaults.baseURL = getApiBase();
  }
);

export const setAuthToken = (token: string | null) => {
  authToken = token?.trim() || '';
  if (authToken) {
    api.defaults.headers.common.Authorization = `Bearer ${authToken}`;
    return;
  }

  delete api.defaults.headers.common.Authorization;
};

export const setUnauthorizedHandler = (handler: (() => void) | null) => {
  unauthorizedHandler = handler;
};

export const dockerApi = {
  // Auth
  getAuthStatus: () => api.get('/auth/status'),
  setupAuth: (payload: { username: string; password: string }) => api.post('/auth/setup', payload),
  login: (payload: { username: string; password: string }) => api.post('/auth/login', payload),
  logout: () => api.post('/auth/logout'),

  // Containers
  getContainers: () => api.get('/containers'),
  startContainer: (id: string) => api.post(`/containers/${id}/start`),
  stopContainer: (id: string) => api.post(`/containers/${id}/stop`),
  restartContainer: (id: string) => api.post(`/containers/${id}/restart`),
  removeContainer: (id: string) => api.delete(`/containers/${id}/remove`),
  inspectContainer: (id: string) => api.get(`/containers/${id}/inspect`),
  pruneContainers: () => api.post('/containers/prune'),

  // Images
  getImages: () => api.get('/images'),
  removeImage: (id: string) => api.delete(`/images/${id}`),
  pruneImages: () => api.post('/images/prune'),

  // Volumes
  getVolumes: () => api.get('/volumes'),
  removeVolume: (id: string) => api.delete(`/volumes/${id}`),
  pruneVolumes: () => api.post('/volumes/prune'),

  // Networks
  getNetworks: () => api.get('/networks'),
  removeNetwork: (id: string) => api.delete(`/networks/${id}`),
  pruneNetworks: () => api.post('/networks/prune'),

  // System
  getSystemInfo: () => api.get('/info'),
  getDiskUsage: () => api.get('/disk-usage'),
  getDashboardMetrics: () => api.get('/dashboard/metrics'),

  // Docker Compose
  getComposeProjects: () => api.get('/compose/projects'),
  startComposeProject: (name: string) => api.post(`/compose/projects/${encodeURIComponent(name)}/start`),
  stopComposeProject: (name: string) => api.post(`/compose/projects/${encodeURIComponent(name)}/stop`),
  restartComposeProject: (name: string) => api.post(`/compose/projects/${encodeURIComponent(name)}/restart`),
  downComposeProject: (name: string) => api.delete(`/compose/projects/${encodeURIComponent(name)}/down`),
  getComposeProjectLogs: (name: string, tail = 200) =>
    api.get(`/compose/projects/${encodeURIComponent(name)}/logs?tail=${tail}`, { responseType: 'text' }),
  getComposeProjectFiles: (name: string) => api.get(`/compose/projects/${encodeURIComponent(name)}/files`),
  validateComposeProjectFile: (name: string, payload: { path: string; content: string }) =>
    api.post(`/compose/projects/${encodeURIComponent(name)}/files/validate`, payload),
  updateComposeProjectFile: (name: string, payload: { path: string; content: string }) =>
    api.put(`/compose/projects/${encodeURIComponent(name)}/files`, payload),
};

export const getWsUrl = (path: string) => {
  const url = new URL(normalizeBase(appSettings.runtime.apiBaseUrl));
  const wsProtocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
  const wsUrl = new URL(`${wsProtocol}//${url.host}/ws${path}`);
  if (authToken) {
    wsUrl.searchParams.set('token', authToken);
  }
  return wsUrl.toString();
};
