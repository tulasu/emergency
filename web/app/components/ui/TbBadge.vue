<script setup lang="ts">
import type { TbBadgeTone } from '~/types/ui';

withDefaults(
  defineProps<{
    tone?: TbBadgeTone;
  }>(),
  {
    tone: 'neutral',
  },
);
</script>

<template>
  <span class="tb-badge" :class="`tb-badge--${tone}`">
    <slot />
  </span>
</template>

<style scoped>
.tb-badge {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 4px 8px;
  border-radius: var(--radius-sm);
  font: 500 12px/1.3 var(--font-sans);
  white-space: nowrap;
  box-sizing: border-box;
}

.tb-badge--success {
  background: var(--color-success-soft);
  color: var(--color-success);
}

.tb-badge--warning {
  background: var(--color-warning-soft);
  color: var(--color-warning);
}

.tb-badge--info {
  background: var(--color-secondary);
  color: var(--color-primary);
}

.tb-badge--neutral {
  background: #14141414;
  color: var(--color-text-muted);
}

.tb-badge--danger {
  background: var(--color-danger-soft);
  color: var(--color-danger);
}
</style>
