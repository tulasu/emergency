<script setup lang="ts">
export interface AssignmentGroupRow {
  label: string;
  count: number;
}

defineProps<{
  groups?: AssignmentGroupRow[];
  individuals?: number;
}>();

const emit = defineEmits<{
  edit: [];
}>();
</script>

<template>
  <aside class="side-panel">
    <div class="side-panel__head">
      <h2 class="side-panel__title">Назначен</h2>
      <button type="button" class="side-panel__link" @click="emit('edit')">Изменить</button>
    </div>
    <div v-if="!groups?.length && !individuals" class="side-panel__empty">не назначен</div>
    <ul v-else class="side-panel__list">
      <li v-for="group in groups || []" :key="group.label" class="side-panel__row">
        <span class="side-panel__row-label">
          <TbIcon name="users" />
          {{ group.label }}
        </span>
        <strong>{{ group.count }}</strong>
      </li>
      <li v-if="individuals" class="side-panel__row">
        <span class="side-panel__row-label">
          <TbIcon name="user" />
          Отдельные ученики
        </span>
        <strong>{{ individuals }}</strong>
      </li>
    </ul>
  </aside>
</template>

<style scoped>
.side-panel__row-label {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.side-panel__row-label :deep(.tb-icon) {
  width: 16px;
  height: 16px;
  color: var(--color-text-subtle);
}
</style>
