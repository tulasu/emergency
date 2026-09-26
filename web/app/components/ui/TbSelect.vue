<script setup lang="ts">
import type { TbSelectOption } from '~/types/ui';

const model = defineModel<string>({ default: '' });

const props = withDefaults(
  defineProps<{
    options?: TbSelectOption[];
    placeholder?: string;
    disabled?: boolean;
  }>(),
  {
    options: () => [],
    placeholder: 'Выберите',
    disabled: false,
  },
);

const open = ref(false);
const host = ref<HTMLElement | null>(null);

const selectedLabel = computed(
  () => props.options.find((opt) => opt.value === model.value)?.label ?? '',
);

function toggle(event: Event): void {
  event.stopPropagation();
  if (props.disabled) {
    return;
  }
  open.value = !open.value;
}

function pick(opt: TbSelectOption, event: Event): void {
  event.stopPropagation();
  model.value = opt.value;
  open.value = false;
}

function onDocumentClick(event: Event): void {
  if (!host.value?.contains(event.target as Node)) {
    open.value = false;
  }
}

function onKeydown(event: KeyboardEvent): void {
  if (event.key === 'Escape') {
    open.value = false;
  }
}

onMounted(() => {
  document.addEventListener('click', onDocumentClick);
  document.addEventListener('keydown', onKeydown);
});

onUnmounted(() => {
  document.removeEventListener('click', onDocumentClick);
  document.removeEventListener('keydown', onKeydown);
});
</script>

<template>
  <div ref="host" class="tb-select">
    <button
      type="button"
      class="tb-select__btn"
      :class="{ 'tb-select__btn--open': open }"
      :disabled="disabled"
      @click="toggle"
    >
      <span :class="{ 'tb-select__placeholder': !selectedLabel }">{{
        selectedLabel || placeholder
      }}</span>
      <TbIcon name="chevron-down" />
    </button>
    <ul v-if="open" class="tb-select__list" role="listbox">
      <li v-for="opt in options" :key="opt.value">
        <button type="button" class="tb-select__opt" @click="pick(opt, $event)">
          <span>{{ opt.label }}</span>
          <TbIcon v-if="opt.value === model" name="check" />
        </button>
      </li>
    </ul>
  </div>
</template>

<style scoped>
.tb-select {
  position: relative;
  display: block;
  width: 100%;
}

.tb-select__btn {
  width: 100%;
  height: 48px;
  padding: 0 16px;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  color: var(--color-text);
  font: var(--font-body);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  text-align: left;
}

.tb-select__btn--open :deep(.tb-icon) {
  transform: rotate(180deg);
}

.tb-select__placeholder {
  color: var(--color-text-muted);
}

.tb-select__btn:disabled {
  color: var(--color-text-subtle);
  cursor: default;
}

.tb-select__list {
  position: absolute;
  z-index: 8;
  left: 0;
  right: 0;
  top: calc(100% + 4px);
  margin: 0;
  padding: 8px;
  list-style: none;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-card);
  max-height: 240px;
  overflow: auto;
}

.tb-select__opt {
  width: 100%;
  min-height: 40px;
  padding: 8px 12px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  font: var(--font-body);
  text-align: left;
}

.tb-select__opt:hover {
  background: var(--color-secondary);
}
</style>
