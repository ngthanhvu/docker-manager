import { reactive } from 'vue';
import { dockerApi, setAuthToken } from '../api';

const AUTH_TOKEN_KEY = 'dock-manager.auth.token';

export type AuthUser = {
  id: number;
  username: string;
  createdAt?: string;
};

export const authState = reactive({
  checked: false,
  setupRequired: false,
  token: '',
  user: null as AuthUser | null,
});

let bootstrapPromise: Promise<void> | null = null;

export const initAuth = () => {
  const token = window.localStorage.getItem(AUTH_TOKEN_KEY) || '';
  authState.token = token;
  setAuthToken(token || null);
};

export const applyAuthSession = (token: string, user: AuthUser | null) => {
  authState.token = token;
  authState.user = user;
  authState.setupRequired = false;
  authState.checked = true;
  window.localStorage.setItem(AUTH_TOKEN_KEY, token);
  setAuthToken(token);
};

export const applyAuthToken = (token: string) => {
  authState.token = token;
  authState.user = null;
  authState.setupRequired = false;
  authState.checked = true;
  window.localStorage.setItem(AUTH_TOKEN_KEY, token);
  setAuthToken(token);
};

export const clearAuthSession = () => {
  authState.token = '';
  authState.user = null;
  authState.checked = true;
  window.localStorage.removeItem(AUTH_TOKEN_KEY);
  setAuthToken(null);
};

export const refreshAuthStatus = async () => {
  const { data } = await dockerApi.getAuthStatus();
  authState.setupRequired = !!data?.setupRequired;
  authState.checked = true;

  if (data?.authenticated && data?.user && authState.token) {
    authState.user = data.user;
    return;
  }

  authState.user = null;
  if (!data?.setupRequired) {
    authState.token = '';
    window.localStorage.removeItem(AUTH_TOKEN_KEY);
    setAuthToken(null);
  }
};

export const ensureAuthBootstrap = async () => {
  if (authState.checked) return;
  if (!bootstrapPromise) {
    bootstrapPromise = refreshAuthStatus().finally(() => {
      bootstrapPromise = null;
    });
  }
  await bootstrapPromise;
};

export const signIn = async (payload: { username: string; password: string }) => {
  const { data } = await dockerApi.login(payload);
  applyAuthToken(data.token);
  await refreshAuthStatus();
  return data;
};

export const setupAccount = async (payload: { username: string; password: string }) => {
  const { data } = await dockerApi.setupAuth(payload);
  applyAuthToken(data.token);
  await refreshAuthStatus();
  return data;
};

export const signOut = async () => {
  try {
    if (authState.token) {
      await dockerApi.logout();
    }
  } finally {
    clearAuthSession();
  }
};
