import { STORAGE_KEYS } from '../config/constants';

const DEFAULT_USERNAME = 'Operator';
const DEFAULT_ROLE = 1;

const getStorage = () => {
  if (typeof window === 'undefined') return null;
  try {
    return window.localStorage;
  } catch {
    return null;
  }
};

const readStorage = (key) => {
  const storage = getStorage();
  if (!storage) return '';
  try {
    return storage.getItem(key) || '';
  } catch {
    return '';
  }
};

const writeStorage = (key, value) => {
  const storage = getStorage();
  if (!storage) return;
  try {
    storage.setItem(key, value);
  } catch {
    // Ignore storage write failures and keep auth flow functional in memory.
  }
};

const removeStorage = (key) => {
  const storage = getStorage();
  if (!storage) return;
  try {
    storage.removeItem(key);
  } catch {
    // Ignore storage removal failures.
  }
};

const generateDeviceId = () => {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return `web-${crypto.randomUUID()}`;
  }
  return `web-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 12)}`;
};

const decodeBase64Url = (value) => {
  if (!value) return null;
  const atobFn =
    (typeof window !== 'undefined' && typeof window.atob === 'function' && window.atob.bind(window)) ||
    (typeof atob === 'function' && atob);
  if (!atobFn) return null;

  try {
    const normalized = value.replace(/-/g, '+').replace(/_/g, '/');
    const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, '=');
    const binary = atobFn(padded);
    const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0));
    return new TextDecoder().decode(bytes);
  } catch {
    return null;
  }
};

export const getAccessToken = () => readStorage(STORAGE_KEYS.ACCESS_TOKEN);

export const setAccessToken = (token) => {
  if (!token) return;
  writeStorage(STORAGE_KEYS.ACCESS_TOKEN, token);
};

export const clearAccessToken = () => {
  removeStorage(STORAGE_KEYS.ACCESS_TOKEN);
};

export const getStoredUsername = () => readStorage(STORAGE_KEYS.USERNAME);

export const setStoredUsername = (username) => {
  const nextValue = String(username || '').trim();
  if (!nextValue) return;
  writeStorage(STORAGE_KEYS.USERNAME, nextValue);
};

export const clearStoredUsername = () => {
  removeStorage(STORAGE_KEYS.USERNAME);
};

export const getDeviceId = () => readStorage(STORAGE_KEYS.DEVICE_ID);

export const ensureDeviceId = () => {
  const current = getDeviceId();
  if (current) return current;
  const next = generateDeviceId();
  writeStorage(STORAGE_KEYS.DEVICE_ID, next);
  return next;
};

export const clearAuthSession = () => {
  clearAccessToken();
  clearStoredUsername();
};

export const getTokenPayload = (token = getAccessToken()) => {
  if (!token) return null;
  const parts = String(token).split('.');
  if (parts.length < 2) return null;
  const decoded = decodeBase64Url(parts[1]);
  if (!decoded) return null;
  try {
    return JSON.parse(decoded);
  } catch {
    return null;
  }
};

export const isTokenExpired = (payload = getTokenPayload()) => {
  const exp = Number(payload?.exp);
  if (!Number.isFinite(exp) || exp <= 0) return false;
  return Date.now() >= exp * 1000;
};

export const buildUserFromToken = (token = getAccessToken()) => {
  const payload = getTokenPayload(token);
  if (!payload) return null;

  const fallbackUsername = getStoredUsername() || DEFAULT_USERNAME;
  const role = Number(payload.role);

  return {
    userId: String(payload.user_id || ''),
    username: String(payload.username || fallbackUsername),
    role: Number.isFinite(role) ? role : DEFAULT_ROLE,
  };
};
