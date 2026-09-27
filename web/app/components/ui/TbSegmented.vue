<script setup lang="ts">
export interface TbSegmentItem {
  id: string;
  label: string;
  count?: number;
}

const props = defineProps<{
  items: TbSegmentItem[];
}>();

const model = defineModel<string>({ required: true });
</script>

<template>
  <div class="tb-segmented" role="tablist">
    <button
      v-for="item in props.items"
      :key="item.id"
      type="button"
      role="tab"
      class="tb-segmented__item"
      :class="{ 'tb-segmented__item--active': model === item.id }"
      :aria-selected="model === item.id"
      @click="model = item.id"
    >
      <span class="tb-segmented__label">{{ item.label }}</span>
      <span v-if="item.count != null" class="tb-segmented__count">{{ item.count }}</span>
    </button>
  </div>
</template>

<style scoped>
.tb-segmented {
  display: inline-flex;
  flex-wrap: wrap;
  gap: 4px;
  align-items: center;
  padding: 4px;
  border-radius: var(--radius-md);
  background: var(--color-bg);
}

.tb-segmented__item {
  height: auto;
  min-height: 32px;
  padding: 8px 14px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-muted);
  font: 500 14px/1.3 var(--font-sans);
  display: inline-flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
}

.tb-segmented__item--active {
  background: var(--color-surface);
  color: var(--color-text);
  font-weight: 700;
}

.tb-segmented__count {
  color: var(--color-text-subtle);
  font-weight: 500;
}

.tb-segmented__item--active .tb-segmented__count {
  color: var(--color-text-subtle);
}
</style>
