<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue';
import { useRouter } from 'vue-router';
import { feedback } from './ui/feedback';
import { authState, clearAuthSession } from './ui/auth';
import { setUnauthorizedHandler } from './api';
import UiFeedback from './components/UiFeedback.vue';

const router = useRouter();

onMounted(() => {
  setUnauthorizedHandler(() => {
    if (!authState.user && !authState.token) return;
    clearAuthSession();
    feedback.warning('Session expired. Please sign in again.');
    if (router.currentRoute.value.name !== 'auth') {
      void router.replace({ name: 'auth' });
    }
  });
});

onUnmounted(() => {
  setUnauthorizedHandler(null);
});
</script>

<template>
  <router-view />
  <UiFeedback />
</template>
