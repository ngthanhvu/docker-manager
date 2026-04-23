<script setup lang="ts">
import { onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { feedback } from './ui/feedback';
import { appSettings } from './ui/settings';
import { updates } from './ui/updates';
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

  try {
    const result = await updates.apply();
    feedback.info(result.message || t('settings.updateStarted'));
    void updates.waitForAppReload();
  } catch {
    feedback.error(updates.state.message || t('common.actionFailed'));
  }
};

onMounted(() => {
  void maybePromptForAppUpdate();
});
</script>

<template>
  <router-view />
  <UiFeedback />
</template>
