import { ref, onMounted, onUnmounted, computed, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { dockerApi } from '../api';
import { feedback } from '../ui/feedback';
import { appSettings } from '../ui/settings';
import { loadStoredNumber, persistStoredValue } from '../ui/viewState';

export const useNetworkList = () => {
    const { t } = useI18n();
    const networks = ref<any[]>([]);
    const loading = ref(true);
    const currentPage = ref(1);
    const NETWORK_PAGE_SIZE_KEY = 'dock-manager.networks.page-size';
    const NETWORK_VIEW_MODE_KEY = 'dock-manager.networks.view-mode';
    const pageSize = ref(loadStoredNumber(NETWORK_PAGE_SIZE_KEY, 10, 10, 50));
    const viewMode = ref<'list' | 'card'>(localStorage.getItem(NETWORK_VIEW_MODE_KEY) === 'card' ? 'card' : 'list');
    const pageSizeOptions = [10, 20, 50];
    const selectedIds = ref<string[]>([]);
    const pruning = ref(false);
    const activeCardMenuId = ref<string | null>(null);
    const sortKey = ref<'name' | 'id' | 'driver' | 'scope' | 'internal'>('name');
    const sortDirection = ref<'asc' | 'desc'>('asc');

    const fetchNetworks = async () => {
        try {
            loading.value = true;
            const { data } = await dockerApi.getNetworks();
            networks.value = data || [];
        } catch (err) {
            console.error('Failed to fetch networks:', err);
        } finally {
            loading.value = false;
        }
    };

    const removeNetwork = async (id: string) => {
        const accepted = await feedback.confirmAction({
            title: t('networksView.deleteTitle'),
            message: t('networksView.deleteMessage'),
            confirmText: t('common.delete'),
            danger: true,
            requireText: appSettings.safety.softDeleteRequireTyping ? 'DELETE' : undefined,
        });
        if (!accepted) return;
        try {
            await dockerApi.removeNetwork(id);
            selectedIds.value = selectedIds.value.filter((x) => x !== id);
            await fetchNetworks();
            feedback.success(t('networksView.deletedSuccess'));
        } catch (err) {
            feedback.error(t('networksView.deleteFailed', { error: String(err) }));
        }
    };

    const bulkDelete = async () => {
        if (selectedIds.value.length === 0) return;
        const removeCount = selectedIds.value.length;
        const accepted = await feedback.confirmAction({
            title: t('networksView.deleteManyTitle'),
            message: t('networksView.deleteManyMessage', { count: removeCount }),
            confirmText: t('common.delete'),
            danger: true,
            requireText: appSettings.safety.softDeleteRequireTyping ? 'DELETE' : undefined,
        });
        if (!accepted) return;
        try {
            for (const id of selectedIds.value) {
                await dockerApi.removeNetwork(id);
            }
            selectedIds.value = [];
            await fetchNetworks();
            feedback.success(t('networksView.deletedManySuccess', { count: removeCount }));
        } catch (err) {
            feedback.error(t('networksView.bulkDeleteFailed', { error: String(err) }));
        }
    };

    const pruneNetworks = async () => {
        if (pruning.value) return;
        const accepted = await feedback.confirmAction({
            title: t('networksView.pruneTitle'),
            message: t('networksView.pruneMessage'),
            confirmText: t('common.prune'),
            danger: true,
            requireText: appSettings.safety.softDeleteRequireTyping ? 'PRUNE' : undefined,
        });
        if (!accepted) return;
        try {
            pruning.value = true;
            const { data } = await dockerApi.pruneNetworks();
            await fetchNetworks();
            const deletedCount = Array.isArray(data?.NetworksDeleted) ? data.NetworksDeleted.length : 0;
            feedback.success(t('networksView.prunedSuccess', { count: deletedCount }));
        } catch (err) {
            feedback.error(t('networksView.pruneFailed', { error: String(err) }));
        } finally {
            pruning.value = false;
        }
    };

    const compareValues = (left: string | number, right: string | number) => {
        if (typeof left === 'number' && typeof right === 'number') return left - right;
        return String(left).localeCompare(String(right), undefined, { numeric: true, sensitivity: 'base' });
    };

    const getNetworkSortValue = (network: any) => {
        if (sortKey.value === 'name') return network.Name || '';
        if (sortKey.value === 'id') return network.Id.substring(0, 12);
        if (sortKey.value === 'driver') return network.Driver || '';
        if (sortKey.value === 'scope') return network.Scope || '';
        return network.Internal ? 1 : 0;
    };

    const sortedNetworks = computed(() => {
        const list = [...networks.value];
        list.sort((a, b) => {
            const result = compareValues(getNetworkSortValue(a), getNetworkSortValue(b));
            return sortDirection.value === 'asc' ? result : -result;
        });
        return list;
    });

    const totalItems = computed(() => sortedNetworks.value.length);
    const totalPages = computed(() => Math.max(1, Math.ceil(totalItems.value / pageSize.value)));
    const paginatedNetworks = computed(() => {
        const start = (currentPage.value - 1) * pageSize.value;
        return sortedNetworks.value.slice(start, start + pageSize.value);
    });
    const pageStart = computed(() => (totalItems.value === 0 ? 0 : (currentPage.value - 1) * pageSize.value + 1));
    const pageEnd = computed(() => Math.min(currentPage.value * pageSize.value, totalItems.value));
    const pageNetworkIds = computed(() => paginatedNetworks.value.map((n) => n.Id));
    const selectedCount = computed(() => selectedIds.value.length);
    const allPageSelected = computed(() => pageNetworkIds.value.length > 0 && pageNetworkIds.value.every((id) => selectedIds.value.includes(id)));

    const toggleSelect = (id: string) => {
        if (selectedIds.value.includes(id)) selectedIds.value = selectedIds.value.filter((x) => x !== id);
        else selectedIds.value = [...selectedIds.value, id];
    };

    const toggleSelectAllPage = () => {
        if (allPageSelected.value) selectedIds.value = selectedIds.value.filter((id) => !pageNetworkIds.value.includes(id));
        else selectedIds.value = Array.from(new Set([...selectedIds.value, ...pageNetworkIds.value]));
    };

    const toggleCardMenu = (id: string) => {
        activeCardMenuId.value = activeCardMenuId.value === id ? null : id;
    };

    const closeCardMenu = () => {
        activeCardMenuId.value = null;
    };

    const handleDocumentClick = (event: MouseEvent) => {
        const target = event.target as HTMLElement | null;
        if (target?.closest('.card-actions-menu')) return;
        closeCardMenu();
    };

    const toggleSort = (key: 'name' | 'id' | 'driver' | 'scope' | 'internal') => {
        if (sortKey.value === key) {
            sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc';
            return;
        }
        sortKey.value = key;
        sortDirection.value = 'asc';
    };

    const getSortIndicator = (key: 'name' | 'id' | 'driver' | 'scope' | 'internal') => {
        if (sortKey.value !== key) return '↕';
        return sortDirection.value === 'asc' ? '↑' : '↓';
    };

    watch(pageSize, () => {
        currentPage.value = 1;
        persistStoredValue(NETWORK_PAGE_SIZE_KEY, pageSize.value);
    });
    watch(viewMode, () => {
        persistStoredValue(NETWORK_VIEW_MODE_KEY, viewMode.value);
    });
    watch(totalPages, (maxPage) => {
        if (currentPage.value > maxPage) currentPage.value = maxPage;
    });
    watch(networks, (list) => {
        const valid = new Set(list.map((n) => n.Id));
        selectedIds.value = selectedIds.value.filter((id) => valid.has(id));
    });

    onMounted(() => {
        fetchNetworks();
        document.addEventListener('click', handleDocumentClick);
    });

    onUnmounted(() => {
        document.removeEventListener('click', handleDocumentClick);
    });

    return {
        t,
        networks,
        loading,
        currentPage,
        pageSize,
        pageSizeOptions,
        viewMode,
        selectedIds,
        pruning,
        activeCardMenuId,
        selectedCount,
        totalItems,
        totalPages,
        paginatedNetworks,
        pageStart,
        pageEnd,
        allPageSelected,
        fetchNetworks,
        removeNetwork,
        bulkDelete,
        pruneNetworks,
        toggleSelect,
        toggleSelectAllPage,
        toggleCardMenu,
        toggleSort,
        getSortIndicator,
    };
};
