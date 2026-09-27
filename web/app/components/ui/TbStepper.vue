<script setup lang="ts">
defineProps<{
  steps: string[];
  current: number;
}>();
</script>

<template>
  <ol class="tb-stepper">
    <li
      v-for="(step, index) in steps"
      :key="step"
      class="tb-stepper__item"
      :class="{
        'tb-stepper__item--done': index < current,
        'tb-stepper__item--current': index === current,
      }"
    >
      <span class="tb-stepper__dot">
        <TbIcon v-if="index < current" name="check" />
        <span v-else>{{ index + 1 }}</span>
      </span>
      <span class="tb-stepper__label">{{ step }}</span>
      <span v-if="index < steps.length - 1" class="tb-stepper__line" aria-hidden="true"></span>
    </li>
  </ol>
</template>

<style scoped>
.tb-stepper {
  display: flex;
  align-items: center;
  gap: 0;
  margin: 0;
  padding: 0;
  list-style: none;
  width: 100%;
}

.tb-stepper__item {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--color-text-subtle);
  font: 500 15px/1.3 var(--font-sans);
  flex: 1;
  min-width: 0;
}

.tb-stepper__item--current,
.tb-stepper__item--done {
  color: var(--color-text);
}

.tb-stepper__item--current .tb-stepper__label {
  font-weight: 700;
}

.tb-stepper__dot {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  border: 1px solid var(--color-border-strong);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  font: 600 13px/1 var(--font-sans);
}

.tb-stepper__item--current .tb-stepper__dot {
  background: var(--color-primary);
  border-color: var(--color-primary);
  color: var(--color-primary-text);
}

.tb-stepper__item--done .tb-stepper__dot {
  background: var(--color-secondary);
  border-color: var(--color-primary);
  color: var(--color-primary);
}

.tb-stepper__line {
  flex: 1;
  height: 1px;
  margin: 0 8px;
  background: var(--color-border);
}

.tb-stepper__label {
  white-space: nowrap;
}

@media (max-width: 720px) {
  .tb-stepper__label {
    display: none;
  }
}
</style>
