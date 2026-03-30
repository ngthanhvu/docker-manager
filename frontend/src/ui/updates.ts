import axios from 'axios';
import { reactive } from 'vue';
import { dockerApi } from '../api';
import { appSettings } from './settings';

type UpdateStatus = 'idle' | 'checking' | 'available' | 'up-to-date' | 'error';

type UpdateCheckResult = {
  latestVersion: string | null;
  updateUrl: string;
  checkedAt: string;
  message: string;
  releaseDate: string | null;
  hasUpdate: boolean;
};

type UpdateApplyResult = {
  started: boolean;
  targetVersion: string;
  message: string;
};

type UpdateApplyStatus = {
  inProgress?: boolean;
  targetVersion?: string;
  message?: string;
  startedAt?: string;
  finishedAt?: string;
  succeeded?: boolean;
};

const state = reactive({
  status: 'idle' as UpdateStatus,
  currentVersion: appSettings.about.appVersion,
  latestVersion: null as string | null,
  message: '',
  checkedAt: '' as string,
  releaseDate: null as string | null,
  updateUrl: '',
  applying: false,
});

const getUpdateUrl = () => {
  const namespace = encodeURIComponent(appSettings.updates.dockerHubNamespace.trim());
  const repoPrefix = encodeURIComponent(appSettings.updates.dockerHubRepoPrefix.trim());
  return `https://hub.docker.com/r/${namespace}/${repoPrefix}-frontend/tags`;
};

const formatError = (error: unknown) => {
  if (axios.isAxiosError(error)) {
    const data = error.response?.data;
    if (typeof data === 'string' && data.trim()) return data.trim();
    if (data && typeof data === 'object' && 'message' in data && typeof data.message === 'string' && data.message.trim()) {
      return data.message.trim();
    }
  }
  if (error instanceof Error && error.message) return error.message;
  return 'Unable to reach Docker Hub right now.';
};

const checkForUpdates = async (): Promise<UpdateCheckResult> => {
  const namespace = appSettings.updates.dockerHubNamespace.trim();
  const repoPrefix = appSettings.updates.dockerHubRepoPrefix.trim();

  if (!namespace || !repoPrefix) {
    throw new Error('Docker Hub namespace and repository prefix are required.');
  }

  const response = await dockerApi.checkAppUpdates({
    currentVersion: appSettings.about.appVersion,
    namespace,
    repoPrefix,
  });
  const payload = response.data as {
    latestVersion?: string | null;
    updateUrl?: string;
    checkedAt?: string;
    message?: string;
    releaseDate?: string | null;
    hasUpdate?: boolean;
  };

  return {
    latestVersion: payload.latestVersion || null,
    updateUrl: payload.updateUrl || getUpdateUrl(),
    checkedAt: payload.checkedAt || new Date().toISOString(),
    releaseDate: payload.releaseDate || null,
    message: payload.message || 'Unable to determine update status.',
    hasUpdate: !!payload.hasUpdate,
  };
};

const refresh = async (opts?: { silent?: boolean }) => {
  state.currentVersion = appSettings.about.appVersion;
  state.status = 'checking';
  if (!opts?.silent) state.message = 'Checking Docker Hub for a newer frontend image...';

  try {
    const result = await checkForUpdates();
    state.latestVersion = result.latestVersion;
    state.checkedAt = result.checkedAt;
    state.releaseDate = result.releaseDate;
    state.updateUrl = result.updateUrl;
    state.message = result.message;
    state.status = result.hasUpdate
      ? 'available'
      : 'up-to-date';
    return result;
  } catch (error) {
    state.status = 'error';
    state.message = formatError(error);
    state.checkedAt = new Date().toISOString();
    state.updateUrl = getUpdateUrl();
    throw error;
  }
};

const openUpdateUrl = () => {
  const target = state.updateUrl || getUpdateUrl();
  window.open(target, '_blank', 'noopener,noreferrer');
};

const syncStatus = async (): Promise<UpdateApplyStatus> => {
  const response = await dockerApi.getAppUpdateStatus();
  const payload = (response.data || {}) as UpdateApplyStatus;
  state.applying = !!payload.inProgress;
  if (payload.targetVersion && !state.latestVersion) {
    state.latestVersion = payload.targetVersion;
  }
  if (payload.message) {
    state.message = payload.message;
  }
  return payload;
};

const apply = async (): Promise<UpdateApplyResult> => {
  const namespace = appSettings.updates.dockerHubNamespace.trim();
  const repoPrefix = appSettings.updates.dockerHubRepoPrefix.trim();

  if (!namespace || !repoPrefix) {
    throw new Error('Docker Hub namespace and repository prefix are required.');
  }
  if (!state.latestVersion) {
    throw new Error('No target version is available yet.');
  }

  state.applying = true;
  try {
    const response = await dockerApi.applyAppUpdate({
      namespace,
      repoPrefix,
      targetVersion: state.latestVersion,
    });
    const payload = response.data as Partial<UpdateApplyResult>;
    state.message = payload.message || `Started updating to version ${state.latestVersion}.`;
    return {
      started: payload.started !== false,
      targetVersion: payload.targetVersion || state.latestVersion || '',
      message: payload.message || state.message,
    };
  } catch (error) {
    try {
      await syncStatus();
    } catch {
      state.applying = false;
    }
    throw new Error(formatError(error));
  }
};

export const updates = {
  state,
  refresh,
  apply,
  syncStatus,
  openUpdateUrl,
};
