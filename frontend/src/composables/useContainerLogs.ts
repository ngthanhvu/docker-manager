import { nextTick, ref, type Ref, useTemplateRef } from 'vue';
import { useI18n } from '../i18n';
import { getWsUrl } from '../api';
import { appSettings } from '../ui/settings';

export const useContainerLogs = (activeContainer: Ref<any | null>) => {
    const { t } = useI18n();
    const showLogsModal = ref(false);
    const logsOutput = ref('');
    const logsEl = useTemplateRef<HTMLElement>('logsEl');
    const logsFollow = ref(true);
    const logsFontSize = ref(13);
    const logsModalExpanded = ref(false);
    let logsSocket: WebSocket | null = null;

    const getContainerName = (container: any) => container?.Names?.[0]?.replace('/', '') || container?.Id?.substring(0, 12) || '';

    const scrollToBottom = async () => {
        await nextTick();
        const el = logsEl.value;
        if (el) el.scrollTop = el.scrollHeight;
    };

    const stripAnsi = (text: string) => text.replace(/\x1B(?:[@-Z\\-_]|\[[0-?]*[ -/]*[@-~])/g, '');

    const isNearBottom = () => {
        const el = logsEl.value;
        if (!el) return true;
        return el.scrollHeight - el.scrollTop - el.clientHeight < 48;
    };

    const appendLogs = (text: string) => {
        const shouldStickToBottom = logsFollow.value && isNearBottom();
        logsOutput.value += stripAnsi(text);
        if (shouldStickToBottom) {
            scrollToBottom();
        }
    };

    const closeLogs = () => {
        showLogsModal.value = false;
        logsModalExpanded.value = false;
        if (logsSocket) {
            logsSocket.close();
            logsSocket = null;
        }
    };

    const openLogs = (container: any) => {
        closeLogs();
        activeContainer.value = container;
        logsOutput.value = '';
        logsFollow.value = true;
        showLogsModal.value = true;

        const tail = Math.max(50, Number(appSettings.runtime.defaultLogTail) || 300);
        logsSocket = new WebSocket(getWsUrl(`/logs/${container.Id}?tail=${tail}`));
        logsSocket.onopen = () => appendLogs(`${t('containersView.logsConnected', { name: getContainerName(container) })}\n`);
        logsSocket.onmessage = (event) => appendLogs(String(event.data));
        logsSocket.onerror = () => appendLogs(`\n${t('containersView.logsError')}\n`);
        logsSocket.onclose = () => appendLogs(`\n${t('containersView.logsClosed')}\n`);
    };

    const handleLogsScroll = () => {
        logsFollow.value = isNearBottom();
    };

    const jumpToLatestLogs = () => {
        logsFollow.value = true;
        scrollToBottom();
    };

    const adjustLogsFontSize = (delta: number) => {
        logsFontSize.value = Math.min(20, Math.max(11, logsFontSize.value + delta));
    };

    const toggleLogsSize = () => {
        logsModalExpanded.value = !logsModalExpanded.value;
    };

    return {
        showLogsModal,
        logsOutput,
        logsFollow,
        logsFontSize,
        logsModalExpanded,
        openLogs,
        closeLogs,
        handleLogsScroll,
        jumpToLatestLogs,
        adjustLogsFontSize,
        toggleLogsSize,
    };
};
