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
  <div class="min-h-screen min-h-dvh p-4 sm:p-6">
    <div class="grid min-h-full place-items-center">
      <div class="grid w-full max-w-5xl overflow-hidden border lg:grid-cols-[1.15fr_0.85fr]" style="border-color: var(--glass-border);">
        <section class="relative hidden min-h-[620px] overflow-hidden border-r p-10 lg:block" style="border-color: var(--glass-border); background: linear-gradient(180deg, rgba(29,78,216,0.08), rgba(0,0,0,0.08));">
          <div class="absolute inset-0 opacity-50" style="background:
            linear-gradient(180deg, rgba(255,255,255,0.04) 1px, transparent 1px),
            linear-gradient(90deg, rgba(255,255,255,0.04) 1px, transparent 1px); background-size: 28px 28px;">
          </div>
          <div class="relative flex h-full flex-col justify-between">
            <div>
              <p class="mb-3 text-[11px] uppercase tracking-[0.26em]" style="color: var(--text-muted);">Docker Access Node</p>
              <h1 class="max-w-md text-5xl font-bold leading-none tracking-tight">
                Secure control without the glossy dashboard look.
              </h1>
              <p class="mt-6 max-w-lg text-base leading-7" style="color: var(--text-muted);">
                This panel is intentionally sharper and more operational: clearer hierarchy, stronger edges, and less decorative noise.
              </p>
            </div>

            <div class="grid gap-4">
              <div class="flex items-start gap-4 border p-4" style="border-color: var(--glass-border); background: var(--glass);">
                <ShieldCheck class="mt-0.5" :size="18" style="color: var(--success);" />
                <div>
                  <p class="font-semibold">Protected session flow</p>
                  <p class="mt-1 text-sm leading-6" style="color: var(--text-muted);">
                    Authentication gates container actions, shell access, and destructive operations.
                  </p>
                </div>
              </div>
              <div class="grid grid-cols-2 gap-4">
                <div class="border p-4" style="border-color: var(--glass-border); background: var(--glass);">
                  <p class="text-[11px] uppercase tracking-[0.22em]" style="color: var(--text-muted);">Surface</p>
                  <p class="mt-2 text-2xl font-bold">CLI-like</p>
                </div>
                <div class="border p-4" style="border-color: var(--glass-border); background: var(--glass);">
                  <p class="text-[11px] uppercase tracking-[0.22em]" style="color: var(--text-muted);">Theme</p>
                  <p class="mt-2 text-2xl font-bold">Rigid UI</p>
                </div>
              </div>
            </div>
          </div>
        </section>

        <section class="glass-panel min-h-[620px] p-6 sm:p-8 lg:p-10">
          <div class="mx-auto flex h-full max-w-md flex-col justify-center">
            <div class="mb-8">
              <div class="mb-5 flex items-center gap-4">
                <div class="grid h-14 w-14 place-items-center border" style="border-color: var(--primary); background: rgba(29, 78, 216, 0.12); color: var(--primary);">
                  <ShieldCheck :size="22" />
                </div>
                <div>
                  <p class="text-[11px] uppercase tracking-[0.24em]" style="color: var(--text-muted);">Secure Access</p>
                  <h2 class="text-3xl font-bold tracking-tight">{{ title }}</h2>
                </div>
              </div>
              <p class="text-sm leading-6" style="color: var(--text-muted);">{{ description }}</p>
            </div>

            <form class="space-y-4" @submit.prevent="submit">
              <label class="block">
                <span class="mb-2 block text-sm font-semibold">Username</span>
                <div class="flex items-center gap-3 border px-4 py-3" style="border-color: var(--glass-border); background: var(--input-bg);">
                  <UserRound :size="16" style="color: var(--text-muted);" />
                  <input v-model="username" class="min-w-0 flex-1 bg-transparent outline-none" type="text" autocomplete="username" placeholder="admin" :disabled="loading" />
                </div>
              </label>

              <label class="block">
                <span class="mb-2 block text-sm font-semibold">Password</span>
                <div class="flex items-center gap-3 border px-4 py-3" style="border-color: var(--glass-border); background: var(--input-bg);">
                  <LockKeyhole :size="16" style="color: var(--text-muted);" />
                  <input v-model="password" class="min-w-0 flex-1 bg-transparent outline-none" type="password" autocomplete="current-password" placeholder="••••••••" :disabled="loading" />
                </div>
              </label>

              <label v-if="setupRequired" class="block">
                <span class="mb-2 block text-sm font-semibold">Confirm password</span>
                <div class="flex items-center gap-3 border px-4 py-3" style="border-color: var(--glass-border); background: var(--input-bg);">
                  <LockKeyhole :size="16" style="color: var(--text-muted);" />
                  <input v-model="confirmPassword" class="min-w-0 flex-1 bg-transparent outline-none" type="password" autocomplete="new-password" placeholder="Repeat password" :disabled="loading" />
                </div>
              </label>

              <p v-if="localError" class="border px-4 py-3 text-sm font-medium" style="border-color: rgba(255,95,86,0.5); color: #fecaca; background: rgba(255,95,86,0.08);">
                {{ localError }}
              </p>

              <button class="btn btn-primary mt-3 w-full" type="submit" :disabled="loading">
                {{ submitLabel }}
              </button>
            </form>
          </div>
        </section>
      </div>
    </div>
  </div>
</template>
