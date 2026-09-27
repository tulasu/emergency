<script setup lang="ts">
export interface ReadinessItem {
  label: string;
  done: boolean;
}

defineProps<{
  items: ReadinessItem[];
  nextLabel?: string;
  nextDisabled?: boolean;
  nextBusy?: boolean;
  note?: string;
}>();

const emit = defineEmits<{
  next: [];
}>();
</script>

<template>
  <aside class="side-panel side-panel--ready readiness-panel">
    <h2 class="side-panel__title">Готовность модуля</h2>
    <ul class="readiness-panel__list">
      <li
        v-for="item in items"
        :key="item.label"
        class="readiness-panel__item"
        :class="{ 'readiness-panel__item--done': item.done }"
      >
        <TbIcon :name="item.done ? 'check' : 'info'" />
        <span>{{ item.label }}</span>
      </li>
    </ul>
    <p class="side-panel__note">
      <TbIcon name="info" />
      <span>
        {{
          note ||
          'Черновик можно сохранить в любой момент. Ученики увидят модуль после назначения и открытия занятий.'
        }}
      </span>
    </p>
    <div class="readiness-panel__spacer" aria-hidden="true"></div>
    <TbButton
      v-if="nextLabel"
      width="block"
      :disabled="nextDisabled"
      :busy="nextBusy"
      @click="emit('next')"
    >
      {{ nextLabel }}
    </TbButton>
  </aside>
</template>

<style scoped>
.readiness-panel {
  align-self: stretch;
}

.readiness-panel__list {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.readiness-panel__item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  color: var(--color-warning);
  font: 400 14px/1.35 var(--font-sans);
}

.readiness-panel__item--done {
  color: var(--color-success);
}

.readiness-panel__spacer {
  flex: 1;
  min-height: 24px;
}
</style>
