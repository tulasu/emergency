<script setup lang="ts">
withDefaults(
  defineProps<{
    title: string;
    tags?: string[];
    meta?: string;
    progress?: number | null;
    actionLabel?: string;
  }>(),
  {
    tags: () => [],
    progress: null,
    actionLabel: 'Назначить',
  },
);

const emit = defineEmits<{
  action: [];
}>();
</script>

<template>
  <article class="lesson-card">
    <div class="lesson-card__body">
      <h3 class="lesson-card__title">{{ title }}</h3>
      <div v-if="tags.length" class="lesson-card__tags">
        <span v-for="tag in tags" :key="tag" class="lesson-card__tag">{{ tag }}</span>
      </div>
      <div class="lesson-card__meta-row">
        <span v-if="meta" class="lesson-card__meta">{{ meta }}</span>
        <span v-else class="lesson-card__meta"></span>
        <span v-if="progress != null" class="lesson-card__pct">{{ Math.round(progress) }}%</span>
      </div>
    </div>
    <TbButton width="block" @click="emit('action')">
      {{ actionLabel }}
    </TbButton>
  </article>
</template>

<style scoped>
.lesson-card {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  gap: 12px;
  min-height: 168px;
  padding: 16px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-card);
}

.lesson-card__body {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.lesson-card__title {
  margin: 0;
  font: 700 16px/1.3 var(--font-sans);
  color: var(--color-text);
}

.lesson-card__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.lesson-card__tag {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 10px;
  border-radius: var(--radius-sm);
  background: var(--color-secondary);
  color: var(--color-primary);
  font: 500 12px/1.3 var(--font-sans);
}

.lesson-card__meta-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.lesson-card__meta {
  color: var(--color-text-muted);
  font: 500 12px/1.3 var(--font-sans);
}

.lesson-card__pct {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 8px;
  border-radius: var(--radius-sm);
  background: var(--color-success-soft);
  color: var(--color-success);
  font: 500 12px/1.3 var(--font-sans);
  flex-shrink: 0;
}

.lesson-card :deep(.tb-btn) {
  height: 44px;
  font: 500 16px/1.3 var(--font-sans);
}
</style>
