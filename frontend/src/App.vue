<script setup lang="ts">
import { onMounted } from 'vue';
import { useI18n } from './i18n';
import { feedback } from './ui/feedback';
import { appSettings } from './ui/settings';
import { updates } from './ui/updates';
import { updateOverlay } from './ui/updateOverlay';
import AppUpdateOverlay from './components/AppUpdateOverlay.vue';
import UiFeedback from './components/UiFeedback.vue';

const { t } = useI18n();
let autoUpdatePromptShown = false;

const maybePromptForAppUpdate = async () => {
  if (autoUpdatePromptShown || !appSettings.updates.autoCheck) return;

  try {
    await updates.refresh({ silent: true });
  } catch {
    return;
  }

  if (updates.state.status !== 'available' || !updates.state.latestVersion) return;
  autoUpdatePromptShown = true;

  const accepted = await feedback.confirmAction({
    title: t('common.pleaseConfirm'),
    message: t('settings.updatePrompt', { version: updates.state.latestVersion }),
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
    const message = error instanceof Error ? error.message : (updates.state.message || t('common.actionFailed'));
    updateOverlay.markFailed(message);
    feedback.error(message);
  }
};

onMounted(() => {
  void maybePromptForAppUpdate();
});
</script>

<template>
  <router-view />
  <AppUpdateOverlay />
  <UiFeedback />
</template>
