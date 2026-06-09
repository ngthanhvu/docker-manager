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
    document.documentElement.style.overflow = open ? 'hidden' : '';
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
  document.documentElement.style.overflow = '';
});
</script>

<template>
  <Teleport to="body">
    <div v-if="overlayState.visible" class="update-console-backdrop" @click.self="closeUpdateConsole"
      @wheel.self.prevent @touchmove.self.prevent>
      <div class="update-console-panel" @wheel.stop @touchmove.stop>
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
  overflow: hidden;
  padding: 16px;
  background: var(--overlay-bg);
  backdrop-filter: blur(14px);
  overscroll-behavior: contain;
  touch-action: none;
}

.update-console-panel {
  width: min(1320px, 100%);
  height: min(860px, calc(100dvh - 32px));
  min-height: 0;
  margin: auto;
  display: flex;
  flex-direction: column;
  gap: 12px;
  border: 1px solid var(--glass-border);
  border-radius: 10px;
  padding: 16px;
  background: var(--bg-card);
  box-shadow: var(--shadow-panel);
  touch-action: auto;
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
  font-size: 1.2rem;
  font-weight: 650;
  color: var(--text-main);
  letter-spacing: 0;
}

.update-progress-detail {
  margin: 6px 0 0;
  color: var(--text-muted);
  font-size: 0.86rem;
  line-height: 1.5;
}

.update-progress-shell {
  display: flex;
  flex-direction: column;
  gap: 12px;
  flex-shrink: 0;
  border: 1px solid var(--glass-border);
  border-radius: 8px;
  padding: 14px;
  background: var(--glass);
}

.update-progress-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  color: var(--text-muted);
  font-size: 0.82rem;
}

.update-progress-head strong {
  font-size: 0.95rem;
  color: var(--text-main);
}

.update-progress-track {
  height: 8px;
  overflow: hidden;
  border-radius: 999px;
  background: color-mix(in srgb, var(--primary) 10%, var(--glass-border));
  border: 1px solid color-mix(in srgb, var(--primary) 18%, var(--glass-border));
}

.update-progress-fill {
  height: 100%;
  border-radius: inherit;
  background: var(--primary);
  transition: width 0.24s ease;
}

.update-progress-fill.is-failed {
  background: var(--danger);
}

.update-phase-list {
  display: grid;
  gap: 8px;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
}

.update-phase-item {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 34px;
  border: 1px solid var(--glass-border);
  border-radius: 8px;
  padding: 0 10px;
  color: var(--text-muted);
  background: var(--bg-card);
  font-size: 0.8rem;
}

.update-phase-item .phase-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: color-mix(in srgb, var(--text-muted) 36%, transparent);
  flex-shrink: 0;
}

.update-phase-item.is-active {
  color: var(--primary);
  border-color: color-mix(in srgb, var(--primary) 34%, var(--glass-border));
  background: color-mix(in srgb, var(--primary) 8%, var(--glass));
}

.update-phase-item.is-active .phase-dot {
  background: var(--primary);
  box-shadow: 0 0 0 4px color-mix(in srgb, var(--primary) 16%, transparent);
}

.update-phase-item.is-done {
  color: var(--success);
  border-color: color-mix(in srgb, var(--success) 28%, var(--glass-border));
}

.update-phase-item.is-done .phase-dot {
  background: var(--success);
}

.update-phase-item.is-error {
  color: var(--danger);
  border-color: color-mix(in srgb, var(--danger) 34%, var(--glass-border));
}

.update-phase-item.is-error .phase-dot {
  background: var(--danger);
}

.update-log-shell {
  min-height: 0;
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 8px;
}

.update-log-header {
  font-size: 0.72rem;
  font-weight: 650;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--text-muted);
}

.update-console-output {
  flex: 1;
  min-height: 0;
  margin: 0;
  overflow: auto;
  border-radius: 8px;
  border: 1px solid var(--glass-border);
  padding: 14px;
  background: var(--code-bg);
  color: var(--code-text);
  font: 12.5px/1.6 var(--font-mono);
  white-space: pre-wrap;
  word-break: break-word;
  overscroll-behavior: contain;
}

@media (max-width: 768px) {
  .update-console-backdrop {
    padding: 8px;
  }

  .update-console-panel {
    height: calc(100dvh - 16px);
    padding: 12px;
  }

  .update-console-header {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
