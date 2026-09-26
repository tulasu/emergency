<script setup lang="ts">
import type { UserRow } from '~/utils/users-form';

defineProps<{
  users: UserRow[];
}>();

const emit = defineEmits<{
  add: [];
  remove: [index: number];
}>();
</script>

<template>
  <div class="table">
    <div class="table__scroll">
      <div class="table__head">
        <span>№</span>
        <span>ФИО</span>
        <span>Логин</span>
        <span></span>
      </div>
      <div v-for="(row, i) in users" :key="i" class="table__row">
        <span>{{ i + 1 }}</span>
        <TbInput v-model="row.full_name" placeholder="ФИО" />
        <TbInput v-model="row.login" placeholder="Логин" />
        <button type="button" class="table__del" aria-label="Удалить строку" @click="emit('remove', i)">
          <TbIcon name="x" />
        </button>
      </div>
    </div>
    <button class="table__add" type="button" @click="emit('add')">
      <TbIcon name="plus" />
      Добавить строку
    </button>
  </div>
</template>

<style scoped>
.table {
  overflow: hidden;
}

.table__scroll {
  overflow-x: auto;
}

.table__head,
.table__row {
  display: grid;
  grid-template-columns: 48px minmax(160px, 1fr) minmax(140px, 280px) 40px;
  gap: 16px;
  align-items: center;
  padding: 10px 24px;
  min-width: 520px;
}

.table__head {
  padding: 12px 24px;
  background: var(--color-bg);
  color: var(--color-text-muted);
  font: var(--font-body);
}

.table__row {
  background: var(--color-surface);
  border-bottom: 1px solid var(--color-border);
}

.table__del {
  width: 40px;
  height: 40px;
  border: 0;
  background: transparent;
  color: var(--color-text-muted);
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.table__add {
  width: 100%;
  border: 0;
  background: var(--color-bg);
  color: var(--color-primary);
  padding: 14px 24px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  font: var(--font-body);
}

@media (max-width: 640px) {
  .table__head,
  .table__row {
    grid-template-columns: 32px 1fr 1fr 36px;
    gap: 8px;
    padding: 10px 12px;
    min-width: 0;
  }

  .table__add {
    padding: 14px 12px;
  }
}
</style>
