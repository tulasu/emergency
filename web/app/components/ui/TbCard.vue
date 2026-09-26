<script setup lang="ts">
withDefaults(
  defineProps<{
    as?: string;
  }>(),
  { as: 'div' },
);
</script>

<template>
  <component :is="as" class="tb-card">
    <slot />
  </component>
</template>

<style scoped>
.tb-card {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-card);
}
</style>
