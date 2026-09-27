<script setup lang="ts">
withDefaults(
  defineProps<{
    title: string;
    tags?: string[];
    progress?: number | null;
    actionLabel?: string;
  }>(),
  {
    tags: () => [],
    progress: null,
    actionLabel: 'В вариант',
  },
);

const emit = defineEmits<{
  action: [];
}>();
</script>

<template>
  <article class="flash-card">
    <div class="flash-card__top">
      <h3 class="flash-card__title">{{ title }}</h3>
      <div v-if="tags.length" class="flash-card__tags">
        <span v-for="tag in tags" :key="tag" class="flash-card__tag">{{ tag }}</span>
      </div>
    </div>
    <div class="flash-card__foot">
      <span v-if="progress != null" class="flash-card__pct">{{ Math.round(progress) }}%</span>
      <span v-else class="flash-card__pct flash-card__pct--empty"></span>
      <TbButton @click="emit('action')">{{ actionLabel }}</TbButton>
    </div>
  </article>
</template>

<style scoped>
.flash-card {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  gap: 12px;
  min-height: 145px;
  padding: 16px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-card);
}

.flash-card__top {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.flash-card__title {
  margin: 0;
  font: 700 16px/1.3 var(--font-sans);
  color: var(--color-text);
}

.flash-card__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.flash-card__tag {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 10px;
  border-radius: var(--radius-sm);
  background: var(--color-secondary);
  color: var(--color-primary);
  font: 500 12px/1.3 var(--font-sans);
}

.flash-card__foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: auto;
}

.flash-card__pct {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 8px;
  border-radius: var(--radius-sm);
  background: var(--color-success-soft);
  color: var(--color-success);
  font: 500 12px/1.3 var(--font-sans);
}

.flash-card__pct--empty {
  visibility: hidden;
  min-width: 1px;
  padding: 0;
  background: transparent;
}

.flash-card :deep(.tb-btn) {
  height: 40px;
  padding: 0 20px;
}
</style>
