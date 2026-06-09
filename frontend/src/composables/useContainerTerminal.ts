import { nextTick, ref, type Ref, useTemplateRef, watch } from 'vue';
import { useI18n } from '../i18n';
import { Terminal as XTerm } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { getWsUrl } from '../api';
import { feedback } from '../ui/feedback';
import { appSettings } from '../ui/settings';
import '@xterm/xterm/css/xterm.css';

type TerminalThemeName = 'system' | 'light' | 'dark';

const getTerminalTheme = (themeName: TerminalThemeName) => {
    const resolvedTheme = themeName === 'system' ? appSettings.ui.theme : themeName;

    if (resolvedTheme === 'light') {
        return {
            foreground: '#24292f',
            background: '#ffffff',
            cursor: '#5865f2',
            cursorAccent: '#ffffff',
            selectionBackground: 'rgba(88, 101, 242, 0.18)',
            black: '#24292f',
            red: '#cf222e',
            green: '#116329',
            yellow: '#953800',
            blue: '#0969da',
            magenta: '#8250df',
            cyan: '#1b7c83',
            white: '#f6f8fa',
            brightBlack: '#57606a',
            brightRed: '#a40e26',
            brightGreen: '#1a7f37',
            brightYellow: '#9a6700',
            brightBlue: '#218bff',
            brightMagenta: '#a475f9',
            brightCyan: '#3192aa',
            brightWhite: '#ffffff',
        };
    }

    return {
        foreground: '#f4f4f5',
        background: '#0f1014',
        cursor: '#7c83ff',
        cursorAccent: '#0f1014',
        selectionBackground: 'rgba(124, 131, 255, 0.24)',
        black: '#18181b',
        red: '#f87171',
        green: '#34d399',
        yellow: '#fbbf24',
        blue: '#60a5fa',
        magenta: '#c084fc',
        cyan: '#22d3ee',
        white: '#e4e4e7',
        brightBlack: '#71717a',
        brightRed: '#fca5a5',
        brightGreen: '#86efac',
        brightYellow: '#fde68a',
        brightBlue: '#93c5fd',
        brightMagenta: '#d8b4fe',
        brightCyan: '#67e8f9',
        brightWhite: '#fafafa',
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
        { value: 'system', label: t('settings.themeSystem') },
        { value: 'light', label: t('settings.themeLight') },
        { value: 'dark', label: t('settings.themeDark') },
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
        terminalResizeObserver = new ResizeObserver(() => {
            requestAnimationFrame(() => fitAddon?.fit());
        });
        terminalResizeObserver.observe(terminalEl.value);
        if (terminalModalPanel.value) {
            terminalResizeObserver.observe(terminalModalPanel.value);
        }
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

    const clearTerminal = () => {
        xterm?.clear();
        xterm?.focus();
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

    watch([() => appSettings.runtime.terminalTheme, () => appSettings.ui.theme], ([themeName]) => {
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
        clearTerminal,
        toggleTerminalFullscreen,
        handleFullscreenChange,
    };
};
