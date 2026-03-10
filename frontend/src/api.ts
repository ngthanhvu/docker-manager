import axios from 'axios';
import { watch } from 'vue';
import { appSettings } from './ui/settings';

const normalizeBase = (raw: string) => raw.replace(/\/+$/, '');
const getApiBase = () => `${normalizeBase(appSettings.runtime.apiBaseUrl)}/api`;

const api = axios.create({
  baseURL: getApiBase(),
});

watch(
  () => appSettings.runtime.apiBaseUrl,
  () => {
    api.defaults.baseURL = getApiBase();
  }
);

export const dockerApi = {
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

  // Docker Compose
  getComposeProjects: () => api.get('/compose/projects'),
  startComposeProject: (name: string) => api.post(`/compose/projects/${encodeURIComponent(name)}/start`),
  stopComposeProject: (name: string) => api.post(`/compose/projects/${encodeURIComponent(name)}/stop`),
  restartComposeProject: (name: string) => api.post(`/compose/projects/${encodeURIComponent(name)}/restart`),
  downComposeProject: (name: string) => api.delete(`/compose/projects/${encodeURIComponent(name)}/down`),
  getComposeProjectLogs: (name: string, tail = 200) =>
    api.get(`/compose/projects/${encodeURIComponent(name)}/logs?tail=${tail}`, { responseType: 'text' }),
  getComposeProjectFiles: (name: string) => api.get(`/compose/projects/${encodeURIComponent(name)}/files`),
};

export const getWsUrl = (path: string) => {
  const url = new URL(normalizeBase(appSettings.runtime.apiBaseUrl));
  const wsProtocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${wsProtocol}//${url.host}/ws${path}`;
};
