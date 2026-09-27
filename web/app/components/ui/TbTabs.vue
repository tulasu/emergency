<script setup lang="ts">
import type { TbTabItem } from '~/types/ui';

const model = defineModel<string>({ default: '' });

defineProps<{
  tabs: TbTabItem[];
}>();
</script>

<template>
  <div class="tb-tabs" role="tablist">
    <button
      v-for="tab in tabs"
      :key="tab.id"
      type="button"
      class="tb-tabs__item"
      :class="{ 'tb-tabs__item--active': model === tab.id }"
      role="tab"
      :aria-selected="model === tab.id"
      @click="model = tab.id"
    >
      <span>{{ tab.label }}</span>
      <span v-if="tab.count != null" class="tb-tabs__count">{{ tab.count }}</span>
    </button>
  </div>
</template>

<style scoped>
.tb-tabs {
  display: flex;
  gap: 8px;
  border-bottom: 1px solid var(--color-border);
}

.tb-tabs__item {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 44px;
  padding: 0 4px;
  margin-bottom: -1px;
  border: 0;
  border-bottom: 2px solid transparent;
  background: transparent;
  color: var(--color-text-muted);
  font: var(--font-body);
}

.tb-tabs__item--active {
  color: var(--color-text);
  border-bottom-color: var(--color-primary);
}

.tb-tabs__count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 22px;
  height: 22px;
  padding: 0 6px;
  border-radius: 999px;
  background: var(--color-secondary);
  color: var(--color-primary);
  font: var(--font-cap);
}
</style>
