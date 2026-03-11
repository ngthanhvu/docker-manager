import { reactive, watch } from 'vue';

const SETTINGS_KEY = 'docker-manager-settings-v1';

export type AppSettings = {
    general: {
        autoRefreshMs: number;
        confirmDestructive: boolean;
        language: 'vi' | 'en';
        timeFormat: '24h' | '12h';
    };
    ui: {
        theme: 'dark' | 'light';
        density: 'comfortable' | 'compact';
        fontScale: number;
        showSidebarStats: boolean;
    };
    runtime: {
        apiBaseUrl: string;
        defaultLogTail: number;
        terminalShell: '/bin/sh' | '/bin/bash';
        terminalTheme: 'ocean' | 'matrix' | 'amber';
        terminalFontSize: number;
        composeRefreshMs: number;
    };
    notifications: {
        toastDurationMs: number;
        showSuccessToast: boolean;
        showDetailedErrors: boolean;
    };
    safety: {
        softDeleteRequireTyping: boolean;
        protectedResources: string;
    };
    about: {
        appVersion: string;
        buildDate: string;
    };
};

const defaults: AppSettings = {
    general: {
        autoRefreshMs: 5000,
        confirmDestructive: true,
        language: 'vi',
        timeFormat: '24h',
    },
    ui: {
        theme: 'dark',
        density: 'comfortable',
        fontScale: 1,
        showSidebarStats: true,
    },
    runtime: {
        apiBaseUrl: 'http://localhost:8080',
        defaultLogTail: 300,
        terminalShell: '/bin/sh',
        terminalTheme: 'ocean',
        terminalFontSize: 13,
        composeRefreshMs: 5000,
    },
    notifications: {
        toastDurationMs: 2800,
        showSuccessToast: true,
        showDetailedErrors: true,
    },
    safety: {
        softDeleteRequireTyping: false,
        protectedResources: '',
    },
    about: {
        appVersion: '0.1',
        buildDate: '2026-03-09',
    },
};

const clamp = (v: number, min: number, max: number) => Math.min(max, Math.max(min, v));
const terminalThemes = new Set(['ocean', 'matrix', 'amber']);

const normalize = (raw: AppSettings): AppSettings => ({
    ...raw,
    ui: {
        ...raw.ui,
        fontScale: clamp(Number(raw.ui?.fontScale || 1), 0.9, 1.15),
    },
    runtime: {
        ...raw.runtime,
        defaultLogTail: clamp(Number(raw.runtime?.defaultLogTail || 300), 50, 5000),
        terminalFontSize: clamp(Number(raw.runtime?.terminalFontSize || 13), 11, 20),
        terminalTheme: terminalThemes.has(String(raw.runtime?.terminalTheme || 'ocean'))
            ? raw.runtime.terminalTheme as 'ocean' | 'matrix' | 'amber'
            : 'ocean',
        composeRefreshMs: Number(raw.runtime?.composeRefreshMs || 5000),
    },
    notifications: {
        ...raw.notifications,
        toastDurationMs: clamp(Number(raw.notifications?.toastDurationMs || 2800), 1000, 10000),
    },
});

const loadSettings = (): AppSettings => {
    try {
        const raw = localStorage.getItem(SETTINGS_KEY);
        if (!raw) return defaults;
        const parsed = JSON.parse(raw) as Partial<AppSettings>;
        return normalize({
            ...defaults,
            ...parsed,
            general: { ...defaults.general, ...(parsed.general || {}) },
            ui: { ...defaults.ui, ...(parsed.ui || {}) },
            runtime: { ...defaults.runtime, ...(parsed.runtime || {}) },
            notifications: { ...defaults.notifications, ...(parsed.notifications || {}) },
            safety: { ...defaults.safety, ...(parsed.safety || {}) },
            about: { ...defaults.about, ...(parsed.about || {}) },
        });
    } catch {
        return defaults;
    }
};

const applySettings = (settings: AppSettings) => {
    const root = document.documentElement;
    root.setAttribute('data-theme', settings.ui.theme);
    root.style.setProperty('--font-scale', String(settings.ui.fontScale));
    root.style.setProperty('--density-scale', settings.ui.density === 'compact' ? '0.92' : '1');
};

export const appSettings = reactive<AppSettings>(loadSettings());

export const initSettings = () => {
    applySettings(appSettings);
};

watch(
    appSettings,
    (next) => {
        localStorage.setItem(SETTINGS_KEY, JSON.stringify(next));
        applySettings(next);
    },
    { deep: true }
);
