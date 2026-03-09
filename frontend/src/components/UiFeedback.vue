<script setup lang="ts">
import { feedback } from '../ui/feedback';
</script>

<template>
    <div v-if="feedback.state.confirm.open" class="confirm-overlay" @click.self="feedback.closeConfirm(false)">
        <div class="confirm-card glass-panel">
            <h3>{{ feedback.state.confirm.title }}</h3>
            <p>{{ feedback.state.confirm.message }}</p>
            <div v-if="feedback.state.confirm.requireText" class="confirm-input-wrap">
                <label>
                    Type <code>{{ feedback.state.confirm.requireText }}</code> to continue
                </label>
                <input
                    v-model="feedback.state.confirm.inputText"
                    type="text"
                    :placeholder="feedback.state.confirm.requireText"
                />
            </div>
            <div class="confirm-actions">
                <button class="btn btn-ghost" @click="feedback.closeConfirm(false)">
                    {{ feedback.state.confirm.cancelText }}
                </button>
                <button
                    class="btn"
                    :class="feedback.state.confirm.danger ? 'btn-ghost text-danger' : 'btn-primary'"
                    :disabled="feedback.state.confirm.requireText !== '' && feedback.state.confirm.inputText !== feedback.state.confirm.requireText"
                    @click="feedback.closeConfirm(true)"
                >
                    {{ feedback.state.confirm.confirmText }}
                </button>
            </div>
        </div>
    </div>

    <div class="toast-stack">
        <div
            v-for="toast in feedback.state.toasts"
            :key="toast.id"
            class="toast-item glass-panel"
            :class="`toast-${toast.type}`"
            @click="feedback.dismissToast(toast.id)"
        >
            {{ toast.message }}
        </div>
    </div>
</template>

<style scoped>
.confirm-overlay {
    position: fixed;
    inset: 0;
    z-index: 80;
    background: var(--overlay-bg);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 24px;
}

.confirm-card {
    width: min(460px, 100%);
    padding: 20px;
    border-radius: 14px;
}

.confirm-card h3 {
    margin: 0 0 8px;
    font-size: 1.1rem;
}

.confirm-card p {
    margin: 0;
    color: var(--text-muted);
    line-height: 1.45;
}

.confirm-actions {
    margin-top: 16px;
    display: flex;
    justify-content: flex-end;
    gap: 8px;
}

.confirm-input-wrap {
    margin-top: 12px;
    display: flex;
    flex-direction: column;
    gap: 8px;
}

.confirm-input-wrap label {
    color: var(--text-muted);
    font-size: 0.83rem;
}

.confirm-input-wrap input {
    border: 1px solid var(--glass-border);
    background: var(--input-bg);
    color: var(--text-main);
    padding: 8px 10px;
    border-radius: 10px;
    outline: none;
}

.toast-stack {
    position: fixed;
    top: 18px;
    right: 18px;
    z-index: 85;
    width: min(360px, calc(100vw - 36px));
    display: flex;
    flex-direction: column;
    gap: 10px;
}

.toast-item {
    padding: 11px 14px;
    border-radius: 12px;
    font-size: 0.88rem;
    cursor: pointer;
    border-width: 1px;
    border-style: solid;
}

.toast-success { border-color: rgba(16, 185, 129, 0.45); color: #bbf7d0; }
.toast-error { border-color: rgba(239, 68, 68, 0.45); color: #fecaca; }
.toast-info { border-color: rgba(56, 189, 248, 0.45); color: #bae6fd; }
.toast-warning { border-color: rgba(245, 158, 11, 0.45); color: #fef08a; }
</style>
