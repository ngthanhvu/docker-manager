export const loadStoredString = (key: string, fallback: string) => {
    try {
        const value = localStorage.getItem(key);
        return value ?? fallback;
    } catch {
        return fallback;
    }
};

export const loadStoredNumber = (key: string, fallback: number, min?: number, max?: number) => {
    try {
        const raw = localStorage.getItem(key);
        if (raw == null) return fallback;
        const parsed = Number(raw);
        if (!Number.isFinite(parsed)) return fallback;
        if (typeof min === 'number' && parsed < min) return min;
        if (typeof max === 'number' && parsed > max) return max;
        return parsed;
    } catch {
        return fallback;
    }
};

export const persistStoredValue = (key: string, value: string | number) => {
    try {
        localStorage.setItem(key, String(value));
    } catch {
        // ignore storage failures
    }
};
