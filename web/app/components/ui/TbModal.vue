<script setup lang="ts">
const open = defineModel<boolean>('open', { default: false });

defineProps<{
  title: string;
  subtitle?: string;
}>();

const emit = defineEmits<{
  close: [];
}>();

function close(): void {
  open.value = false;
  emit('close');
}

function onKeydown(event: KeyboardEvent): void {
  if (event.key === 'Escape' && open.value) {
    close();
  }
}

onMounted(() => {
  document.addEventListener('keydown', onKeydown);
});

onUnmounted(() => {
  document.removeEventListener('keydown', onKeydown);
});
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="tb-modal" role="dialog" aria-modal="true">
      <button class="tb-modal__backdrop" type="button" aria-label="Закрыть" @click="close" />
      <div class="tb-modal__panel">
        <header class="tb-modal__header">
          <div class="tb-modal__titles">
            <h2>{{ title }}</h2>
            <p v-if="subtitle" class="tb-modal__subtitle">{{ subtitle }}</p>
          </div>
          <TbButton variant="icon" aria-label="Закрыть" @click="close">
            <TbIcon name="x" />
          </TbButton>
        </header>
        <div class="tb-modal__body">
          <slot />
        </div>
        <footer v-if="$slots.footer" class="tb-modal__footer">
          <slot name="footer" />
        </footer>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.tb-modal {
  position: fixed;
  inset: 0;
  z-index: 40;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}

.tb-modal__backdrop {
  position: absolute;
  inset: 0;
  border: 0;
  background: #14141466;
  cursor: pointer;
}

.tb-modal__panel {
  position: relative;
  z-index: 1;
  width: min(760px, 100%);
  max-height: min(90vh, 900px);
  display: flex;
  flex-direction: column;
  background: var(--color-surface);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-card);
  overflow: hidden;
}

.tb-modal__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding: 28px 28px 0;
}

.tb-modal__titles {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.tb-modal__header h2 {
  margin: 0;
  font: 700 20px/1.3 var(--font-sans);
  color: var(--color-text);
}

.tb-modal__subtitle {
  margin: 0;
  color: var(--color-text-muted);
  font: var(--font-cap);
}

.tb-modal__body {
  padding: 20px 28px;
  overflow: auto;
}

.tb-modal__footer {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 28px 28px;
  border-top: 1px solid var(--color-border);
}
</style>
