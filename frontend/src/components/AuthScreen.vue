<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { LockKeyhole, ShieldCheck, UserRound } from 'lucide-vue-next';

const props = defineProps<{
  setupRequired: boolean;
  loading?: boolean;
}>();

const emit = defineEmits<{
  login: [payload: { username: string; password: string }];
  setup: [payload: { username: string; password: string }];
}>();

const username = ref('');
const password = ref('');
const confirmPassword = ref('');
const localError = ref('');

const title = computed(() =>
  props.setupRequired ? 'Create your admin account' : 'Sign in to Dock Manager'
);

const description = computed(() =>
  props.setupRequired
    ? 'First launch detected. Create the initial account stored in SQLite to secure the panel.'
    : 'Use your account to access containers, compose stacks, metrics, and terminal sessions.'
);

const submitLabel = computed(() => {
  if (props.loading) return props.setupRequired ? 'Creating account...' : 'Signing in...';
  return props.setupRequired ? 'Create account' : 'Sign in';
});

const submit = () => {
  localError.value = '';

  if (username.value.trim().length < 3) {
    localError.value = 'Username must be at least 3 characters.';
    return;
  }

  if (password.value.trim().length < 8) {
    localError.value = 'Password must be at least 8 characters.';
    return;
  }

  if (props.setupRequired && password.value !== confirmPassword.value) {
    localError.value = 'Password confirmation does not match.';
    return;
  }

  const payload = {
    username: username.value.trim(),
    password: password.value,
  };

  if (props.setupRequired) {
    emit('setup', payload);
    return;
  }

  emit('login', payload);
};

watch(
  () => props.setupRequired,
  () => {
    password.value = '';
    confirmPassword.value = '';
    localError.value = '';
  }
);
</script>

<template>
  <div class="auth-shell">
    <div class="auth-panel glass-panel">
      <div class="auth-brand">
        <div class="brand-badge">
          <ShieldCheck :size="22" />
        </div>
        <div>
          <p class="eyebrow">Secure Access</p>
          <h1>{{ title }}</h1>
          <p class="description">{{ description }}</p>
        </div>
      </div>

      <form class="auth-form" @submit.prevent="submit">
        <label class="field">
          <span>Username</span>
          <div class="input-shell">
            <UserRound :size="16" />
            <input v-model="username" type="text" autocomplete="username" placeholder="admin" :disabled="loading" />
          </div>
        </label>

        <label class="field">
          <span>Password</span>
          <div class="input-shell">
            <LockKeyhole :size="16" />
            <input v-model="password" type="password" autocomplete="current-password" placeholder="••••••••"
              :disabled="loading" />
          </div>
        </label>

        <label v-if="setupRequired" class="field">
          <span>Confirm password</span>
          <div class="input-shell">
            <LockKeyhole :size="16" />
            <input v-model="confirmPassword" type="password" autocomplete="new-password" placeholder="Repeat password"
              :disabled="loading" />
          </div>
        </label>

        <p v-if="localError" class="inline-error">{{ localError }}</p>

        <button class="submit-btn" type="submit" :disabled="loading">
          {{ submitLabel }}
        </button>
      </form>
    </div>
  </div>
</template>

<style scoped>
.auth-shell {
  min-height: 100vh;
  min-height: 100dvh;
  display: grid;
  place-items: center;
  padding: 28px;
  background:
    radial-gradient(circle at top left, rgba(34, 197, 94, 0.16), transparent 36%),
    radial-gradient(circle at bottom right, rgba(36, 150, 237, 0.18), transparent 42%),
    linear-gradient(145deg, rgba(6, 17, 34, 0.98), rgba(15, 23, 42, 0.94));
}

.auth-panel {
  width: min(100%, 460px);
  padding: 28px;
  border-radius: 24px;
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.auth-brand {
  display: flex;
  gap: 16px;
  align-items: flex-start;
}

.brand-badge {
  width: 50px;
  height: 50px;
  border-radius: 16px;
  display: grid;
  place-items: center;
  background: linear-gradient(145deg, rgba(34, 197, 94, 0.22), rgba(36, 150, 237, 0.22));
  color: #f8fafc;
  box-shadow: 0 18px 40px rgba(2, 8, 23, 0.22);
}

.eyebrow {
  margin: 0 0 6px;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  font-size: 0.72rem;
  color: rgba(148, 163, 184, 0.9);
}

.auth-brand h1 {
  margin: 0;
  font-size: 1.8rem;
  line-height: 1.1;
}

.description {
  margin: 10px 0 0;
  color: rgba(226, 232, 240, 0.74);
  line-height: 1.55;
}

.auth-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.field span {
  color: rgba(226, 232, 240, 0.86);
  font-size: 0.92rem;
  font-weight: 600;
}

.input-shell {
  display: flex;
  align-items: center;
  gap: 10px;
  height: 50px;
  padding: 0 14px;
  border-radius: 14px;
  border: 1px solid rgba(148, 163, 184, 0.16);
  background: rgba(15, 23, 42, 0.55);
  color: rgba(226, 232, 240, 0.86);
}

.input-shell input {
  flex: 1;
  border: none;
  outline: none;
  background: transparent;
  color: inherit;
  font-size: 0.98rem;
}

.input-shell input::placeholder {
  color: rgba(148, 163, 184, 0.56);
}

.inline-error {
  margin: 0;
  color: #fda4af;
  font-size: 0.9rem;
}

.submit-btn {
  margin-top: 6px;
  height: 52px;
  border: none;
  border-radius: 16px;
  background: linear-gradient(135deg, #1fbf75, #2496ed);
  color: #f8fafc;
  font-size: 1rem;
  font-weight: 700;
  cursor: pointer;
  transition: transform 0.18s ease, box-shadow 0.18s ease, opacity 0.18s ease;
  box-shadow: 0 22px 40px rgba(15, 23, 42, 0.34);
}

.submit-btn:hover:not(:disabled) {
  transform: translateY(-1px);
}

.submit-btn:disabled {
  opacity: 0.7;
  cursor: wait;
}

@media (max-width: 640px) {
  .auth-shell {
    padding: 18px;
  }

  .auth-panel {
    padding: 22px;
    border-radius: 20px;
  }

  .auth-brand {
    flex-direction: column;
  }
}
</style>
