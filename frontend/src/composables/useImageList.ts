import { ref, onMounted, onUnmounted, computed, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { dockerApi } from '../api';
import { feedback } from '../ui/feedback';
import { appSettings } from '../ui/settings';
import { loadStoredNumber, persistStoredValue } from '../ui/viewState';

export const useImageList = () => {
    const { t } = useI18n();
    const images = ref<any[]>([]);
    const loading = ref(true);
    const currentPage = ref(1);
    const IMAGE_PAGE_SIZE_KEY = 'dock-manager.images.page-size';
    const IMAGE_VIEW_MODE_KEY = 'dock-manager.images.view-mode';
    const pageSize = ref(loadStoredNumber(IMAGE_PAGE_SIZE_KEY, 10, 10, 50));
    const viewMode = ref<'list' | 'card'>(localStorage.getItem(IMAGE_VIEW_MODE_KEY) === 'card' ? 'card' : 'list');
    const pageSizeOptions = [10, 20, 50];
    const selectedIds = ref<string[]>([]);
    const pruning = ref(false);
    const activeCardMenuId = ref<string | null>(null);
    const sortKey = ref<'repoTag' | 'id' | 'size' | 'created'>('created');
    const sortDirection = ref<'asc' | 'desc'>('desc');

    const formatBytes = (bytes?: number) => {
        if (!bytes || bytes <= 0) return '0 B';
        const units = ['B', 'KB', 'MB', 'GB', 'TB'];
        let value = bytes;
        let unitIndex = 0;
        while (value >= 1024 && unitIndex < units.length - 1) {
            value /= 1024;
            unitIndex += 1;
        }
        return `${value.toFixed(value >= 10 || unitIndex === 0 ? 0 : 1)} ${units[unitIndex]}`;
    };

    const fetchImages = async () => {
        try {
            loading.value = true;
            const { data } = await dockerApi.getImages();
            images.value = data || [];
        } catch (err) {
            console.error('Failed to fetch images:', err);
        } finally {
            loading.value = false;
        }
    };

    const removeImage = async (id: string) => {
        const accepted = await feedback.confirmAction({
            title: t('imagesView.deleteTitle'),
            message: t('imagesView.deleteMessage'),
            confirmText: t('common.delete'),
            danger: true,
            requireText: appSettings.safety.softDeleteRequireTyping ? 'DELETE' : undefined,
        });
        if (!accepted) return;
        try {
            await dockerApi.removeImage(id);
            selectedIds.value = selectedIds.value.filter((x) => x !== id);
            await fetchImages();
            feedback.success(t('imagesView.deletedSuccess'));
        } catch (err) {
            feedback.error(t('imagesView.deleteFailed', { error: String(err) }));
        }
    };

    const bulkDelete = async () => {
        if (selectedIds.value.length === 0) return;
        const removeCount = selectedIds.value.length;
        const accepted = await feedback.confirmAction({
            title: t('imagesView.deleteManyTitle'),
            message: t('imagesView.deleteManyMessage', { count: removeCount }),
            confirmText: t('common.delete'),
            danger: true,
            requireText: appSettings.safety.softDeleteRequireTyping ? 'DELETE' : undefined,
        });
        if (!accepted) return;
        try {
            for (const id of selectedIds.value) {
                await dockerApi.removeImage(id);
            }
            selectedIds.value = [];
            await fetchImages();
            feedback.success(t('imagesView.deletedManySuccess', { count: removeCount }));
        } catch (err) {
            feedback.error(t('imagesView.bulkDeleteFailed', { error: String(err) }));
        }
    };

    const pruneImages = async () => {
        if (pruning.value) return;
        const accepted = await feedback.confirmAction({
            title: t('imagesView.pruneTitle'),
            message: t('imagesView.pruneMessage'),
            confirmText: t('common.prune'),
            danger: true,
            requireText: appSettings.safety.softDeleteRequireTyping ? 'PRUNE' : undefined,
        });
        if (!accepted) return;
        try {
            pruning.value = true;
            const { data } = await dockerApi.pruneImages();
            await fetchImages();
            const deletedCount = Array.isArray(data?.ImagesDeleted) ? data.ImagesDeleted.length : 0;
            const reclaimed = formatBytes(Number(data?.SpaceReclaimed || 0));
            feedback.success(t('imagesView.prunedSuccess', { count: deletedCount, size: reclaimed }));
        } catch (err) {
            feedback.error(t('imagesView.pruneFailed', { error: String(err) }));
        } finally {
            pruning.value = false;
        }
    };

    const formatSize = (bytes: number) => `${(bytes / 1024 / 1024).toFixed(1)} MB`;

    const compareValues = (left: string | number, right: string | number) => {
        if (typeof left === 'number' && typeof right === 'number') return left - right;
        return String(left).localeCompare(String(right), undefined, { numeric: true, sensitivity: 'base' });
    };

    const getImageSortValue = (image: any) => {
        if (sortKey.value === 'repoTag') return image.RepoTags?.[0] || '<none>:<none>';
        if (sortKey.value === 'id') return image.Id.substring(7, 19);
        if (sortKey.value === 'size') return Number(image.Size || 0);
        return Number(image.Created || 0);
    };

    const sortedImages = computed(() => {
        const list = [...images.value];
        list.sort((a, b) => {
            const result = compareValues(getImageSortValue(a), getImageSortValue(b));
            return sortDirection.value === 'asc' ? result : -result;
        });
        return list;
    });

    const totalItems = computed(() => sortedImages.value.length);
    const totalPages = computed(() => Math.max(1, Math.ceil(totalItems.value / pageSize.value)));
    const paginatedImages = computed(() => {
        const start = (currentPage.value - 1) * pageSize.value;
        return sortedImages.value.slice(start, start + pageSize.value);
    });
    const pageStart = computed(() => (totalItems.value === 0 ? 0 : (currentPage.value - 1) * pageSize.value + 1));
    const pageEnd = computed(() => Math.min(currentPage.value * pageSize.value, totalItems.value));
    const pageImageIds = computed(() => paginatedImages.value.map((img) => img.Id));
    const selectedCount = computed(() => selectedIds.value.length);
    const allPageSelected = computed(() => pageImageIds.value.length > 0 && pageImageIds.value.every((id) => selectedIds.value.includes(id)));

    const toggleSelect = (id: string) => {
        if (selectedIds.value.includes(id)) selectedIds.value = selectedIds.value.filter((x) => x !== id);
        else selectedIds.value = [...selectedIds.value, id];
    };

    const toggleSelectAllPage = () => {
        if (allPageSelected.value) selectedIds.value = selectedIds.value.filter((id) => !pageImageIds.value.includes(id));
        else selectedIds.value = Array.from(new Set([...selectedIds.value, ...pageImageIds.value]));
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

    const toggleSort = (key: 'repoTag' | 'id' | 'size' | 'created') => {
        if (sortKey.value === key) {
            sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc';
            return;
        }
        sortKey.value = key;
        sortDirection.value = key === 'created' ? 'desc' : 'asc';
    };

    const getSortIndicator = (key: 'repoTag' | 'id' | 'size' | 'created') => {
        if (sortKey.value !== key) return '↕';
        return sortDirection.value === 'asc' ? '↑' : '↓';
    };

    watch(pageSize, () => {
        currentPage.value = 1;
        persistStoredValue(IMAGE_PAGE_SIZE_KEY, pageSize.value);
    });
    watch(viewMode, () => {
        persistStoredValue(IMAGE_VIEW_MODE_KEY, viewMode.value);
    });
    watch(totalPages, (maxPage) => {
        if (currentPage.value > maxPage) currentPage.value = maxPage;
    });
    watch(images, (list) => {
        const valid = new Set(list.map((img) => img.Id));
        selectedIds.value = selectedIds.value.filter((id) => valid.has(id));
    });

    onMounted(() => {
        fetchImages();
        document.addEventListener('click', handleDocumentClick);
    });

    onUnmounted(() => {
        document.removeEventListener('click', handleDocumentClick);
    });

    return {
        t,
        images,
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
        paginatedImages,
        pageStart,
        pageEnd,
        allPageSelected,
        fetchImages,
        removeImage,
        bulkDelete,
        pruneImages,
        formatSize,
        toggleSelect,
        toggleSelectAllPage,
        toggleCardMenu,
        toggleSort,
        getSortIndicator,
    };
};
