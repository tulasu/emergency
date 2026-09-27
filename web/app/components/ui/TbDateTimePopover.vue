<script setup lang="ts">
const model = defineModel<string>({ default: '' });

const open = ref(false);
const host = ref<HTMLElement | null>(null);

const localValue = computed({
  get() {
    if (!model.value) {
      return '';
    }
    const date = new Date(model.value);
    if (Number.isNaN(date.getTime())) {
      return model.value;
    }
    const pad = (n: number) => String(n).padStart(2, '0');
    return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
  },
  set(value: string) {
    if (!value) {
      model.value = '';
      return;
    }
    const date = new Date(value);
    model.value = Number.isNaN(date.getTime()) ? value : date.toISOString();
  },
});

const display = computed(() => {
  if (!model.value) {
    return 'Выберите дату';
  }
  const date = new Date(model.value);
  if (Number.isNaN(date.getTime())) {
    return model.value;
  }
  return date.toLocaleString('ru-RU', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
});

function toggle(event: Event): void {
  event.stopPropagation();
  open.value = !open.value;
}

function onDocumentClick(event: Event): void {
  if (!host.value?.contains(event.target as Node)) {
    open.value = false;
  }
}

onMounted(() => {
  document.addEventListener('click', onDocumentClick);
});

onUnmounted(() => {
  document.removeEventListener('click', onDocumentClick);
});
</script>

<template>
  <div ref="host" class="tb-dt">
    <button type="button" class="tb-dt__btn" @click="toggle">
      <TbIcon name="calendar" />
      <span>{{ display }}</span>
    </button>
    <div v-if="open" class="tb-dt__panel" @click.stop>
      <input v-model="localValue" class="tb-dt__input" type="datetime-local" />
    </div>
  </div>
</template>

<style scoped>
.tb-dt {
  position: relative;
  width: 100%;
}

.tb-dt__btn {
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
  gap: 10px;
  text-align: left;
}

.tb-dt__panel {
  position: absolute;
  z-index: 10;
  left: 0;
  right: 0;
  top: calc(100% + 4px);
  padding: 12px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-card);
}

.tb-dt__input {
  width: 100%;
  height: 40px;
  padding: 0 10px;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  font: var(--font-body);
}
</style>
