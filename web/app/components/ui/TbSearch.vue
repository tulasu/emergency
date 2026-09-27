<script setup lang="ts">
const model = defineModel<string>({ default: '' });

withDefaults(
  defineProps<{
    placeholder?: string;
    disabled?: boolean;
  }>(),
  {
    placeholder: 'Поиск',
    disabled: false,
  },
);
</script>

<template>
  <label class="tb-search">
    <TbIcon name="search" />
    <input
      class="tb-search__input"
      type="search"
      :placeholder="placeholder"
      :disabled="disabled"
      :value="model"
      @input="model = ($event.target as HTMLInputElement).value"
    />
  </label>
</template>

<style scoped>
.tb-search {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  height: 48px;
  padding: 0 16px;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  color: var(--color-text-muted);
}

.tb-search:focus-within {
  border-color: var(--color-primary);
  color: var(--color-primary);
}

.tb-search__input {
  flex: 1;
  min-width: 0;
  border: 0;
  outline: none;
  background: transparent;
  color: var(--color-text);
  font: var(--font-body);
}

.tb-search__input::placeholder {
  color: var(--color-text-muted);
}

.tb-search__input:disabled {
  color: var(--color-text-subtle);
  cursor: default;
}
</style>
