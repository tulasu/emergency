<script setup lang="ts">
import type { Module } from '~/types/curriculum';
import {
  formatOpened,
  formatSuccessRate,
  moduleDisplayStatus,
  moduleStatusLabel,
  moduleStatusTone,
  ruGroups,
  ruUsers,
} from '~/utils/curriculum-labels';

const props = defineProps<{
  module: Module;
  selected?: boolean;
}>();

const emit = defineEmits<{
  click: [];
}>();

const displayStatus = computed(() => moduleDisplayStatus(props.module));

const groupCount = computed(() => props.module.assigned_groups ?? 0);
const userCount = computed(
  () => props.module.assigned_users ?? props.module.assigned_count ?? 0,
);

const openedPct = computed(() => {
  if (!props.module.opened_total) {
    return 0;
  }
  return Math.round(((props.module.opened_done ?? 0) / props.module.opened_total) * 100);
});

const barComplete = computed(
  () =>
    Boolean(props.module.opened_total) &&
    props.module.opened_done === props.module.opened_total,
);
</script>

<template>
  <button
    type="button"
    class="module-table-row"
    :class="{ 'module-table-row--selected': selected }"
    @click="emit('click')"
  >
    <div class="module-table-row__main">
      <span class="module-table-row__icon" aria-hidden="true">
        <TbIcon name="folder" />
      </span>
      <div class="module-table-row__text">
        <h2 class="module-table-row__title">{{ module.title }}</h2>
        <p class="module-table-row__desc">
          {{ module.description || 'Без описания' }}
        </p>
      </div>
    </div>
    <div class="module-table-row__cell module-table-row__cell--count">
      {{ module.lesson_count ?? 0 }}
    </div>
    <div class="module-table-row__cell module-table-row__cell--assigned">
      <template v-if="groupCount || userCount">
        <span v-if="groupCount" class="assign-chip">
          <TbIcon name="users" />
          {{ ruGroups(groupCount) }}
        </span>
        <span v-if="userCount" class="assign-chip">
          <TbIcon name="user" />
          {{ ruUsers(userCount) }}
        </span>
      </template>
      <span v-else class="assign-empty">не назначен</span>
    </div>
    <div class="module-table-row__cell module-table-row__cell--progress">
      <div class="module-table-row__track">
        <span
          :class="{ 'module-table-row__fill--done': barComplete }"
          :style="{ width: `${openedPct}%` }"
        ></span>
      </div>
      <span class="module-table-row__opened">
        {{ formatOpened(module.opened_done, module.opened_total) }}
      </span>
    </div>
    <div
      class="module-table-row__cell module-table-row__cell--score"
      :class="{ 'is-success': module.success_rate != null && module.success_rate >= 70 }"
    >
      {{ formatSuccessRate(module.success_rate) }}
    </div>
    <div class="module-table-row__cell module-table-row__cell--status">
      <TbBadge :tone="moduleStatusTone(displayStatus)">
        {{ moduleStatusLabel(displayStatus) }}
      </TbBadge>
      <TbIcon name="chevron-right" />
    </div>
  </button>
</template>

<style scoped>
.module-table-row {
  display: grid;
  grid-template-columns: minmax(240px, 1fr) 90px 300px 150px 110px 138px;
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

.module-table-row:hover {
  background: var(--color-primary-soft);
}

.module-table-row--selected {
  background: var(--color-secondary);
  border-color: var(--color-primary);
}

.module-table-row__main {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  min-width: 0;
}

.module-table-row__icon {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  background: var(--color-primary);
  color: var(--color-primary-text);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.module-table-row__text {
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
}

.module-table-row__title {
  margin: 0;
  font: var(--font-block);
}

.module-table-row__desc {
  margin: 0;
  color: var(--color-text-muted);
  font: 400 12px/1.3 var(--font-sans);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.module-table-row__cell--count {
  text-align: center;
  color: var(--color-text);
  font: var(--font-body);
}

.module-table-row__cell--assigned {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.assign-chip {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 9px;
  border-radius: var(--radius-sm);
  background: var(--color-bg);
  color: var(--color-text-muted);
  font: var(--font-cap);
}

.assign-chip :deep(.tb-icon) {
  width: 12px;
  height: 12px;
}

.assign-empty {
  color: var(--color-text-subtle);
  font: var(--font-mute);
}

.module-table-row__cell--progress {
  display: flex;
  align-items: center;
  gap: 8px;
}

.module-table-row__track {
  flex: 1;
  height: 6px;
  border-radius: 3px;
  background: var(--color-bg);
  overflow: hidden;
}

.module-table-row__track > span {
  display: block;
  height: 100%;
  background: var(--color-primary);
  border-radius: 3px;
}

.module-table-row__fill--done {
  background: var(--color-success) !important;
}

.module-table-row__opened {
  width: 40px;
  color: var(--color-text);
  font: 500 13px/1.3 var(--font-sans);
}

.module-table-row__cell--score {
  text-align: center;
  color: var(--color-text-subtle);
  font: var(--font-body);
}

.module-table-row__cell--score.is-success {
  color: var(--color-success);
}

.module-table-row__cell--status {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  color: var(--color-text-subtle);
}

@media (max-width: 1360px) {
  .module-table-row {
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 12px;
    align-items: center;
  }

  .module-table-row__cell--count,
  .module-table-row__cell--assigned,
  .module-table-row__cell--progress,
  .module-table-row__cell--score {
    display: none;
  }

  .module-table-row__cell--status {
    justify-content: flex-end;
    align-self: center;
  }
}

@media (max-width: 640px) {
  .module-table-row {
    grid-template-columns: 1fr;
    gap: 10px;
  }

  .module-table-row__cell--status {
    justify-content: flex-start;
  }

  .module-table-row__main {
    min-width: 0;
  }

  .module-table-row__title {
    overflow-wrap: anywhere;
  }
}
</style>
