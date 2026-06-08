<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue';
import { useI18n } from '../i18n';
import { appSettings } from '../ui/settings';
import { updates } from '../ui/updates';
import { updateOverlay } from '../ui/updateOverlay';

const { t } = useI18n();

const updateState = updates.state;
const overlayState = updateOverlay.state;
const updateConsoleEl = ref<HTMLElement | null>(null);

const updateProgress = computed(() => {
  const output = overlayState.output.toLowerCase();
  const has = (value: string) => output.includes(value);

  const preparing = updateState.applying || has('[update] preparing version');
  const located = has('locating current docker manager container') || has('compose working directory') || has('resolved docker tag');
  const helperReady = has('pulling helper image') || has('helper image ready') || has('helper container created');
  const deploying = has('helper container started') || has('docker compose pull/up completed') || has('update commands finished');
  const succeeded = !updateState.applying && !!updateState.latestVersion && has('[done] update commands finished');
  const failed = overlayState.progressFailed || output.includes('[error]');

  const steps = [
    {
      key: 'prepare',
      label: t('settings.updatePhasePrepare'),
      state: failed ? (preparing ? 'error' : 'pending') : (located || helperReady || deploying || overlayState.waitingForReload || succeeded ? 'done' : preparing ? 'active' : 'pending'),
    },
    {
      key: 'helper',
      label: t('settings.updatePhaseHelper'),
      state: failed ? (helperReady ? 'error' : 'pending') : (deploying || overlayState.waitingForReload || succeeded ? 'done' : helperReady ? 'active' : 'pending'),
    },
    {
      key: 'deploy',
      label: t('settings.updatePhaseDeploy'),
      state: failed ? (deploying ? 'error' : 'pending') : (overlayState.waitingForReload || succeeded ? 'done' : deploying ? 'active' : 'pending'),
    },
    {
      key: 'reload',
      label: t('settings.updatePhaseReload'),
      state: failed ? 'pending' : (succeeded ? 'done' : overlayState.waitingForReload ? 'active' : 'pending'),
    },
  ] as const;

  let percent = 6;
  if (preparing) percent = 18;
  if (located) percent = 36;
  if (helperReady) percent = 58;
  if (deploying) percent = 82;
  if (overlayState.waitingForReload) percent = 94;
  if (succeeded) percent = 100;
  if (failed) percent = Math.max(percent, 12);

  let title = t('settings.updateConsoleTitle');
  let detail = updateState.message || t('settings.updateConsoleHelp');

  if (failed) {
    title = t('settings.updateFailed');
  } else if (succeeded) {
    title = t('settings.updateCompleted');
    detail = t('settings.updateReloading');
  } else if (overlayState.waitingForReload) {
    title = t('settings.updateReloading');
    detail = t('settings.updateReconnectHelp');
  }

  return { percent, title, detail, failed, steps };
});

const canClose = computed(() => !updateState.applying && !overlayState.waitingForReload);

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

const handleUpdateConsoleScroll = () => {
  overlayState.follow = isUpdateConsoleNearBottom();
};

const openMessages = () => ({
  socketError: t('settings.updateConsoleSocketError'),
  socketClosed: t('settings.updateConsoleSocketClosed'),
});

const closeUpdateConsole = () => {
  if (!canClose.value) return;
  updateOverlay.close();
};

watch(
  () => overlayState.visible,
  (open) => {
    document.body.style.overflow = open ? 'hidden' : '';
  }
);

watch(
  () => overlayState.output,
  () => {
    if (overlayState.follow && isUpdateConsoleNearBottom()) {
      void scrollUpdateConsoleToBottom();
    }
  }
);

onMounted(() => {
  void updates.syncStatus()
    .then((status) => {
      if (status.inProgress) {
        updateOverlay.open(openMessages());
        overlayState.waitingForReload = true;
        updateOverlay.waitForReload({ reloadTimeout: t('settings.updateReloadTimeout') });
      } else if (appSettings.updates.autoCheck && !updateState.checkedAt && updateState.status === 'idle') {
        void updates.refresh({ silent: true }).catch(() => undefined);
      }
    })
    .catch(() => undefined);
});

onUnmounted(() => {
  updateOverlay.disconnect();
  document.body.style.overflow = '';
});
</script>

<template>
  <Teleport to="body">
    <div v-if="overlayState.visible" class="update-console-backdrop" @click.self="closeUpdateConsole">
      <div class="update-console-panel">
        <div class="update-console-header">
          <div class="update-progress-copy">
            <p class="section-heading mb-0">{{ t('settings.openUpdateConsole') }}</p>
            <h3 class="update-console-title">{{ updateProgress.title }}</h3>
            <p class="update-progress-detail">{{ updateProgress.detail }}</p>
          </div>
          <button class="btn btn-ghost" type="button" :disabled="!canClose" @click="closeUpdateConsole">
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
          <pre ref="updateConsoleEl" class="update-console-output"
            @scroll="handleUpdateConsoleScroll">{{ overlayState.output }}</pre>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.update-console-backdrop {
  position: fixed;
  inset: 0;
  z-index: 9999;
  display: flex;
  padding: 0;
  background: rgba(2, 6, 23, 0.86);
  backdrop-filter: blur(10px);
}

.update-console-panel {
  width: 100%;
  height: 100vh;
  height: 100dvh;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
  border: 0;
  border-radius: 0;
  padding: 28px;
  background: linear-gradient(180deg, rgba(7, 14, 27, 0.98), rgba(3, 7, 18, 1));
  box-shadow: none;
}

.update-console-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-shrink: 0;
}

.update-progress-copy {
  min-width: 0;
}

.update-console-title {
  margin: 6px 0 0;
  font-size: 1.35rem;
  font-weight: 700;
  color: #e2e8f0;
}

.update-progress-detail {
  margin: 8px 0 0;
  color: #94a3b8;
  font-size: 0.95rem;
  line-height: 1.6;
}

.update-progress-shell {
  display: flex;
  flex-direction: column;
  gap: 14px;
  flex-shrink: 0;
  border: 1px solid rgba(148, 163, 184, 0.16);
  border-radius: 8px;
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
  border-radius: 8px;
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
  flex: 1;
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
  min-height: 0;
  margin: 0;
  overflow: auto;
  border-radius: 8px;
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
  .update-console-panel {
    padding: 16px;
  }

  .update-console-header {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
