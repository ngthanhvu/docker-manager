<script setup lang="ts">
import { ref } from 'vue';
import { useRouter } from 'vue-router';
import { AxiosError } from 'axios';
import AuthScreen from '../components/AuthScreen.vue';
import { feedback } from '../ui/feedback';
import { authState, signIn, setupAccount } from '../ui/auth';

const router = useRouter();
const submitting = ref(false);

const extractErrorMessage = (error: unknown, fallback: string) => {
  if (error instanceof AxiosError) {
    const data = error.response?.data;
    if (typeof data === 'string' && data.trim()) return data;
  }
  if (error instanceof Error && error.message) return error.message;
  return fallback;
};

const submitLogin = async (payload: { username: string; password: string }) => {
  submitting.value = true;
  try {
    const data = await signIn(payload);
    feedback.success(`Welcome back, ${data?.user?.username || payload.username}.`);
    await router.replace({ name: 'app', params: { tab: 'dashboard' } });
  } catch (error) {
    feedback.error(extractErrorMessage(error, 'Sign in failed.'));
  } finally {
    submitting.value = false;
  }
};

const submitSetup = async (payload: { username: string; password: string }) => {
  submitting.value = true;
  try {
    const data = await setupAccount(payload);
    feedback.success(`Admin account ${data?.user?.username || payload.username} created.`);
    await router.replace({ name: 'app', params: { tab: 'dashboard' } });
  } catch (error) {
    feedback.error(extractErrorMessage(error, 'Initial setup failed.'));
  } finally {
    submitting.value = false;
  }
};
</script>

<template>
  <AuthScreen
    :setup-required="authState.setupRequired"
    :loading="submitting"
    @login="submitLogin"
    @setup="submitSetup"
  />
</template>
