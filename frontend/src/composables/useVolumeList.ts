import { ref, onMounted, onUnmounted, computed, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import dayjs from 'dayjs';
import { dockerApi } from '../api';
import { feedback } from '../ui/feedback';
import { appSettings } from '../ui/settings';
import { loadStoredNumber, persistStoredValue } from '../ui/viewState';

export const useVolumeList = () => {
    const { t } = useI18n();
    const volumes = ref<any[]>([]);
    const loading = ref(true);
    const currentPage = ref(1);
    const VOLUME_PAGE_SIZE_KEY = 'dock-manager.volumes.page-size';
    const VOLUME_VIEW_MODE_KEY = 'dock-manager.volumes.view-mode';
    const pageSize = ref(loadStoredNumber(VOLUME_PAGE_SIZE_KEY, 10, 10, 50));
    const viewMode = ref<'list' | 'card'>(localStorage.getItem(VOLUME_VIEW_MODE_KEY) === 'card' ? 'card' : 'list');
    const pageSizeOptions = [10, 20, 50];
    const selectedNames = ref<string[]>([]);
    const deletingName = ref<string | null>(null);
    const bulkDeleting = ref(false);
    const pruning = ref(false);
    const activeCardMenuId = ref<string | null>(null);
    const sortKey = ref<'name' | 'driver' | 'mountpoint' | 'created'>('name');
    const sortDirection = ref<'asc' | 'desc'>('asc');

    const getErrorMessage = (err: unknown) => {
        if (typeof err === 'string') return err;
        if (err && typeof err === 'object') {
            const maybeAxiosError = err as {
                response?: { data?: unknown };
                message?: string;
            };
            if (typeof maybeAxiosError.response?.data === 'string') return maybeAxiosError.response.data;
            if (typeof maybeAxiosError.message === 'string') return maybeAxiosError.message;
        }
        return t('common.unknownError');
    };

    const fetchVolumes = async () => {
        try {
            loading.value = true;
            const { data } = await dockerApi.getVolumes();
            volumes.value = data || [];
        } catch (err) {
            console.error('Failed to fetch volumes:', err);
        } finally {
            loading.value = false;
        }
    };

    const removeVolume = async (name: string) => {
        if (deletingName.value || bulkDeleting.value) return;
        const accepted = await feedback.confirmAction({
            title: t('volumesView.deleteTitle'),
            message: t('volumesView.deleteMessage'),
            confirmText: t('common.delete'),
            danger: true,
            requireText: appSettings.safety.softDeleteRequireTyping ? 'DELETE' : undefined,
        });
        if (!accepted) return;
        try {
            deletingName.value = name;
            await dockerApi.removeVolume(name);
            selectedNames.value = selectedNames.value.filter((x) => x !== name);
            await fetchVolumes();
            feedback.success(t('volumesView.deletedSuccess'));
        } catch (err) {
            feedback.error(t('volumesView.deleteFailed', { error: String(err) }));
        } finally {
            deletingName.value = null;
        }
    };

    const bulkDelete = async () => {
        if (selectedNames.value.length === 0 || bulkDeleting.value || deletingName.value) return;
        const namesToDelete = [...selectedNames.value];
        const removeCount = namesToDelete.length;
        const accepted = await feedback.confirmAction({
            title: t('volumesView.deleteManyTitle'),
            message: t('volumesView.deleteManyMessage', { count: removeCount }),
            confirmText: t('common.delete'),
            danger: true,
            requireText: appSettings.safety.softDeleteRequireTyping ? 'DELETE' : undefined,
        });
        if (!accepted) return;

        const deleted: string[] = [];
        const failed: Array<{ name: string; reason: string }> = [];

        try {
            bulkDeleting.value = true;
            for (const name of namesToDelete) {
                try {
                    await dockerApi.removeVolume(name);
                    deleted.push(name);
                } catch (err) {
                    failed.push({ name, reason: getErrorMessage(err) });
                }
            }

            selectedNames.value = failed.map((item) => item.name);
            await fetchVolumes();

            if (deleted.length > 0) {
                feedback.success(t('volumesView.deletedManySuccess', { deleted: deleted.length, total: removeCount }));
            }

            if (failed.length > 0) {
                const failedNames = failed.map((item) => item.name).join(', ');
                const usedVolumes = failed.filter((item) => item.reason.toLowerCase().includes('in use'));
                const message = usedVolumes.length === failed.length
                    ? t('volumesView.deleteInUseFailed', { count: failed.length, names: failedNames })
                    : t('volumesView.deleteSomeFailed', { count: failed.length, names: failedNames });
                feedback.warning(message);
            }
        } finally {
            bulkDeleting.value = false;
        }
    };

    const pruneVolumes = async () => {
        if (pruning.value || bulkDeleting.value || deletingName.value) return;
        const accepted = await feedback.confirmAction({
            title: t('volumesView.pruneTitle'),
            message: t('volumesView.pruneMessage'),
            confirmText: t('common.prune'),
            danger: true,
            requireText: appSettings.safety.softDeleteRequireTyping ? 'PRUNE' : undefined,
        });
        if (!accepted) return;

        try {
            pruning.value = true;
            const { data } = await dockerApi.pruneVolumes();
            await fetchVolumes();
            const deletedCount = Array.isArray(data?.VolumesDeleted) ? data.VolumesDeleted.length : 0;
            feedback.success(t('volumesView.prunedSuccess', { count: deletedCount }));
        } catch (err) {
            feedback.error(t('volumesView.pruneFailed', { error: getErrorMessage(err) }));
        } finally {
            pruning.value = false;
        }
    };

    const compareValues = (left: string | number, right: string | number) => {
        if (typeof left === 'number' && typeof right === 'number') return left - right;
        return String(left).localeCompare(String(right), undefined, { numeric: true, sensitivity: 'base' });
    };

    const getVolumeSortValue = (volume: any) => {
        if (sortKey.value === 'name') return volume.Name || '';
        if (sortKey.value === 'driver') return volume.Driver || '';
        if (sortKey.value === 'mountpoint') return volume.Mountpoint || '';
        return volume.CreatedAt ? dayjs(volume.CreatedAt).valueOf() : 0;
    };

    const sortedVolumes = computed(() => {
        const list = [...volumes.value];
        list.sort((a, b) => {
            const result = compareValues(getVolumeSortValue(a), getVolumeSortValue(b));
            return sortDirection.value === 'asc' ? result : -result;
        });
        return list;
    });

    const totalItems = computed(() => sortedVolumes.value.length);
    const totalPages = computed(() => Math.max(1, Math.ceil(totalItems.value / pageSize.value)));
    const paginatedVolumes = computed(() => {
        const start = (currentPage.value - 1) * pageSize.value;
        return sortedVolumes.value.slice(start, start + pageSize.value);
    });
    const pageStart = computed(() => (totalItems.value === 0 ? 0 : (currentPage.value - 1) * pageSize.value + 1));
    const pageEnd = computed(() => Math.min(currentPage.value * pageSize.value, totalItems.value));
    const pageVolumeNames = computed(() => paginatedVolumes.value.map((v) => v.Name));
    const selectedCount = computed(() => selectedNames.value.length);
    const allPageSelected = computed(() => pageVolumeNames.value.length > 0 && pageVolumeNames.value.every((n) => selectedNames.value.includes(n)));

    const toggleSelect = (name: string) => {
        if (selectedNames.value.includes(name)) selectedNames.value = selectedNames.value.filter((x) => x !== name);
        else selectedNames.value = [...selectedNames.value, name];
    };

    const toggleSelectAllPage = () => {
        if (allPageSelected.value) selectedNames.value = selectedNames.value.filter((n) => !pageVolumeNames.value.includes(n));
        else selectedNames.value = Array.from(new Set([...selectedNames.value, ...pageVolumeNames.value]));
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

    const toggleSort = (key: 'name' | 'driver' | 'mountpoint' | 'created') => {
        if (sortKey.value === key) {
            sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc';
            return;
        }
        sortKey.value = key;
        sortDirection.value = key === 'created' ? 'desc' : 'asc';
    };

    const getSortIndicator = (key: 'name' | 'driver' | 'mountpoint' | 'created') => {
        if (sortKey.value !== key) return '↕';
        return sortDirection.value === 'asc' ? '↑' : '↓';
    };

    watch(pageSize, () => {
        currentPage.value = 1;
        persistStoredValue(VOLUME_PAGE_SIZE_KEY, pageSize.value);
    });
    watch(viewMode, () => {
        persistStoredValue(VOLUME_VIEW_MODE_KEY, viewMode.value);
    });
    watch(totalPages, (maxPage) => {
        if (currentPage.value > maxPage) currentPage.value = maxPage;
    });
    watch(volumes, (list) => {
        const valid = new Set(list.map((v) => v.Name));
        selectedNames.value = selectedNames.value.filter((n) => valid.has(n));
    });

    onMounted(() => {
        fetchVolumes();
        document.addEventListener('click', handleDocumentClick);
    });

    onUnmounted(() => {
        document.removeEventListener('click', handleDocumentClick);
    });

    return {
        t,
        volumes,
        loading,
        currentPage,
        pageSize,
        pageSizeOptions,
        viewMode,
        selectedNames,
        deletingName,
        bulkDeleting,
        pruning,
        activeCardMenuId,
        selectedCount,
        totalItems,
        totalPages,
        paginatedVolumes,
        pageStart,
        pageEnd,
        allPageSelected,
        fetchVolumes,
        removeVolume,
        bulkDelete,
        pruneVolumes,
        toggleSelect,
        toggleSelectAllPage,
        toggleCardMenu,
        toggleSort,
        getSortIndicator,
    };
};
