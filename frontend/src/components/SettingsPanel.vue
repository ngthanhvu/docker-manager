<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from '../i18n';
import { appSettings } from '../ui/settings';
import { updates } from '../ui/updates';
import { feedback } from '../ui/feedback';
import { updateOverlay } from '../ui/updateOverlay';

const props = defineProps<{
  systemInfo?: any;
}>();

const { t } = useI18n();

const apiHint = computed(() => `${appSettings.runtime.apiBaseUrl.replace(/\/+$/, '')}/api`);
const formatVersion = (version?: string | null) => {
  const value = String(version || '').trim();
  if (!value) return t('common.notAvailable');
  return value.toLowerCase().startsWith('v') ? value : `v${value}`;
};
const summaryCards = computed(() => [
  { label: t('settings.appVersion'), value: formatVersion(appSettings.about.appVersion) },
  { label: t('settings.buildDate'), value: appSettings.about.buildDate },
]);
const fontScaleRangeStyle = computed<Record<string, string>>(() => {
  const min = 0.9;
  const max = 1.15;
  const percent = ((appSettings.ui.fontScale - min) / (max - min)) * 100;
  return { '--range-fill': `${Math.min(100, Math.max(0, percent))}%` };
});

const updateState = updates.state;

const updateStatusTone = computed(() => {
  if (updateState.applying) {
    return 'border-color: rgba(96, 165, 250, 0.36); color: #bfdbfe; background: rgba(59, 130, 246, 0.14);';
  }
  switch (updateState.status) {
    case 'available':
      return 'border-color: var(--warning-soft-border); color: var(--warning-soft-text); background: var(--warning-soft-bg);';
    case 'up-to-date':
      return 'border-color: rgba(61, 220, 132, 0.4); color: #bbf7d0; background: rgba(61, 220, 132, 0.12);';
    case 'error':
      return 'border-color: var(--danger-soft-border); color: var(--danger-soft-text); background: var(--danger-soft-bg);';
    default:
      return 'border-color: var(--glass-border); color: var(--text-muted); background: var(--glass);';
  }
});

const checkedAtLabel = computed(() => {
  if (!updateState.checkedAt) return t('settings.neverChecked');
  return new Date(updateState.checkedAt).toLocaleString();
});

const releaseDateLabel = computed(() => {
  if (!updateState.releaseDate) return t('common.notAvailable');
  return new Date(updateState.releaseDate).toLocaleString();
});

const statusLabel = computed(() => {
  if (updateState.applying) {
    return t('settings.updateInProgress');
  }
  switch (updateState.status) {
    case 'checking':
      return t('settings.updateChecking');
    case 'available':
      return t('settings.updateAvailable');
    case 'up-to-date':
      return t('settings.updateUpToDate');
    case 'error':
      return t('settings.updateCheckFailed');
    default:
      return t('settings.updateIdle');
  }
});

const openUpdateConsole = () => {
  updateOverlay.open({
    socketError: t('settings.updateConsoleSocketError'),
    socketClosed: t('settings.updateConsoleSocketClosed'),
  });
};

const checkUpdates = async (silent = false) => {
  try {
    await updates.refresh({ silent });
    if (!silent) {
      if (updateState.status === 'available') feedback.info(updateState.message);
      else if (updateState.status === 'up-to-date') feedback.success(updateState.message);
    }
  } catch {
    if (!silent) feedback.error(updateState.message);
  }
};

const applyUpdate = async () => {
  const accepted = await feedback.confirmAction({
    title: t('common.pleaseConfirm'),
    message: t('settings.updateConfirm', { version: updateState.latestVersion || 'latest' }),
    confirmText: t('settings.updateNow'),
    cancelText: t('common.cancel'),
  });
  if (!accepted) return;

  updateOverlay.beginUpdate({
    socketError: t('settings.updateConsoleSocketError'),
    socketClosed: t('settings.updateConsoleSocketClosed'),
    starting: t('settings.updateConsoleStarting'),
  });

  try {
    const result = await updates.apply();
    feedback.info(result.message || t('settings.updateStarted'));
    updateOverlay.waitForReload({ reloadTimeout: t('settings.updateReloadTimeout') });
  } catch (error) {
    const message = error instanceof Error ? error.message : (updateState.message || t('common.actionFailed'));
    updateOverlay.markFailed(message);
    feedback.error(message);
  }
};
</script>

<template>
  <div class="settings-view flex flex-col gap-4">
    <section>
      <div class="glass-panel settings-hero">
        <div class="min-w-0">
          <h2 class="settings-title">{{ t('settings.title') }}</h2>
          <p class="settings-subtitle">
            {{ t('settings.subtitle') }}
          </p>
        </div>

        <div class="settings-summary">
          <div v-for="card in summaryCards" :key="card.label" class="settings-summary-card">
            <p>{{ card.label }}</p>
            <strong>{{ card.value }}</strong>
          </div>
        </div>
      </div>
    </section>

    <div class="settings-grid">
      <section class="glass-panel settings-section">
        <p class="section-heading">{{ t('settings.general') }}</p>
        <div class="settings-fields">
          <label class="block lg:col-span-2">
            <span class="mb-2 block text-sm font-semibold">{{ t('settings.autoRefresh') }}</span>
            <select v-model.number="appSettings.general.autoRefreshMs" class="app-select">
              <option :value="0">{{ t('settings.off') }}</option>
              <option :value="2000">2s</option>
              <option :value="5000">5s</option>
              <option :value="10000">10s</option>
            </select>
            <small class="mt-2 block text-xs" style="color: var(--text-muted);">{{ t('settings.autoRefreshHelp')
              }}</small>
          </label>

          <label class="settings-switch-row">
            <span class="text-sm font-semibold">{{ t('settings.confirmDestructive') }}</span>
            <span class="settings-switch">
              <input v-model="appSettings.general.confirmDestructive" type="checkbox" class="settings-switch-input" />
              <span class="settings-switch-track"></span>
            </span>
          </label>

          <label class="settings-control-row">
            <span>{{ t('settings.timeFormat') }}</span>
            <select v-model="appSettings.general.timeFormat" class="app-select">
              <option value="24h">{{ t('settings.hour24') }}</option>
              <option value="12h">{{ t('settings.hour12') }}</option>
            </select>
          </label>
        </div>
      </section>

      <section class="glass-panel settings-section">
        <p class="section-heading">{{ t('settings.interface') }}</p>
        <div class="settings-fields">
          <label class="block lg:col-span-2">
            <span class="mb-2 block text-sm font-semibold">{{ t('settings.density') }}</span>
            <select v-model="appSettings.ui.density" class="app-select">
              <option value="comfortable">{{ t('settings.comfortable') }}</option>
              <option value="compact">{{ t('settings.compact') }}</option>
            </select>
          </label>

          <label class="block lg:col-span-2">
            <span class="mb-2 block text-sm font-semibold">{{ t('settings.fontScale') }} ({{
              appSettings.ui.fontScale.toFixed(2) }})</span>
            <div class="settings-range-box">
              <input v-model.number="appSettings.ui.fontScale" type="range" min="0.9" max="1.15" step="0.01"
                class="settings-range-input" :style="fontScaleRangeStyle" />
            </div>
          </label>

          <label class="settings-switch-row lg:col-span-2">
            <span class="text-sm font-semibold">{{ t('settings.showSidebarStats') }}</span>
            <span class="settings-switch">
              <input v-model="appSettings.ui.showSidebarStats" type="checkbox" class="settings-switch-input" />
              <span class="settings-switch-track"></span>
            </span>
          </label>
        </div>
      </section>

      <section class="glass-panel settings-section">
        <p class="section-heading">{{ t('settings.runtime') }}</p>
        <div class="settings-fields">
          <label class="block lg:col-span-2">
            <span class="mb-2 block text-sm font-semibold">{{ t('settings.dockerApiEndpoint') }}</span>
            <input v-model.trim="appSettings.runtime.apiBaseUrl" type="text" placeholder="http://localhost:8080"
              class="app-input" />
            <small class="mt-2 block text-xs" style="color: var(--text-muted);">
              {{ t('settings.dockerApiHelp', { value: apiHint }) }}
            </small>
          </label>

          <label class="block">
            <span class="mb-2 block text-sm font-semibold">{{ t('settings.defaultLogTail') }}</span>
            <input v-model.number="appSettings.runtime.defaultLogTail" type="number" min="50" max="5000" step="50"
              class="app-input" />
          </label>

          <label class="block">
            <span class="mb-2 block text-sm font-semibold">{{ t('settings.terminalShell') }}</span>
            <select v-model="appSettings.runtime.terminalShell" class="app-select">
              <option value="/bin/sh">/bin/sh</option>
              <option value="/bin/bash">/bin/bash</option>
            </select>
          </label>

          <label class="block">
            <span class="mb-2 block text-sm font-semibold">{{ t('settings.terminalTheme') }}</span>
            <select v-model="appSettings.runtime.terminalTheme" class="app-select">
              <option value="system">{{ t('settings.themeSystem') }}</option>
              <option value="light">{{ t('settings.themeLight') }}</option>
              <option value="dark">{{ t('settings.themeDark') }}</option>
            </select>
          </label>

          <label class="block">
            <span class="mb-2 block text-sm font-semibold">{{ t('settings.terminalFontSize') }}</span>
            <input v-model.number="appSettings.runtime.terminalFontSize" type="number" min="11" max="20" step="1"
              class="app-input" />
          </label>

          <label class="block lg:col-span-2">
            <span class="mb-2 block text-sm font-semibold">{{ t('settings.composeRefresh') }}</span>
            <select v-model.number="appSettings.runtime.composeRefreshMs" class="app-select">
              <option :value="0">{{ t('settings.off') }}</option>
              <option :value="2000">2s</option>
              <option :value="5000">5s</option>
              <option :value="10000">10s</option>
            </select>
          </label>
        </div>
      </section>

      <section class="glass-panel settings-section">
        <div class="settings-section-head">
          <p class="section-heading mb-0">{{ t('settings.updates') }}</p>
          <span class="settings-status-pill" :style="updateStatusTone">
            {{ statusLabel }}
          </span>
        </div>

        <div class="settings-fields">
          <label class="settings-switch-row lg:col-span-2">
            <span class="text-sm font-semibold">{{ t('settings.autoCheckUpdates') }}</span>
            <span class="settings-switch">
              <input v-model="appSettings.updates.autoCheck" type="checkbox" class="settings-switch-input" />
              <span class="settings-switch-track"></span>
            </span>
          </label>

          <label class="block">
            <span class="mb-2 block text-sm font-semibold">{{ t('settings.dockerHubNamespace') }}</span>
            <input v-model.trim="appSettings.updates.dockerHubNamespace" type="text" class="app-input"
              placeholder="ngthanhvu" />
          </label>

          <label class="block">
            <span class="mb-2 block text-sm font-semibold">{{ t('settings.dockerHubRepoPrefix') }}</span>
            <input v-model.trim="appSettings.updates.dockerHubRepoPrefix" type="text" class="app-input"
              placeholder="docker-manager" />
          </label>

          <div class="settings-update-card lg:col-span-2">
            <div class="settings-update-grid">
              <div>
                <p>{{
                  t('settings.currentVersion') }}</p>
                <strong>{{ formatVersion(updateState.currentVersion) }}</strong>
              </div>
              <div>
                <p>{{
                  t('settings.latestVersion') }}</p>
                <strong>{{ formatVersion(updateState.latestVersion) }}</strong>
              </div>
              <div>
                <p>{{
                  t('settings.lastChecked') }}</p>
                <span>{{ checkedAtLabel }}</span>
              </div>
              <div>
                <p>{{
                  t('settings.latestPublished') }}</p>
                <span>{{ releaseDateLabel }}</span>
              </div>
            </div>

            <p class="settings-update-message">
              {{ updateState.message || t('settings.updateHelp') }}
            </p>
          </div>

          <div class="settings-actions lg:col-span-2">
            <button class="btn btn-ghost" type="button" :disabled="updateState.status === 'checking'"
              @click="checkUpdates()">
              {{ updateState.status === 'checking' ? t('settings.updateChecking') : t('settings.checkUpdates') }}
            </button>
            <button class="btn btn-primary" type="button"
              :disabled="updateState.status !== 'available' || updateState.applying" @click="applyUpdate">
              {{ t('settings.updateNow') }}
            </button>
            <button class="btn btn-ghost" type="button"
              :disabled="updateState.status === 'checking' || (!updateState.applying && !updateOverlay.state.output)"
              @click="openUpdateConsole">
              {{ t('settings.openUpdateConsole') }}
            </button>
          </div>
        </div>
      </section>

      <section class="glass-panel settings-section">
        <p class="section-heading">{{ t('settings.notifications') }}</p>
        <div class="settings-fields">
          <label class="block lg:col-span-2">
            <span class="mb-2 block text-sm font-semibold">{{ t('settings.toastDuration') }}</span>
            <input v-model.number="appSettings.notifications.toastDurationMs" type="number" min="1000" max="10000"
              step="100" class="app-input" />
          </label>

          <label class="settings-switch-row">
            <span class="text-sm font-semibold">{{ t('settings.showSuccessToasts') }}</span>
            <span class="settings-switch">
              <input v-model="appSettings.notifications.showSuccessToast" type="checkbox"
                class="settings-switch-input" />
              <span class="settings-switch-track"></span>
            </span>
          </label>

          <label class="settings-switch-row">
            <span class="text-sm font-semibold">{{ t('settings.showDetailedErrors') }}</span>
            <span class="settings-switch">
              <input v-model="appSettings.notifications.showDetailedErrors" type="checkbox"
                class="settings-switch-input" />
              <span class="settings-switch-track"></span>
            </span>
          </label>
        </div>
      </section>

      <section class="glass-panel settings-section">
        <p class="section-heading">{{ t('settings.safety') }}</p>
        <div class="settings-fields settings-fields-single">
          <label class="settings-switch-row">
            <span class="text-sm font-semibold">{{ t('settings.requireDeleteTyping') }}</span>
            <span class="settings-switch">
              <input v-model="appSettings.safety.softDeleteRequireTyping" type="checkbox"
                class="settings-switch-input" />
              <span class="settings-switch-track"></span>
            </span>
          </label>

          <label class="block">
            <span class="mb-2 block text-sm font-semibold">{{ t('settings.protectedResources') }}</span>
            <input v-model="appSettings.safety.protectedResources" type="text"
              :placeholder="t('settings.protectedResourcesPlaceholder')" class="app-input" />
          </label>
        </div>
      </section>

      <section class="glass-panel settings-section">
        <p class="section-heading">{{ t('settings.about') }}</p>
        <div class="settings-about-table">
          <div>{{ t('settings.appVersion')
            }}</div>
          <div>{{ formatVersion(appSettings.about.appVersion) }}</div>
          <div>{{ t('settings.buildDate') }}
          </div>
          <div>{{ appSettings.about.buildDate }}</div>
          <div>{{ t('settings.engine') }}
          </div>
          <div>{{ props.systemInfo?.ServerVersion ||
            t('common.notAvailable') }}</div>
          <div>{{ t('settings.os') }}</div>
          <div>{{ props.systemInfo?.OperatingSystem ||
            t('common.notAvailable') }}</div>
        </div>
      </section>
    </div>

  </div>
</template>

<style scoped>
.settings-view {
  max-width: 1480px;
  margin: 0 auto;
  width: 100%;
}

.settings-hero {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  padding: 16px 18px;
}

.settings-title {
  margin: 0;
  font-size: clamp(1.2rem, 1.5vw, 1.6rem);
  line-height: 1.2;
  font-weight: 650;
  letter-spacing: 0;
}

.settings-subtitle {
  margin: 6px 0 0;
  max-width: 760px;
  color: var(--text-muted);
  font-size: 0.84rem;
  line-height: 1.55;
}

.settings-summary {
  display: grid;
  grid-template-columns: repeat(2, minmax(128px, 1fr));
  gap: 8px;
  width: min(360px, 42vw);
  flex: 0 0 auto;
}

.settings-summary-card {
  border: 1px solid var(--glass-border);
  border-radius: 8px;
  background: var(--glass);
  padding: 10px 12px;
  min-width: 0;
}

.settings-summary-card p,
.settings-update-grid p {
  margin: 0;
  color: var(--text-muted);
  font-size: 0.66rem;
  font-weight: 650;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

.settings-summary-card strong {
  display: block;
  margin-top: 5px;
  overflow: hidden;
  color: var(--text-main);
  font-size: 0.98rem;
  font-weight: 650;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.settings-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.settings-section {
  padding: 14px;
  min-width: 0;
}

.settings-section :deep(.section-heading),
.settings-section>.section-heading {
  margin-bottom: 12px;
}

.settings-fields {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  align-items: start;
}

.settings-fields-single {
  grid-template-columns: minmax(0, 1fr);
}

.settings-fields label {
  min-width: 0;
}

.settings-fields label>span:first-child {
  margin-bottom: 6px;
  font-size: 0.8rem;
  font-weight: 600;
}

.settings-fields small {
  margin-top: 6px;
  line-height: 1.45;
}

.settings-fields :deep(.app-input),
.settings-fields :deep(.app-select),
.settings-fields .app-input,
.settings-fields .app-select {
  min-height: 34px;
  border-radius: 7px;
  padding-top: 7px;
  padding-bottom: 7px;
  font-size: 0.84rem;
}

.settings-range-box {
  border: 1px solid var(--glass-border);
  border-radius: 8px;
  background: var(--glass);
  padding: 10px 14px;
}

.settings-range-input {
  width: 100%;
  height: 18px;
  margin: 0;
  accent-color: var(--primary);
  background: transparent;
  cursor: pointer;
}

.settings-range-input::-webkit-slider-runnable-track {
  height: 6px;
  border: 1px solid color-mix(in srgb, var(--primary) 28%, var(--glass-border));
  border-radius: 999px;
  background:
    linear-gradient(var(--primary), var(--primary)) 0 / var(--range-fill, 50%) 100% no-repeat,
    color-mix(in srgb, var(--primary) 8%, var(--glass));
}

.settings-range-input::-webkit-slider-thumb {
  appearance: none;
  width: 16px;
  height: 16px;
  margin-top: -6px;
  border: 2px solid var(--bg-card);
  border-radius: 999px;
  background: var(--primary);
  box-shadow: 0 3px 10px color-mix(in srgb, var(--primary) 34%, transparent);
}

.settings-range-input::-moz-range-track {
  height: 6px;
  border: 1px solid color-mix(in srgb, var(--primary) 28%, var(--glass-border));
  border-radius: 999px;
  background: color-mix(in srgb, var(--primary) 8%, var(--glass));
}

.settings-range-input::-moz-range-progress {
  height: 6px;
  border-radius: 999px;
  background: var(--primary);
}

.settings-range-input::-moz-range-thumb {
  width: 16px;
  height: 16px;
  border: 2px solid var(--bg-card);
  border-radius: 999px;
  background: var(--primary);
  box-shadow: 0 3px 10px color-mix(in srgb, var(--primary) 34%, transparent);
}

.settings-range-input:focus-visible {
  outline: 2px solid color-mix(in srgb, var(--primary) 45%, transparent);
  outline-offset: 4px;
}

.settings-section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 12px;
}

.settings-status-pill {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  border: 1px solid var(--glass-border);
  border-radius: 999px;
  padding: 0 9px;
  font-size: 0.64rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  white-space: nowrap;
}

.settings-update-card {
  border: 1px solid var(--glass-border);
  border-radius: 8px;
  background: var(--glass);
  padding: 12px;
}

.settings-update-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.settings-update-grid strong,
.settings-update-grid span {
  display: block;
  margin-top: 5px;
  overflow: hidden;
  color: var(--text-main);
  font-size: 0.86rem;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.settings-update-message {
  margin: 12px 0 0;
  color: var(--text-muted);
  font-size: 0.82rem;
  line-height: 1.5;
}

.settings-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.settings-actions .btn {
  min-height: 34px;
  padding: 0 11px;
  font-size: 0.82rem;
}

.settings-about-table {
  display: grid;
  grid-template-columns: minmax(150px, 0.42fr) minmax(0, 1fr);
  overflow: hidden;
  border: 1px solid var(--glass-border);
  border-radius: 8px;
  background: var(--glass-border);
  gap: 1px;
  font-size: 0.82rem;
}

.settings-about-table>div {
  min-width: 0;
  padding: 9px 11px;
  background: var(--bg-card);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.settings-about-table>div:nth-child(odd) {
  background: var(--table-header-bg);
  color: var(--text-muted);
  font-weight: 650;
}

.settings-switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  border: 1px solid var(--glass-border);
  border-radius: 8px;
  padding: 9px 11px;
  background: var(--glass);
}

.settings-control-row {
  display: grid;
  grid-template-columns: minmax(120px, 0.7fr) minmax(0, 1fr);
  align-items: center;
  gap: 12px;
  border: 1px solid var(--glass-border);
  border-radius: 8px;
  padding: 4px 11px;
  background: var(--glass);
}

.settings-control-row>span {
  min-width: 0;
  font-size: 0.82rem;
  font-weight: 600;
  line-height: 1.35;
}

.settings-control-row .app-select {
  min-height: 34px;
}

.settings-switch-row>span:first-child {
  min-width: 0;
  font-size: 0.82rem;
  line-height: 1.35;
}

.settings-switch {
  position: relative;
  display: inline-flex;
  align-items: center;
  width: 42px;
  height: 24px;
  flex-shrink: 0;
}

.settings-switch-input {
  position: absolute;
  inset: 0;
  opacity: 0;
  cursor: pointer;
  z-index: 2;
}

.settings-switch-track {
  position: relative;
  width: 100%;
  height: 100%;
  border-radius: 999px;
  background: rgba(148, 163, 184, 0.22);
  border: 1px solid rgba(148, 163, 184, 0.22);
  transition: background 0.18s ease, border-color 0.18s ease;
}

.settings-switch-track::after {
  content: '';
  position: absolute;
  top: 2px;
  left: 2px;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: #f8fafc;
  box-shadow: 0 4px 12px rgba(15, 23, 42, 0.28);
  transition: transform 0.18s ease;
}

.settings-switch-input:checked+.settings-switch-track {
  background: color-mix(in srgb, var(--primary) 70%, transparent);
  border-color: color-mix(in srgb, var(--primary) 72%, transparent);
}

.settings-switch-input:checked+.settings-switch-track::after {
  transform: translateX(18px);
}

.settings-switch-input:focus-visible+.settings-switch-track {
  outline: 2px solid color-mix(in srgb, var(--primary) 55%, transparent);
  outline-offset: 2px;
}

@media (max-width: 1320px) {
  .settings-view {
    gap: 12px;
  }

  .settings-grid {
    gap: 12px;
  }

  .settings-section {
    padding: 12px;
  }

  .settings-update-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 1020px) {

  .settings-grid,
  .settings-fields {
    grid-template-columns: minmax(0, 1fr);
  }

  .settings-hero {
    align-items: stretch;
    flex-direction: column;
  }

  .settings-summary {
    width: 100%;
  }
}

@media (max-width: 640px) {

  .settings-hero,
  .settings-section {
    padding: 11px;
  }

  .settings-summary,
  .settings-update-grid,
  .settings-about-table {
    grid-template-columns: minmax(0, 1fr);
  }

  .settings-status-pill {
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
  }
}
</style>
