import { reactive } from 'vue';
import { appSettings } from './settings';

export type ToastType = 'success' | 'error' | 'info' | 'warning';

type ToastItem = {
    id: number;
    message: string;
    type: ToastType;
    duration: number;
};

type ConfirmState = {
    open: boolean;
    title: string;
    message: string;
    confirmText: string;
    cancelText: string;
    danger: boolean;
    requireText: string;
    inputText: string;
};

const state = reactive({
    toasts: [] as ToastItem[],
    confirm: {
        open: false,
        title: '',
        message: '',
        confirmText: 'Confirm',
        cancelText: 'Cancel',
        danger: false,
        requireText: '',
        inputText: '',
    } as ConfirmState,
});

let toastSeed = 1;
let confirmResolver: ((accepted: boolean) => void) | null = null;

const pushToast = (message: string, type: ToastType = 'info', duration = 2800) => {
    const resolvedDuration = duration || appSettings.notifications.toastDurationMs;
    const id = toastSeed++;
    state.toasts.push({ id, message, type, duration: resolvedDuration });
    window.setTimeout(() => {
        const idx = state.toasts.findIndex((item) => item.id === id);
        if (idx >= 0) state.toasts.splice(idx, 1);
    }, resolvedDuration);
};

const dismissToast = (id: number) => {
    state.toasts = state.toasts.filter((item) => item.id !== id);
};

const confirmAction = (opts: {
    title?: string;
    message: string;
    confirmText?: string;
    cancelText?: string;
    danger?: boolean;
    requireText?: string;
}) =>
    new Promise<boolean>((resolve) => {
        if (opts.danger && !appSettings.general.confirmDestructive) {
            resolve(true);
            return;
        }

        if (confirmResolver) {
            confirmResolver(false);
            confirmResolver = null;
        }

        state.confirm.open = true;
        state.confirm.title = opts.title || 'Please Confirm';
        state.confirm.message = opts.message;
        state.confirm.confirmText = opts.confirmText || 'Confirm';
        state.confirm.cancelText = opts.cancelText || 'Cancel';
        state.confirm.danger = !!opts.danger;
        state.confirm.requireText = opts.requireText || '';
        state.confirm.inputText = '';
        confirmResolver = resolve;
    });

const closeConfirm = (accepted: boolean) => {
    state.confirm.open = false;
    const resolver = confirmResolver;
    confirmResolver = null;
    if (resolver) resolver(accepted);
};

export const feedback = {
    state,
    pushToast,
    dismissToast,
    confirmAction,
    closeConfirm,
    success: (message: string) => {
        if (!appSettings.notifications.showSuccessToast) return;
        pushToast(message, 'success', appSettings.notifications.toastDurationMs);
    },
    error: (message: string) => {
        const output = appSettings.notifications.showDetailedErrors
            ? message
            : (message.split(':')[0] || 'Action failed');
        pushToast(output, 'error', Math.max(appSettings.notifications.toastDurationMs, 3200));
    },
    info: (message: string) => pushToast(message, 'info', appSettings.notifications.toastDurationMs),
    warning: (message: string) => pushToast(message, 'warning', appSettings.notifications.toastDurationMs),
};
