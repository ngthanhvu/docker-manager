<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { getWsUrl } from '../api';
import { appSettings } from '../ui/settings';
import { updates } from '../ui/updates';
import { feedback } from '../ui/feedback';

const props = defineProps<{
  systemInfo?: any;
}>();

const { t } = useI18n();

const apiHint = computed(() => `${appSettings.runtime.apiBaseUrl.replace(/\/+$/, '')}/api`);
const languageLabelKey = computed(() => {
  switch (appSettings.general.language) {
    case 'vi':
      return 'settings.vietnamese';
    case 'zh':
      return 'settings.chinese';
    default:
      return 'settings.english';
  }
});
const summaryCards = computed(() => [
  { label: t('settings.activeLanguage'), value: t(languageLabelKey.value) },
  { label: t('settings.appVersion'), value: `v${appSettings.about.appVersion}` },
  { label: t('settings.buildDate'), value: appSettings.about.buildDate },
]);

const resetUI = () => {
  appSettings.ui.theme = 'dark';
  appSettings.ui.density = 'comfortable';
  appSettings.ui.fontScale = 1;
  appSettings.ui.showSidebarStats = true;
};

const updateState = updates.state;
const showUpdateConsole = ref(false);
const updateConsoleOutput = ref('');
const updateConsoleEl = ref<HTMLElement | null>(null);
const updateConsoleFollow = ref(true);
const waitingForReload = ref(false);
const updateProgressFailed = ref(false);
let updateConsoleSocket: WebSocket | null = null;

const updateProgress = computed(() => {
  const output = updateConsoleOutput.value.toLowerCase();
  const has = (value: string) => output.includes(value);

  const preparing = updateState.applying || has('[update] preparing version');
  const located = has('locating current docker manager container') || has('compose working directory') || has('resolved docker tag');
  const helperReady = has('pulling helper image') || has('helper image ready') || has('helper container created');
  const deploying = has('helper container started') || has('docker compose pull/up completed') || has('update commands finished');
  const succeeded = !updateState.applying && !!updateState.latestVersion && has('[done] update commands finished');
  const failed = updateProgressFailed.value || output.includes('[error]');

  const steps = [
    {
      key: 'prepare',
      label: t('settings.updatePhasePrepare'),
      state: failed ? (preparing ? 'error' : 'pending') : (located || helperReady || deploying || waitingForReload.value || succeeded ? 'done' : preparing ? 'active' : 'pending'),
    },
    {
      key: 'helper',
      label: t('settings.updatePhaseHelper'),
      state: failed ? (helperReady ? 'error' : 'pending') : (deploying || waitingForReload.value || succeeded ? 'done' : helperReady ? 'active' : 'pending'),
    },
    {
      key: 'deploy',
      label: t('settings.updatePhaseDeploy'),
      state: failed ? (deploying ? 'error' : 'pending') : (waitingForReload.value || succeeded ? 'done' : deploying ? 'active' : 'pending'),
    },
    {
      key: 'reload',
      label: t('settings.updatePhaseReload'),
      state: failed ? 'pending' : (succeeded ? 'done' : waitingForReload.value ? 'active' : 'pending'),
    },
  ] as const;

  let percent = 6;
  if (preparing) percent = 18;
  if (located) percent = 36;
  if (helperReady) percent = 58;
  if (deploying) percent = 82;
  if (waitingForReload.value) percent = 94;
  if (succeeded) percent = 100;
  if (failed) percent = Math.max(percent, 12);

  let title = t('settings.updateConsoleTitle');
  let detail = updateState.message || t('settings.updateConsoleHelp');

  if (failed) {
    title = t('settings.updateFailed');
  } else if (succeeded) {
    title = t('settings.updateCompleted');
    detail = t('settings.updateReloading');
  } else if (waitingForReload.value) {
    title = t('settings.updateReloading');
    detail = t('settings.updateReconnectHelp');
  }

  return { percent, title, detail, failed, succeeded, steps };
});

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

const stripAnsi = (text: string) => text.replace(/\x1B(?:[@-Z\\-_]|\[[0-?]*[ -/]*[@-~])/g, '');

const isUpdateConsoleNearBottom = () => {
  const el = updateConsoleEl.value;
  if (!el) return true;
  return el.scrollHeight - el.scrollTop - el.clientHeight < 48;
};

const scrollUpdateConsoleToBottom = async () => {
  await nextTick();
  const el = updateConsoleEl.value;
  if (el) el.scrollTop = el.scrollHeight;
};

const appendUpdateConsole = (text: string) => {
  const shouldStickToBottom = updateConsoleFollow.value && isUpdateConsoleNearBottom();
  updateConsoleOutput.value += stripAnsi(text);
  if (shouldStickToBottom) {
    void scrollUpdateConsoleToBottom();
  }
};

const handleUpdateConsoleScroll = () => {
  updateConsoleFollow.value = isUpdateConsoleNearBottom();
};

const disconnectUpdateConsole = () => {
  if (updateConsoleSocket) {
    updateConsoleSocket.close();
    updateConsoleSocket = null;
  }
};

const connectUpdateConsole = () => {
  disconnectUpdateConsole();
  updateConsoleSocket = new WebSocket(getWsUrl('/app-updates'));
  updateConsoleSocket.onmessage = (event) => appendUpdateConsole(String(event.data));
  updateConsoleSocket.onerror = () => appendUpdateConsole(`\n[error] ${t('settings.updateConsoleSocketError')}\n`);
  updateConsoleSocket.onclose = () => {
    updateConsoleSocket = null;
    appendUpdateConsole(`\n[closed] ${t('settings.updateConsoleSocketClosed')}\n`);
    void updates.syncStatus().catch(() => undefined);
  };
};

const openUpdateConsole = () => {
  showUpdateConsole.value = true;
  if (!updateConsoleSocket) {
    updateConsoleOutput.value = '';
    updateConsoleFollow.value = true;
    connectUpdateConsole();
  }
};

const closeUpdateConsole = () => {
  showUpdateConsole.value = false;
  disconnectUpdateConsole();
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

const openUpdatePage = () => {
  updates.openUpdateUrl();
};

const applyUpdate = async () => {
  const accepted = await feedback.confirmAction({
    title: t('common.pleaseConfirm'),
    message: t('settings.updateConfirm', { version: updateState.latestVersion || 'latest' }),
    confirmText: t('settings.updateNow'),
    cancelText: t('common.cancel'),
  });
  if (!accepted) return;

  openUpdateConsole();
  updateProgressFailed.value = false;
  waitingForReload.value = false;
  appendUpdateConsole(`[info] ${t('settings.updateConsoleStarting')}\n`);

  try {
    const result = await updates.apply();
    feedback.info(result.message || t('settings.updateStarted'));
    waitingForReload.value = true;
    void updates.waitForAppReload().then((reloaded) => {
      if (!reloaded) {
        waitingForReload.value = false;
        updateProgressFailed.value = true;
        appendUpdateConsole(`\n[warn] ${t('settings.updateReloadTimeout')}\n`);
      }
    });
  } catch (error) {
    waitingForReload.value = false;
    updateProgressFailed.value = true;
    appendUpdateConsole(`[error] ${error instanceof Error ? error.message : t('common.actionFailed')}\n`);
    feedback.error(error instanceof Error ? error.message : (updateState.message || t('common.actionFailed')));
  }
};

onMounted(() => {
  void updates.syncStatus()
    .then((status) => {
      if (status.inProgress) {
        showUpdateConsole.value = true;
        waitingForReload.value = true;
        connectUpdateConsole();
        void updates.waitForAppReload().then((reloaded) => {
          if (!reloaded) {
            waitingForReload.value = false;
            updateProgressFailed.value = true;
          }
        });
      } else if (appSettings.updates.autoCheck && !updateState.checkedAt && updateState.status === 'idle') {
        void checkUpdates(true);
      }
    })
    .catch(() => undefined);
});

onUnmounted(() => {
  disconnectUpdateConsole();
  document.body.style.overflow = '';
});

watch(showUpdateConsole, (open) => {
  document.body.style.overflow = open ? 'hidden' : '';
});
</script>

<template>
  <div class="flex flex-col gap-6">
    <section class="grid gap-5 xl:grid-cols-[1.1fr_0.9fr]">
      <div class="glass-panel p-6">
        <p class="section-heading">{{ t('settings.title') }}</p>
        <h2 class="max-w-3xl text-3xl font-bold tracking-tight">{{ t('settings.title') }}</h2>
        <p class="mt-3 max-w-2xl text-sm leading-7" style="color: var(--text-muted);">
          {{ t('settings.subtitle') }}
        </p>

        <div class="mt-6 grid gap-3 sm:grid-cols-3">
          <div v-for="card in summaryCards" :key="card.label" class="border p-4" style="border-color: var(--glass-border); background: var(--glass);">
            <p class="text-[11px] uppercase tracking-[0.22em]" style="color: var(--text-muted);">{{ card.label }}</p>
            <p class="mt-2 text-2xl font-bold">{{ card.value }}</p>
          </div>
        </div>
      </div>

      <div class="glass-panel p-6">
        <div class="mb-4 flex items-center justify-between gap-4">
          <p class="section-heading mb-0">{{ t('settings.quickActions') }}</p>
          <span class="border px-3 py-1 text-xs font-semibold uppercase tracking-[0.18em]" style="border-color: var(--primary); color: var(--primary);">
            {{ t('settings.statusLive') }}
          </span>
        </div>

        <div class="grid gap-4">
          <label class="block">
            <span class="mb-2 block text-sm font-semibold">{{ t('settings.language') }}</span>
            <select v-model="appSettings.general.language" class="app-select">
              <option value="vi">{{ t('settings.vietnamese') }}</option>
              <option value="en">{{ t('settings.english') }}</option>
              <option value="zh">{{ t('settings.chinese') }}</option>
            </select>
          </label>

          <label class="block">
            <span class="mb-2 block text-sm font-semibold">{{ t('settings.theme') }}</span>
            <select v-model="appSettings.ui.theme" class="app-select">
              <option value="dark">{{ t('settings.dark') }}</option>
              <option value="light">{{ t('settings.light') }}</option>
            </select>
          </label>

          <button class="btn btn-ghost w-full" @click="resetUI">{{ t('settings.resetUi') }}</button>
        </div>
      </div>
    </section>

    <div class="grid gap-6 2xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
      <section class="glass-panel p-5">
        <p class="section-heading">{{ t('settings.general') }}</p>
        <div class="grid gap-4 lg:grid-cols-2">
          <label class="block lg:col-span-2">
            <span class="mb-2 block text-sm font-semibold">{{ t('settings.autoRefresh') }}</span>
            <select v-model.number="appSettings.general.autoRefreshMs" class="app-select">
              <option :value="0">{{ t('settings.off') }}</option>
              <option :value="2000">2s</option>
              <option :value="5000">5s</option>
              <option :value="10000">10s</option>
            </select>
            <small class="mt-2 block text-xs" style="color: var(--text-muted);">{{ t('settings.autoRefreshHelp') }}</small>
          </label>

          <label class="settings-switch-row">
            <span class="text-sm font-semibold">{{ t('settings.confirmDestructive') }}</span>
            <span class="settings-switch">
              <input v-model="appSettings.general.confirmDestructive" type="checkbox" class="settings-switch-input" />
              <span class="settings-switch-track"></span>
            </span>
          </label>

          <label class="block">
            <span class="mb-2 block text-sm font-semibold">{{ t('settings.timeFormat') }}</span>
            <select v-model="appSettings.general.timeFormat" class="app-select">
              <option value="24h">{{ t('settings.hour24') }}</option>
              <option value="12h">{{ t('settings.hour12') }}</option>
            </select>
          </label>
        </div>
      </section>

      <section class="glass-panel p-5">
        <p class="section-heading">{{ t('settings.interface') }}</p>
        <div class="grid gap-4 lg:grid-cols-2">
          <label class="block">
            <span class="mb-2 block text-sm font-semibold">{{ t('settings.theme') }}</span>
            <select v-model="appSettings.ui.theme" class="app-select">
              <option value="dark">{{ t('settings.dark') }}</option>
              <option value="light">{{ t('settings.light') }}</option>
            </select>
          </label>

          <label class="block">
            <span class="mb-2 block text-sm font-semibold">{{ t('settings.density') }}</span>
            <select v-model="appSettings.ui.density" class="app-select">
              <option value="comfortable">{{ t('settings.comfortable') }}</option>
              <option value="compact">{{ t('settings.compact') }}</option>
            </select>
          </label>

          <label class="block lg:col-span-2">
            <span class="mb-2 block text-sm font-semibold">{{ t('settings.fontScale') }} ({{ appSettings.ui.fontScale.toFixed(2) }})</span>
            <div class="border px-4 py-4" style="border-color: var(--glass-border); background: var(--glass);">
              <input v-model.number="appSettings.ui.fontScale" type="range" min="0.9" max="1.15" step="0.01" class="w-full accent-blue-600" />
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

      <section class="glass-panel p-5">
        <p class="section-heading">{{ t('settings.runtime') }}</p>
        <div class="grid gap-4 lg:grid-cols-2">
          <label class="block lg:col-span-2">
            <span class="mb-2 block text-sm font-semibold">{{ t('settings.dockerApiEndpoint') }}</span>
            <input v-model.trim="appSettings.runtime.apiBaseUrl" type="text" placeholder="http://localhost:8080" class="app-input" />
            <small class="mt-2 block text-xs" style="color: var(--text-muted);">
              {{ t('settings.dockerApiHelp', { value: apiHint }) }}
            </small>
          </label>

          <label class="block">
            <span class="mb-2 block text-sm font-semibold">{{ t('settings.defaultLogTail') }}</span>
            <input v-model.number="appSettings.runtime.defaultLogTail" type="number" min="50" max="5000" step="50" class="app-input" />
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
              <option value="ocean">{{ t('settings.themeOcean') }}</option>
              <option value="matrix">{{ t('settings.themeMatrix') }}</option>
              <option value="amber">{{ t('settings.themeAmber') }}</option>
            </select>
          </label>

          <label class="block">
            <span class="mb-2 block text-sm font-semibold">{{ t('settings.terminalFontSize') }}</span>
            <input v-model.number="appSettings.runtime.terminalFontSize" type="number" min="11" max="20" step="1" class="app-input" />
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

      <section class="glass-panel p-5">
        <div class="mb-4 flex items-center justify-between gap-4">
          <p class="section-heading mb-0">{{ t('settings.updates') }}</p>
          <span class="border px-3 py-1 text-xs font-semibold uppercase tracking-[0.18em]" :style="updateStatusTone">
            {{ statusLabel }}
          </span>
        </div>

        <div class="grid gap-4 lg:grid-cols-2">
          <label class="settings-switch-row lg:col-span-2">
            <span class="text-sm font-semibold">{{ t('settings.autoCheckUpdates') }}</span>
            <span class="settings-switch">
              <input v-model="appSettings.updates.autoCheck" type="checkbox" class="settings-switch-input" />
              <span class="settings-switch-track"></span>
            </span>
          </label>

          <label class="block">
            <span class="mb-2 block text-sm font-semibold">{{ t('settings.dockerHubNamespace') }}</span>
            <input v-model.trim="appSettings.updates.dockerHubNamespace" type="text" class="app-input" placeholder="ngthanhvu" />
          </label>

          <label class="block">
            <span class="mb-2 block text-sm font-semibold">{{ t('settings.dockerHubRepoPrefix') }}</span>
            <input v-model.trim="appSettings.updates.dockerHubRepoPrefix" type="text" class="app-input" placeholder="docker-manager" />
          </label>

          <div class="border p-4 lg:col-span-2" style="border-color: var(--glass-border); background: var(--glass);">
            <div class="grid gap-3 sm:grid-cols-2">
              <div>
                <p class="text-[11px] uppercase tracking-[0.22em]" style="color: var(--text-muted);">{{ t('settings.currentVersion') }}</p>
                <p class="mt-2 text-xl font-bold">v{{ updateState.currentVersion }}</p>
              </div>
              <div>
                <p class="text-[11px] uppercase tracking-[0.22em]" style="color: var(--text-muted);">{{ t('settings.latestVersion') }}</p>
                <p class="mt-2 text-xl font-bold">{{ updateState.latestVersion ? `v${updateState.latestVersion}` : t('common.notAvailable') }}</p>
              </div>
              <div>
                <p class="text-[11px] uppercase tracking-[0.22em]" style="color: var(--text-muted);">{{ t('settings.lastChecked') }}</p>
                <p class="mt-2 text-sm font-medium">{{ checkedAtLabel }}</p>
              </div>
              <div>
                <p class="text-[11px] uppercase tracking-[0.22em]" style="color: var(--text-muted);">{{ t('settings.latestPublished') }}</p>
                <p class="mt-2 text-sm font-medium">{{ releaseDateLabel }}</p>
              </div>
            </div>

            <p class="mt-4 text-sm leading-6" style="color: var(--text-muted);">
              {{ updateState.message || t('settings.updateHelp') }}
            </p>
          </div>

          <div class="flex flex-wrap gap-3 lg:col-span-2">
            <button class="btn btn-ghost" type="button" :disabled="updateState.status === 'checking'" @click="checkUpdates()">
              {{ updateState.status === 'checking' ? t('settings.updateChecking') : t('settings.checkUpdates') }}
            </button>
            <button
              class="btn btn-primary"
              type="button"
              :disabled="updateState.status !== 'available' || updateState.applying"
              @click="applyUpdate"
            >
              {{ t('settings.updateNow') }}
            </button>
            <button
              class="btn btn-ghost"
              type="button"
              :disabled="updateState.status === 'checking' || (!updateState.applying && !updateConsoleOutput)"
              @click="openUpdateConsole"
            >
              {{ t('settings.openUpdateConsole') }}
            </button>
            <button
              class="btn btn-ghost"
              type="button"
              :disabled="updateState.status === 'checking'"
              @click="openUpdatePage"
            >
              {{ t('settings.openUpdatePage') }}
            </button>
          </div>
        </div>
      </section>

      <section class="glass-panel p-5">
        <p class="section-heading">{{ t('settings.notifications') }}</p>
        <div class="grid gap-4 lg:grid-cols-2">
          <label class="block lg:col-span-2">
            <span class="mb-2 block text-sm font-semibold">{{ t('settings.toastDuration') }}</span>
            <input v-model.number="appSettings.notifications.toastDurationMs" type="number" min="1000" max="10000" step="100" class="app-input" />
          </label>

          <label class="settings-switch-row">
            <span class="text-sm font-semibold">{{ t('settings.showSuccessToasts') }}</span>
            <span class="settings-switch">
              <input v-model="appSettings.notifications.showSuccessToast" type="checkbox" class="settings-switch-input" />
              <span class="settings-switch-track"></span>
            </span>
          </label>

          <label class="settings-switch-row">
            <span class="text-sm font-semibold">{{ t('settings.showDetailedErrors') }}</span>
            <span class="settings-switch">
              <input v-model="appSettings.notifications.showDetailedErrors" type="checkbox" class="settings-switch-input" />
              <span class="settings-switch-track"></span>
            </span>
          </label>
        </div>
      </section>

      <section class="glass-panel p-5">
        <p class="section-heading">{{ t('settings.safety') }}</p>
        <div class="grid gap-4">
          <label class="settings-switch-row">
            <span class="text-sm font-semibold">{{ t('settings.requireDeleteTyping') }}</span>
            <span class="settings-switch">
              <input v-model="appSettings.safety.softDeleteRequireTyping" type="checkbox" class="settings-switch-input" />
              <span class="settings-switch-track"></span>
            </span>
          </label>

          <label class="block">
            <span class="mb-2 block text-sm font-semibold">{{ t('settings.protectedResources') }}</span>
            <input
              v-model="appSettings.safety.protectedResources"
              type="text"
              :placeholder="t('settings.protectedResourcesPlaceholder')"
              class="app-input"
            />
          </label>
        </div>
      </section>

      <section class="glass-panel p-5">
        <p class="section-heading">{{ t('settings.about') }}</p>
        <div class="grid gap-px border text-sm sm:grid-cols-[220px_minmax(0,1fr)]" style="border-color: var(--glass-border); background: var(--glass-border);">
          <div class="px-4 py-3 font-semibold" style="background: var(--table-header-bg);">{{ t('settings.appVersion') }}</div>
          <div class="px-4 py-3" style="background: var(--bg-card);">v{{ appSettings.about.appVersion }}</div>
          <div class="px-4 py-3 font-semibold" style="background: var(--table-header-bg);">{{ t('settings.buildDate') }}</div>
          <div class="px-4 py-3" style="background: var(--bg-card);">{{ appSettings.about.buildDate }}</div>
          <div class="px-4 py-3 font-semibold" style="background: var(--table-header-bg);">{{ t('settings.engine') }}</div>
          <div class="px-4 py-3" style="background: var(--bg-card);">{{ props.systemInfo?.ServerVersion || t('common.notAvailable') }}</div>
          <div class="px-4 py-3 font-semibold" style="background: var(--table-header-bg);">{{ t('settings.os') }}</div>
          <div class="px-4 py-3" style="background: var(--bg-card);">{{ props.systemInfo?.OperatingSystem || t('common.notAvailable') }}</div>
        </div>
      </section>
    </div>

    <div v-if="showUpdateConsole" class="update-console-backdrop" @click.self="closeUpdateConsole">
      <div class="update-console-panel">
        <div class="update-console-header">
          <div class="update-progress-copy">
            <p class="section-heading mb-0">{{ t('settings.openUpdateConsole') }}</p>
            <h3 class="update-console-title">{{ updateProgress.title }}</h3>
            <p class="update-progress-detail">{{ updateProgress.detail }}</p>
          </div>
          <button class="btn btn-ghost" type="button" :disabled="updateState.applying || waitingForReload" @click="closeUpdateConsole">
            {{ t('common.close') }}
          </button>
        </div>

        <div class="update-progress-shell">
          <div class="update-progress-head">
            <span>{{ t('settings.updateProgressLabel') }}</span>
            <strong>{{ updateProgress.percent }}%</strong>
          </div>
          <div class="update-progress-track">
            <div class="update-progress-fill" :class="{ 'is-failed': updateProgress.failed }"
              :style="{ width: `${updateProgress.percent}%` }"></div>
          </div>
          <div class="update-phase-list">
            <div v-for="step in updateProgress.steps" :key="step.key" class="update-phase-item"
              :class="[`is-${step.state}`]">
              <span class="phase-dot"></span>
              <span>{{ step.label }}</span>
            </div>
          </div>
        </div>

        <div class="update-log-shell">
          <div class="update-log-header">
            <span>{{ t('settings.updateLogs') }}</span>
          </div>
          <pre ref="updateConsoleEl" class="update-console-output" @scroll="handleUpdateConsoleScroll">{{ updateConsoleOutput }}</pre>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.settings-switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  border: 1px solid var(--glass-border);
  padding: 12px 16px;
  background: var(--glass);
}

.settings-switch {
  position: relative;
  display: inline-flex;
  align-items: center;
  width: 52px;
  height: 30px;
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
  top: 3px;
  left: 3px;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: #f8fafc;
  box-shadow: 0 4px 12px rgba(15, 23, 42, 0.28);
  transition: transform 0.18s ease;
}

.settings-switch-input:checked + .settings-switch-track {
  background: color-mix(in srgb, var(--primary) 70%, transparent);
  border-color: color-mix(in srgb, var(--primary) 72%, transparent);
}

.settings-switch-input:checked + .settings-switch-track::after {
  transform: translateX(22px);
}

.settings-switch-input:focus-visible + .settings-switch-track {
  outline: 2px solid color-mix(in srgb, var(--primary) 55%, transparent);
  outline-offset: 2px;
}

.update-console-backdrop {
  position: fixed;
  inset: 0;
  z-index: 70;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: rgba(2, 6, 23, 0.72);
  backdrop-filter: blur(10px);
}

.update-console-panel {
  width: min(980px, 100%);
  max-height: min(82vh, 860px);
  display: flex;
  flex-direction: column;
  gap: 14px;
  border: 1px solid rgba(96, 165, 250, 0.24);
  border-radius: 22px;
  padding: 22px;
  background:
    radial-gradient(circle at top right, rgba(59, 130, 246, 0.14), transparent 28%),
    linear-gradient(180deg, rgba(7, 14, 27, 0.96), rgba(3, 7, 18, 0.98));
  box-shadow: 0 24px 70px rgba(2, 6, 23, 0.45);
}

.update-console-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.update-progress-copy {
  min-width: 0;
}

.update-console-title {
  margin: 6px 0 0;
  font-size: 1.15rem;
  font-weight: 700;
  color: #e2e8f0;
}

.update-progress-detail {
  margin: 8px 0 0;
  color: #94a3b8;
  font-size: 0.92rem;
  line-height: 1.6;
}

.update-progress-shell {
  display: flex;
  flex-direction: column;
  gap: 14px;
  border: 1px solid rgba(148, 163, 184, 0.16);
  border-radius: 18px;
  padding: 18px;
  background: rgba(15, 23, 42, 0.34);
}

.update-progress-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  color: #cbd5e1;
  font-size: 0.9rem;
}

.update-progress-head strong {
  font-size: 1rem;
  color: #f8fafc;
}

.update-progress-track {
  height: 14px;
  overflow: hidden;
  border-radius: 999px;
  background: rgba(148, 163, 184, 0.16);
  border: 1px solid rgba(148, 163, 184, 0.12);
}

.update-progress-fill {
  height: 100%;
  border-radius: inherit;
  background: linear-gradient(90deg, #38bdf8, #2563eb 55%, #22c55e);
  transition: width 0.24s ease;
}

.update-progress-fill.is-failed {
  background: linear-gradient(90deg, #f97316, #ef4444);
}

.update-phase-list {
  display: grid;
  gap: 10px;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
}

.update-phase-item {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 42px;
  border: 1px solid rgba(148, 163, 184, 0.12);
  border-radius: 14px;
  padding: 0 12px;
  color: #94a3b8;
  background: rgba(15, 23, 42, 0.22);
}

.update-phase-item .phase-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: rgba(148, 163, 184, 0.35);
  flex-shrink: 0;
}

.update-phase-item.is-active {
  color: #dbeafe;
  border-color: rgba(59, 130, 246, 0.3);
}

.update-phase-item.is-active .phase-dot {
  background: #60a5fa;
  box-shadow: 0 0 0 4px rgba(59, 130, 246, 0.18);
}

.update-phase-item.is-done {
  color: #dcfce7;
  border-color: rgba(34, 197, 94, 0.25);
}

.update-phase-item.is-done .phase-dot {
  background: #22c55e;
}

.update-phase-item.is-error {
  color: #fecaca;
  border-color: rgba(239, 68, 68, 0.28);
}

.update-phase-item.is-error .phase-dot {
  background: #ef4444;
}

.update-log-shell {
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.update-log-header {
  font-size: 0.84rem;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: #94a3b8;
}

.update-console-output {
  flex: 1;
  min-height: 280px;
  margin: 0;
  overflow: auto;
  border-radius: 18px;
  border: 1px solid rgba(148, 163, 184, 0.18);
  padding: 18px;
  background:
    linear-gradient(180deg, rgba(15, 23, 42, 0.96), rgba(2, 6, 23, 0.98));
  color: #dbeafe;
  font: 13px/1.7 'JetBrains Mono', 'Fira Code', ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  white-space: pre-wrap;
  word-break: break-word;
}

@media (max-width: 768px) {
  .update-console-backdrop {
    padding: 12px;
  }

  .update-console-panel {
    max-height: calc(100vh - 24px);
    padding: 16px;
    border-radius: 18px;
  }

  .update-console-header {
    align-items: flex-start;
    flex-direction: column;
  }

  .update-console-output {
    min-height: 220px;
  }
}
</style>
