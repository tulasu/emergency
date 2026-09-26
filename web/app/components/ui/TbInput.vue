<script setup lang="ts">
const model = defineModel<string>({ default: '' });

withDefaults(
  defineProps<{
    type?: string;
    placeholder?: string;
    autocomplete?: string;
    disabled?: boolean;
  }>(),
  {
    type: 'text',
    placeholder: '',
    autocomplete: '',
    disabled: false,
  },
);
</script>

<template>
  <span class="tb-input-host">
    <input
      class="tb-input"
      :type="type"
      :placeholder="placeholder"
      :autocomplete="autocomplete || undefined"
      :disabled="disabled"
      :value="model"
      @input="model = ($event.target as HTMLInputElement).value"
    />
  </span>
</template>

<style scoped>
.tb-input-host {
  display: block;
  width: 100%;
}

.tb-input {
  width: 100%;
  height: 48px;
  padding: 0 16px;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  color: var(--color-text);
  font: var(--font-body);
  outline: none;
}

.tb-input::placeholder {
  color: var(--color-text-muted);
  font: var(--font-body);
}

.tb-input:focus {
  border-color: var(--color-primary);
}

.tb-input:disabled {
  color: var(--color-text-subtle);
  cursor: default;
}
</style>
