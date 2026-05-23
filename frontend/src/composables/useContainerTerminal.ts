import { nextTick, ref, type Ref, useTemplateRef, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { Terminal as XTerm } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { getWsUrl } from '../api';
import { feedback } from '../ui/feedback';
import { appSettings } from '../ui/settings';
import '@xterm/xterm/css/xterm.css';

type TerminalThemeName = 'ocean' | 'matrix' | 'amber';

const getTerminalTheme = (themeName: TerminalThemeName) => {
    if (themeName === 'matrix') {
        return {
            foreground: '#d1fae5',
            background: '#03140c',
            cursor: '#22c55e',
            cursorAccent: '#03140c',
            selectionBackground: 'rgba(34, 197, 94, 0.22)',
            black: '#04130a',
            red: '#f87171',
            green: '#22c55e',
            yellow: '#84cc16',
            blue: '#34d399',
            magenta: '#10b981',
            cyan: '#2dd4bf',
            white: '#d1fae5',
            brightBlack: '#166534',
            brightRed: '#fca5a5',
            brightGreen: '#86efac',
            brightYellow: '#bef264',
            brightBlue: '#6ee7b7',
            brightMagenta: '#34d399',
            brightCyan: '#5eead4',
            brightWhite: '#ecfdf5',
        };
    }

    if (themeName === 'amber') {
        return {
            foreground: '#fef3c7',
            background: '#1a1206',
            cursor: '#f59e0b',
            cursorAccent: '#1a1206',
            selectionBackground: 'rgba(245, 158, 11, 0.24)',
            black: '#120d05',
            red: '#fb7185',
            green: '#fbbf24',
            yellow: '#f59e0b',
            blue: '#fcd34d',
            magenta: '#f97316',
            cyan: '#fdba74',
            white: '#fffbeb',
            brightBlack: '#78350f',
            brightRed: '#fda4af',
            brightGreen: '#fde68a',
            brightYellow: '#fcd34d',
            brightBlue: '#fef08a',
            brightMagenta: '#fdba74',
            brightCyan: '#fed7aa',
            brightWhite: '#fff7ed',
        };
    }

    return {
        foreground: '#dbeafe',
        background: '#081121',
        cursor: '#60a5fa',
        cursorAccent: '#081121',
        selectionBackground: 'rgba(96, 165, 250, 0.24)',
        black: '#0f172a',
        red: '#f87171',
        green: '#34d399',
        yellow: '#fbbf24',
        blue: '#60a5fa',
        magenta: '#c084fc',
        cyan: '#22d3ee',
        white: '#e2e8f0',
        brightBlack: '#475569',
        brightRed: '#fca5a5',
        brightGreen: '#86efac',
        brightYellow: '#fde68a',
        brightBlue: '#93c5fd',
        brightMagenta: '#d8b4fe',
        brightCyan: '#67e8f9',
        brightWhite: '#f8fafc',
    };
};

export const useContainerTerminal = (activeContainer: Ref<any | null>) => {
    const { t } = useI18n();
    const showTerminalModal = ref(false);
    const terminalEl = useTemplateRef<HTMLDivElement>('terminalEl');
    const terminalModalExpanded = ref(false);
    const terminalModalPanel = useTemplateRef<HTMLElement>('terminalModalPanel');
    const terminalIsFullscreen = ref(false);
    let terminalSocket: WebSocket | null = null;
    let terminalReconnectTimer: number | null = null;
    let terminalReconnectAttempts = 0;
    let terminalManualClose = false;
    let xterm: XTerm | null = null;
    let fitAddon: FitAddon | null = null;
    let terminalResizeObserver: ResizeObserver | null = null;
    let terminalDataDisposable: { dispose: () => void } | null = null;
    let terminalContainerName = '';

    const terminalThemeOptions = [
        { value: 'ocean', label: t('settings.themeOcean') },
        { value: 'matrix', label: t('settings.themeMatrix') },
        { value: 'amber', label: t('settings.themeAmber') },
    ] as const;

    const getContainerName = (container: any) => container?.Names?.[0]?.replace('/', '') || container?.Id?.substring(0, 12) || '';

    const closeTerminal = () => {
        showTerminalModal.value = false;
        terminalModalExpanded.value = false;
        if (document.fullscreenElement === terminalModalPanel.value) {
            document.exitFullscreen().catch(() => { });
        }
        terminalIsFullscreen.value = false;
        terminalManualClose = true;
        if (terminalReconnectTimer) {
            window.clearTimeout(terminalReconnectTimer);
            terminalReconnectTimer = null;
        }
        if (terminalSocket) {
            terminalSocket.close();
            terminalSocket = null;
        }
        if (terminalResizeObserver && terminalEl.value) {
            terminalResizeObserver.unobserve(terminalEl.value);
        }
        terminalResizeObserver = null;
        if (xterm) {
            if (terminalDataDisposable) {
                terminalDataDisposable.dispose();
                terminalDataDisposable = null;
            }
            xterm.dispose();
            xterm = null;
        }
        fitAddon = null;
    };

    const initTerminalUi = async () => {
        await nextTick();
        if (!terminalEl.value) return;
        xterm = new XTerm({
            cursorBlink: true,
            fontFamily: 'JetBrains Mono, Fira Code, ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, Liberation Mono, monospace',
            fontSize: appSettings.runtime.terminalFontSize,
            convertEol: true,
            theme: getTerminalTheme(appSettings.runtime.terminalTheme),
        });
        fitAddon = new FitAddon();
        xterm.loadAddon(fitAddon);
        xterm.open(terminalEl.value);
        fitAddon.fit();
        xterm.focus();
        terminalDataDisposable = xterm.onData((data) => {
            if (!terminalSocket || terminalSocket.readyState !== WebSocket.OPEN) return;
            terminalSocket.send(data);
        });
        terminalResizeObserver = new ResizeObserver(() => fitAddon?.fit());
        terminalResizeObserver.observe(terminalEl.value);
    };

    const writeTerminal = (text: string) => {
        xterm?.write(text);
    };

    const adjustTerminalFontSize = (delta: number) => {
        appSettings.runtime.terminalFontSize = Math.min(20, Math.max(11, appSettings.runtime.terminalFontSize + delta));
        if (xterm) {
            xterm.options.fontSize = appSettings.runtime.terminalFontSize;
            fitAddon?.fit();
            xterm.focus();
        }
    };

    const toggleTerminalSize = async () => {
        terminalModalExpanded.value = !terminalModalExpanded.value;
        await nextTick();
        fitAddon?.fit();
        xterm?.focus();
    };

    const copyTerminalSelection = async () => {
        const selectedText = xterm?.getSelection()?.trim() || '';
        if (!selectedText) {
            feedback.warning(t('containersView.selectTerminalText'));
            return;
        }
        try {
            await navigator.clipboard.writeText(selectedText);
            feedback.success(t('containersView.selectionCopied'));
        } catch (err) {
            feedback.error(t('containersView.copyFailed', { error: String(err) }));
        }
    };

    const pasteIntoTerminal = async () => {
        if (!terminalSocket || terminalSocket.readyState !== WebSocket.OPEN) {
            feedback.warning(t('containersView.terminalNotConnected'));
            return;
        }
        try {
            const text = await navigator.clipboard.readText();
            if (!text) {
                feedback.warning(t('containersView.clipboardEmpty'));
                return;
            }
            terminalSocket.send(text);
            xterm?.focus();
        } catch (err) {
            feedback.error(t('containersView.pasteFailed', { error: String(err) }));
        }
    };

    const toggleTerminalFullscreen = async () => {
        const panel = terminalModalPanel.value;
        if (!panel) return;
        try {
            if (document.fullscreenElement === panel) {
                await document.exitFullscreen();
                terminalIsFullscreen.value = false;
            } else {
                await panel.requestFullscreen();
                terminalIsFullscreen.value = true;
            }
            await nextTick();
            fitAddon?.fit();
            xterm?.focus();
        } catch (err) {
            feedback.error(t('containersView.fullscreenFailed', { error: String(err) }));
        }
    };

    const handleFullscreenChange = async () => {
        terminalIsFullscreen.value = document.fullscreenElement === terminalModalPanel.value;
        await nextTick();
        fitAddon?.fit();
    };

    const openTerminal = async (container: any) => {
        closeTerminal();
        activeContainer.value = container;
        showTerminalModal.value = true;
        terminalManualClose = false;
        terminalReconnectAttempts = 0;
        terminalContainerName = getContainerName(container);
        await initTerminalUi();

        const connectTerminal = (silent = false) => {
            const shell = encodeURIComponent(appSettings.runtime.terminalShell);
            terminalSocket = new WebSocket(getWsUrl(`/terminal/${container.Id}?shell=${shell}`));
            terminalSocket.onopen = () => {
                terminalReconnectAttempts = 0;
                if (!silent) {
                    writeTerminal(`\r\n${t('containersView.terminalConnected', { name: terminalContainerName })}\r\n`);
                }
                xterm?.focus();
            };
            terminalSocket.onmessage = (event) => writeTerminal(String(event.data));
            terminalSocket.onerror = () => writeTerminal(`\r\n${t('containersView.terminalError')}\r\n`);
            terminalSocket.onclose = () => {
                terminalSocket = null;
                if (terminalManualClose || !showTerminalModal.value) return;
                terminalReconnectAttempts += 1;
                if (terminalReconnectAttempts <= 3) {
                    writeTerminal(`\r\n${t('containersView.terminalReconnect', { attempt: terminalReconnectAttempts })}\r\n`);
                    terminalReconnectTimer = window.setTimeout(() => connectTerminal(true), 900);
                    return;
                }
                writeTerminal(`\r\n${t('containersView.terminalClosed')}\r\n`);
            };
        };

        connectTerminal();
    };

    watch(() => appSettings.runtime.terminalTheme, (themeName) => {
        if (!xterm) return;
        xterm.options.theme = getTerminalTheme(themeName);
        fitAddon?.fit();
    });

    watch(() => appSettings.runtime.terminalFontSize, (fontSize) => {
        if (!xterm) return;
        xterm.options.fontSize = fontSize;
        fitAddon?.fit();
    });

    return {
        showTerminalModal,
        terminalModalExpanded,
        terminalIsFullscreen,
        terminalThemeOptions,
        openTerminal,
        closeTerminal,
        adjustTerminalFontSize,
        toggleTerminalSize,
        copyTerminalSelection,
        pasteIntoTerminal,
        toggleTerminalFullscreen,
        handleFullscreenChange,
    };
};
