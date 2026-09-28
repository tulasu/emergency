<script setup lang="ts">
import type { OpenedForTone } from '~/utils/curriculum-labels';

defineProps<{
  title: string;
  variantsLabel?: string;
  openedLabel?: string;
  openedTone?: OpenedForTone;
  passedLabel?: string;
  averageLabel?: string;
  hint?: string;
  selected?: boolean;
}>();

const emit = defineEmits<{
  click: [];
}>();
</script>

<template>
  <button
    type="button"
    class="lesson-table-row"
    :class="{ 'lesson-table-row--selected': selected }"
    @click="emit('click')"
  >
    <span class="lesson-table-row__drag" aria-hidden="true">⋮⋮</span>
    <div class="lesson-table-row__main">
      <strong>{{ title }}</strong>
      <p v-if="hint" class="lesson-table-row__hint">{{ hint }}</p>
    </div>
    <div class="lesson-table-row__cell">{{ variantsLabel || '—' }}</div>
    <div class="lesson-table-row__cell">
      <span
        class="lesson-table-row__pill"
        :class="`lesson-table-row__pill--${openedTone || 'neutral'}`"
      >
        {{ openedLabel || 'не открыто' }}
      </span>
    </div>
    <div class="lesson-table-row__cell">{{ passedLabel || '—' }}</div>
    <div
      class="lesson-table-row__cell lesson-table-row__cell--avg"
      :class="{ 'is-success': averageLabel && averageLabel !== '—' }"
    >
      <span>{{ averageLabel || '—' }}</span>
      <span
        v-if="averageLabel && averageLabel !== '—'"
        class="lesson-table-row__bar"
        :style="{
          '--pct': averageLabel.replace('%', '') + '%',
        }"
      />
    </div>
    <TbIcon name="chevron-right" />
  </button>
</template>

<style scoped>
.lesson-table-row {
  display: grid;
  grid-template-columns: 16px minmax(200px, 344px) 200px 250px 120px 80px 18px;
  gap: 16px;
  align-items: center;
  width: 100%;
  padding: 14px 16px;
  border: 1.5px solid transparent;
  border-radius: var(--radius-sm);
  background: transparent;
  text-align: left;
  cursor: pointer;
  color: inherit;
  font: inherit;
  box-sizing: border-box;
}

.lesson-table-row:hover {
  background: var(--color-primary-soft);
}

.lesson-table-row--selected {
  background: var(--color-secondary);
  border-color: var(--color-primary);
}

.lesson-table-row__drag {
  color: var(--color-text-subtle);
  font-size: 12px;
  letter-spacing: -2px;
  line-height: 1;
}

.lesson-table-row__main strong {
  font: var(--font-block);
  overflow-wrap: anywhere;
}

.lesson-table-row__hint {
  margin: 3px 0 0;
  color: var(--color-warning);
  font: var(--font-cap);
}

.lesson-table-row__cell {
  color: var(--color-text-muted);
  font: 500 14px/1.3 var(--font-sans);
}

.lesson-table-row__pill {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 10px;
  border-radius: 999px;
  font: 500 12px/1.3 var(--font-sans);
  white-space: nowrap;
}

.lesson-table-row__pill--info {
  background: var(--color-secondary);
  color: var(--color-primary);
}

.lesson-table-row__pill--warning {
  background: var(--color-warning-soft);
  color: var(--color-warning);
}

.lesson-table-row__pill--neutral {
  background: #14141414;
  color: var(--color-text-muted);
}

.lesson-table-row__cell--avg {
  display: flex;
  flex-direction: column;
  gap: 4px;
  color: var(--color-text-subtle);
}

.lesson-table-row__cell--avg.is-success {
  color: var(--color-success);
}

.lesson-table-row__bar {
  display: block;
  width: 100%;
  height: 4px;
  border-radius: 999px;
  background: #14141414;
  overflow: hidden;
}

.lesson-table-row__bar::after {
  content: '';
  display: block;
  width: var(--pct, 0%);
  height: 100%;
  border-radius: inherit;
  background: currentColor;
}

@media (max-width: 1360px) {
  .lesson-table-row {
    grid-template-columns: 16px 1fr 24px;
  }

  .lesson-table-row__cell {
    display: none;
  }
}
</style>
