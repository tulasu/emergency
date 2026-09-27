<script setup lang="ts">
defineProps<{
  label: string;
  hint?: string;
}>();
</script>

<template>
  <label class="tb-field">
    <span class="tb-field__label-row">
      <span class="tb-field__label">{{ label }}</span>
      <span v-if="hint" class="tb-field__hint">{{ hint }}</span>
    </span>
    <slot />
  </label>
</template>

<style scoped>
.tb-field {
  display: flex;
  flex-direction: column;
  gap: var(--space-sm);
  width: 100%;
}

.tb-field__label-row {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
}

.tb-field__label {
  font: var(--font-body);
}

.tb-field__hint {
  color: var(--color-text-subtle);
  font: 400 12px/1.3 var(--font-sans);
}
</style>
