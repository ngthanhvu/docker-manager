<script setup lang="ts">
import { HardDrive, Trash2, RefreshCw, BrushCleaning, List, LayoutGrid, Ellipsis, ChevronLeft, ChevronRight } from 'lucide-vue-next';
import { useVolumeList } from '../composables/useVolumeList';
import dayjs from 'dayjs';

const {
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
} = useVolumeList();
</script>

<template>
    <div class="volume-list-view">
        <div class="toolbar glass-panel">
            <div class="title-with-icon">
                <HardDrive :size="20" class="icon-indigo" />
                <h2>{{ t('volumesView.title') }}</h2>
            </div>
            <div class="toolbar-actions">
                <div class="view-toggle" role="group" :aria-label="t('common.viewMode')">
                    <button class="view-toggle-btn" :class="{ 'is-active': viewMode === 'list' }" type="button"
                        :title="t('common.listView')" @click="viewMode = 'list'">
                        <List :size="16" />
                    </button>
                    <button class="view-toggle-btn" :class="{ 'is-active': viewMode === 'card' }" type="button"
                        :title="t('common.cardView')" @click="viewMode = 'card'">
                        <LayoutGrid :size="16" />
                    </button>
                </div>
                <button class="btn btn-ghost text-danger"
                    :disabled="selectedCount === 0 || bulkDeleting || !!deletingName || pruning" @click="bulkDelete">
                    <RefreshCw v-if="bulkDeleting" :size="16" class="animate-spin" />
                    <Trash2 v-else :size="16" />
                    {{ selectedCount || '' }}
                </button>
                <button class="btn btn-ghost text-warning" :disabled="bulkDeleting || !!deletingName || pruning"
                    @click="pruneVolumes">
                    <RefreshCw v-if="pruning" :size="16" class="animate-spin" />
                    <BrushCleaning v-else :size="16" />
                </button>
                <button class="btn btn-ghost" :disabled="bulkDeleting || !!deletingName || pruning"
                    @click="fetchVolumes">
                    <RefreshCw :size="18" :class="{ 'animate-spin': loading || pruning }" />
                </button>
            </div>
        </div>

        <div v-if="viewMode === 'list'" class="table-container glass-panel">
            <table class="docker-table">
                <thead>
                    <tr>
                        <th class="check-col"><input class="bulk-checkbox" type="checkbox" :checked="allPageSelected"
                                :disabled="bulkDeleting || !!deletingName || pruning" @change="toggleSelectAllPage" />
                        </th>
                        <th class="name-cell"><button class="sort-header" type="button" @click="toggleSort('name')">{{
                            t('volumesView.name') }}<span class="sort-indicator">{{ getSortIndicator('name')
                                    }}</span></button></th>
                        <th><button class="sort-header" type="button" @click="toggleSort('driver')">{{
                            t('volumesView.driver') }}<span class="sort-indicator">{{ getSortIndicator('driver')
                                    }}</span></button></th>
                        <th><button class="sort-header" type="button" @click="toggleSort('mountpoint')">{{
                            t('volumesView.mountpoint') }}<span class="sort-indicator">{{
                                    getSortIndicator('mountpoint') }}</span></button></th>
                        <th class="time-cell"><button class="sort-header" type="button"
                                @click="toggleSort('created')">{{ t('volumesView.created') }}<span
                                    class="sort-indicator">{{ getSortIndicator('created') }}</span></button></th>
                        <th class="actions-cell">{{ t('common.actions') }}</th>
                    </tr>
                </thead>
                <tbody>
                    <tr v-for="vol in paginatedVolumes" :key="vol.Name">
                        <td class="check-col"><input class="bulk-checkbox" type="checkbox"
                                :checked="selectedNames.includes(vol.Name)"
                                :disabled="bulkDeleting || !!deletingName || pruning"
                                @change="toggleSelect(vol.Name)" /></td>
                        <td class="name-cell">{{ vol.Name }}</td>
                        <td>{{ vol.Driver }}</td>
                        <td class="mountpoint-cell">
                            <div class="mountpoint-trigger">
                                <code class="mountpoint-value">{{ vol.Mountpoint || '-' }}</code>
                                <div v-if="vol.Mountpoint" class="mountpoint-tooltip" role="tooltip">
                                    {{ vol.Mountpoint }}
                                </div>
                            </div>
                        </td>
                        <td class="time-cell">{{ vol.CreatedAt ? dayjs(vol.CreatedAt).format('YYYY-MM-DD HH:mm') :
                            t('common.notAvailable') }}</td>
                        <td class="actions-cell">
                            <div class="action-group">
                                <button class="action-btn action-danger"
                                    :disabled="bulkDeleting || !!deletingName || pruning" :title="t('common.remove')"
                                    @click="removeVolume(vol.Name)">
                                    <RefreshCw v-if="deletingName === vol.Name" :size="16" class="animate-spin" />
                                    <Trash2 v-else :size="16" />
                                </button>
                            </div>
                        </td>
                    </tr>
                    <tr v-if="volumes.length === 0 && !loading">
                        <td colspan="6" class="empty-state">{{ t('volumesView.noItems') }}</td>
                    </tr>
                </tbody>
            </table>
        </div>

        <div v-else class="card-container">
            <div v-if="volumes.length === 0 && !loading" class="glass-panel card-empty-state">
                {{ t('volumesView.noItems') }}
            </div>
            <div v-else class="card-grid">
                <article v-for="vol in paginatedVolumes" :key="vol.Name" class="glass-panel entity-card">
                    <div class="entity-card-header">
                        <label class="card-check">
                            <input class="bulk-checkbox" type="checkbox" :checked="selectedNames.includes(vol.Name)"
                                :disabled="bulkDeleting || !!deletingName || pruning"
                                @change="toggleSelect(vol.Name)" />
                        </label>
                        <div class="card-title-block">
                            <div class="name-cell">{{ vol.Name }}</div>
                        </div>
                    </div>
                    <div class="card-meta-list">
                        <div class="card-meta-item">
                            <span class="card-meta-label">{{ t('volumesView.driver') }}</span>
                            <span>{{ vol.Driver }}</span>
                        </div>
                        <div class="card-meta-item">
                            <span class="card-meta-label">{{ t('volumesView.mountpoint') }}</span>
                            <div class="mountpoint-trigger">
                                <code class="mountpoint-value">{{ vol.Mountpoint || '-' }}</code>
                                <div v-if="vol.Mountpoint" class="mountpoint-tooltip" role="tooltip">
                                    {{ vol.Mountpoint }}
                                </div>
                            </div>
                        </div>
                        <div class="card-meta-item">
                            <span class="card-meta-label">{{ t('volumesView.created') }}</span>
                            <span>{{ vol.CreatedAt ? dayjs(vol.CreatedAt).format('YYYY-MM-DD HH:mm') :
                                t('common.notAvailable') }}</span>
                        </div>
                    </div>
                    <div class="card-actions-menu">
                        <button class="action-btn action-neutral card-menu-trigger" type="button"
                            :title="t('common.actions')" @click.stop="toggleCardMenu(vol.Name)">
                            <Ellipsis :size="16" />
                        </button>
                        <div v-if="activeCardMenuId === vol.Name" class="card-actions-popover glass-panel" @click.stop>
                            <button class="card-action-item action-danger" type="button"
                                :disabled="bulkDeleting || !!deletingName || pruning" @click="removeVolume(vol.Name)">
                                <RefreshCw v-if="deletingName === vol.Name" :size="16" class="animate-spin" />
                                <Trash2 v-else :size="16" />
                                {{ t('common.remove') }}
                            </button>
                        </div>
                    </div>
                </article>
            </div>
        </div>

        <div v-if="volumes.length > 0" class="pagination glass-panel">
            <div class="pager-meta">
                <span>{{ t('common.rows') }}</span>
                <select v-model.number="pageSize" :disabled="bulkDeleting || !!deletingName || pruning">
                    <option v-for="size in pageSizeOptions" :key="size" :value="size">{{ size }}</option>
                </select>
                <span>{{ pageStart }}-{{ pageEnd }} / {{ totalItems }}</span>
            </div>
            <div class="pager-actions">
                <button class="btn btn-ghost btn-icon"
                    :disabled="currentPage === 1 || bulkDeleting || !!deletingName || pruning"
                    :aria-label="t('common.prev')" :title="t('common.prev')" @click="currentPage--">
                    <ChevronLeft :size="16" />
                </button>
                <span class="pager-page">{{ t('common.page') }} {{ currentPage }} / {{ totalPages }}</span>
                <button class="btn btn-ghost btn-icon"
                    :disabled="currentPage >= totalPages || bulkDeleting || !!deletingName || pruning"
                    :aria-label="t('common.next')" :title="t('common.next')" @click="currentPage++">
                    <ChevronRight :size="16" />
                </button>
            </div>
        </div>
    </div>
</template>

<style scoped>
.volume-list-view {
    display: flex;
    flex-direction: column;
    gap: 24px;
}

.toolbar {
    padding: 12px 24px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 12px;
}

.toolbar-actions {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
}

.title-with-icon {
    display: flex;
    align-items: center;
    gap: 12px;
}

.title-with-icon h2 {
    font-size: 1.2rem;
    margin: 0;
}

.icon-indigo {
    color: var(--primary);
}

.view-toggle {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 4px;
    border-radius: 12px;
    border: 1px solid var(--glass-border);
    background: var(--glass);
}

.view-toggle-btn {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    min-height: 36px;
    padding: 0 12px;
    border: none;
    border-radius: 9px;
    background: transparent;
    color: var(--text-muted);
    cursor: pointer;
    transition: all 0.18s ease;
}

.view-toggle-btn:hover {
    color: var(--text-main);
    background: rgba(255, 255, 255, 0.04);
}

.view-toggle-btn.is-active {
    background: color-mix(in srgb, var(--primary) 18%, var(--glass));
    color: var(--primary);
}

.table-container {
    overflow: auto;
}

.card-container {
    min-width: 0;
}

.card-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
    gap: 18px;
}

.entity-card {
    position: relative;
    display: flex;
    flex-direction: column;
    gap: 16px;
    padding: 18px;
}

.entity-card-header {
    display: grid;
    grid-template-columns: auto minmax(0, 1fr);
    align-items: flex-start;
    gap: 12px;
}

.card-check {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding-top: 2px;
}

.card-title-block {
    display: flex;
    flex-direction: column;
    gap: 6px;
    min-width: 0;
}

.card-meta-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
}

.card-meta-item {
    display: flex;
    flex-direction: column;
    gap: 6px;
    min-width: 0;
}

.card-meta-label {
    font-size: 0.76rem;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    color: var(--text-muted);
}

.docker-table {
    width: 100%;
    border-collapse: collapse;
}

.docker-table th {
    text-align: left;
    padding: 14px 20px;
    font-size: 0.86rem;
    color: var(--text-muted);
    border-bottom: 1px solid var(--glass-border);
    vertical-align: middle;
}

.docker-table td {
    padding: 14px 20px;
    font-size: 0.88rem;
    border-bottom: 1px solid var(--glass-border);
    vertical-align: middle;
}

.sort-header {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 0;
    border: none;
    background: transparent;
    color: inherit;
    font: inherit;
    cursor: pointer;
}

.sort-indicator {
    font-size: 0.8em;
    color: var(--text-muted);
}

.check-col {
    width: 56px;
    text-align: center !important;
    padding: 10px !important;
}

.bulk-checkbox {
    width: 22px;
    height: 22px;
    cursor: pointer;
    accent-color: var(--primary);
    border-radius: 6px;
}

.bulk-checkbox:hover {
    filter: brightness(1.08);
}

.bulk-checkbox:focus-visible {
    outline: 2px solid rgba(36, 150, 237, 0.55);
    outline-offset: 2px;
}

.docker-table tr:last-child td {
    border-bottom: none;
}

.docker-table tr:hover {
    background: var(--glass);
}

.name-cell {
    font-weight: 600;
    word-break: break-all;
}

.mountpoint-cell {
    width: 300px;
    max-width: 300px;
}

.mountpoint-trigger {
    position: relative;
    width: 100%;
}

.mountpoint-value {
    display: block;
    width: 100%;
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    box-sizing: border-box;
}

.mountpoint-tooltip {
    position: absolute;
    left: 0;
    top: calc(100% + 8px);
    z-index: 20;
    min-width: 100%;
    max-width: min(520px, 60vw);
    padding: 10px 12px;
    border: 1px solid var(--glass-border);
    background: rgba(15, 23, 42, 0.96);
    color: #f8fafc;
    font-size: 0.78rem;
    line-height: 1.45;
    white-space: normal;
    word-break: break-all;
    box-shadow: 0 14px 32px rgba(15, 23, 42, 0.3);
    opacity: 0;
    pointer-events: none;
    transform: translateY(4px);
    transition: opacity 0.16s ease, transform 0.16s ease;
}

.mountpoint-trigger:hover .mountpoint-tooltip {
    opacity: 1;
    transform: translateY(0);
}

.time-cell {
    text-align: right;
    white-space: nowrap;
}

th.time-cell .sort-header {
    justify-content: flex-end;
    width: 100%;
}

.actions-cell {
    width: 100px;
    text-align: center;
}

.action-group {
    display: flex;
    align-items: center;
    justify-content: center;
}

.card-actions-menu {
    position: absolute;
    top: 18px;
    right: 18px;
}

.card-menu-trigger {
    background: rgba(255, 255, 255, 0.05);
}

.card-actions-popover {
    position: absolute;
    top: calc(100% + 8px);
    right: 0;
    display: flex;
    flex-direction: column;
    gap: 6px;
    min-width: 160px;
    padding: 8px;
    z-index: 5;
}

.card-action-item {
    display: inline-flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    min-height: 36px;
    padding: 0 12px;
    border-radius: 10px;
    border: 1px solid var(--glass-border);
    background: rgba(255, 255, 255, 0.03);
    color: var(--text-main);
    cursor: pointer;
    transition: all 0.18s ease;
}

.card-action-item:hover {
    border-color: color-mix(in srgb, var(--primary) 30%, var(--glass-border));
    background: color-mix(in srgb, var(--primary) 8%, var(--glass));
}

.card-action-item:disabled {
    opacity: 0.45;
    cursor: not-allowed;
    transform: none;
}

.action-btn {
    width: 34px;
    height: 34px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border-radius: 10px;
    border: 1px solid var(--glass-border);
    background: rgba(255, 255, 255, 0.03);
    color: var(--text-muted);
    cursor: pointer;
    transition: all 0.18s ease;
}

.action-btn:hover {
    border-color: color-mix(in srgb, var(--primary) 30%, var(--glass-border));
    color: var(--text-main);
}

.action-btn:disabled {
    opacity: 0.45;
    cursor: not-allowed;
    transform: none;
}

.action-danger {
    color: #fda4af;
    border-color: rgba(239, 68, 68, 0.32);
    background: rgba(239, 68, 68, 0.08);
}

.action-danger:hover {
    background: rgba(239, 68, 68, 0.16);
    border-color: rgba(239, 68, 68, 0.55);
}

.empty-state {
    text-align: center;
    color: var(--text-muted);
    padding: 56px 0;
}

.card-empty-state {
    text-align: center;
    color: var(--text-muted);
    padding: 56px 24px;
}

.pagination {
    padding: 10px 14px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 12px;
}

.pager-meta,
.pager-actions {
    display: flex;
    align-items: center;
    gap: 8px;
    color: var(--text-muted);
    font-size: 0.82rem;
}

.pager-meta select {
    background: var(--glass);
    border: 1px solid var(--glass-border);
    color: var(--text-main);
    border-radius: 6px;
    padding: 4px 6px;
}

.pager-page {
    min-width: 92px;
    text-align: center;
}

.animate-spin {
    animation: spin 1s linear infinite;
}

@keyframes spin {
    from {
        transform: rotate(0deg);
    }

    to {
        transform: rotate(360deg);
    }
}

@media (max-width: 900px) {
    .toolbar {
        flex-direction: column;
        align-items: stretch;
    }

    .view-toggle {
        width: 100%;
        justify-content: space-between;
    }

    .view-toggle-btn {
        flex: 1;
        justify-content: center;
    }

    .card-grid {
        grid-template-columns: 1fr;
    }
}

@media (max-width: 1280px) {
    .volume-list-view {
        gap: 18px;
    }

    .toolbar {
        padding: 10px 16px;
        gap: 10px;
        flex-wrap: wrap;
    }

    .toolbar-actions {
        gap: 6px;
    }

    .toolbar-actions .btn {
        min-height: 36px;
        padding: 7px 10px;
        font-size: 0.82rem;
    }

    .view-toggle-btn {
        min-height: 34px;
        padding: 0 10px;
    }

    .docker-table {
        table-layout: fixed;
    }

    .docker-table th,
    .docker-table td {
        overflow: hidden;
        padding: 10px 12px;
        text-overflow: ellipsis;
    }

    .check-col {
        width: 44px;
        min-width: 44px;
    }

    .name-cell {
        width: 28%;
    }

    .mountpoint-cell {
        width: 34%;
        max-width: none;
    }

    .time-cell {
        display: none;
    }

    .actions-cell {
        width: 72px;
    }

    .action-btn {
        width: 28px;
        height: 28px;
        border-radius: 8px;
    }
}

@media (max-width: 1100px) and (min-width: 901px) {
    .mountpoint-cell {
        display: none;
    }

    .name-cell {
        width: 42%;
    }
}
</style>
