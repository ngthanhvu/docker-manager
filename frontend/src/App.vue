<script setup lang="ts">
import { onMounted, onUnmounted, watch } from 'vue';
import { useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import { feedback } from './ui/feedback';
import { authState, clearAuthSession } from './ui/auth';
import { setUnauthorizedHandler } from './api';
import { appSettings } from './ui/settings';
import { updates } from './ui/updates';
import UiFeedback from './components/UiFeedback.vue';

const router = useRouter();
const { t } = useI18n();
let autoUpdatePromptShown = false;

const maybePromptForAppUpdate = async () => {
  if (autoUpdatePromptShown || !appSettings.updates.autoCheck || !authState.user) return;

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
  setUnauthorizedHandler(() => {
    if (!authState.user && !authState.token) return;
    clearAuthSession();
    feedback.warning(t('app.sessionExpired'));
    if (router.currentRoute.value.name !== 'auth') {
      void router.replace({ name: 'auth' });
    }
  });

  void maybePromptForAppUpdate();
});

onUnmounted(() => {
  setUnauthorizedHandler(null);
});

watch(
  () => authState.user,
  () => {
    void maybePromptForAppUpdate();
  }
);
</script>

<template>
  <router-view />
  <UiFeedback />
</template>
