<script setup lang="ts">
const model = defineModel<string>({ default: '' });

const props = withDefaults(
  defineProps<{
    placeholder?: string;
    disabled?: boolean;
    rows?: number;
    maxlength?: number;
    showCounter?: boolean;
  }>(),
  {
    placeholder: '',
    disabled: false,
    rows: 4,
    showCounter: false,
  },
);

const count = computed(() => model.value.length);
</script>

<template>
  <div class="tb-textarea">
    <textarea
      class="tb-textarea__field"
      :placeholder="placeholder"
      :disabled="disabled"
      :rows="rows"
      :maxlength="maxlength"
      :value="model"
      @input="model = ($event.target as HTMLTextAreaElement).value"
    />
    <div v-if="showCounter || maxlength" class="tb-textarea__meta">
      <span>{{ count }}{{ maxlength ? ` / ${maxlength}` : '' }}</span>
    </div>
  </div>
</template>

<style scoped>
.tb-textarea {
  display: flex;
  flex-direction: column;
  gap: 6px;
  width: 100%;
}

.tb-textarea__field {
  width: 100%;
  min-height: 96px;
  padding: 12px 16px;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  color: var(--color-text);
  font: var(--font-body);
  resize: vertical;
  outline: none;
}

.tb-textarea__field::placeholder {
  color: var(--color-text-muted);
}

.tb-textarea__field:focus {
  border-color: var(--color-primary);
}

.tb-textarea__field:disabled {
  color: var(--color-text-subtle);
  cursor: default;
}

.tb-textarea__meta {
  display: flex;
  justify-content: flex-end;
  color: var(--color-text-subtle);
  font: var(--font-cap);
}
</style>
