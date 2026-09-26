<script setup lang="ts">
defineProps<{
  label: string;
}>();
</script>

<template>
  <label class="tb-field">
    <span class="tb-field__label">{{ label }}</span>
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

.tb-field__label {
  font: var(--font-body);
}
</style>
