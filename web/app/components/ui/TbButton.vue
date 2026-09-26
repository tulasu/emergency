<script setup lang="ts">
import type { TbButtonVariant, TbButtonWidth } from '~/types/ui';

const props = withDefaults(
  defineProps<{
    variant?: TbButtonVariant;
    width?: TbButtonWidth;
    disabled?: boolean;
    busy?: boolean;
    type?: 'button' | 'submit' | 'reset';
  }>(),
  {
    variant: 'primary',
    width: 'auto',
    disabled: false,
    busy: false,
    type: 'button',
  },
);

const showSpin = ref(false);

watch(
  () => props.busy,
  (busy) => {
    if (!busy) {
      showSpin.value = false;
      return;
    }
    const id = window.setTimeout(() => {
      showSpin.value = true;
    }, 200);
    onWatcherCleanup(() => window.clearTimeout(id));
  },
);
</script>

<template>
  <button
    class="tb-btn"
    :class="[`tb-btn--${variant}`, `tb-btn--${width}`, { 'tb-btn--busy': busy }]"
    :type="type"
    :disabled="disabled || busy"
    :aria-busy="busy || undefined"
  >
    <span v-if="showSpin" class="tb-btn__spin" aria-hidden="true"></span>
    <span class="tb-btn__label" :class="{ 'tb-btn__label--hidden': showSpin }">
      <slot />
    </span>
  </button>
</template>

<style scoped>
.tb-btn {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-sm);
  height: 48px;
  padding: 0 20px;
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  font: var(--font-body);
  transition: opacity 0.15s ease;
}

.tb-btn--block {
  display: flex;
  width: 100%;
}

.tb-btn:disabled {
  opacity: 0.5;
  cursor: default;
}

.tb-btn--primary {
  background: var(--color-primary);
  color: var(--color-primary-text);
}

.tb-btn--secondary {
  background: var(--color-surface);
  color: var(--color-text);
  border-color: var(--color-border-strong);
}

.tb-btn--ghost {
  background: transparent;
  color: var(--color-text);
}

.tb-btn--danger {
  background: var(--color-danger-soft);
  color: var(--color-danger);
}

.tb-btn__label--hidden {
  visibility: hidden;
}

.tb-btn__spin {
  position: absolute;
  width: 20px;
  height: 20px;
  border: 2px solid currentColor;
  border-right-color: transparent;
  border-radius: 50%;
  animation: tb-spin 0.6s linear infinite;
}

@keyframes tb-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
