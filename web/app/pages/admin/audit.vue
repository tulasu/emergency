<script setup lang="ts">
definePageMeta({
  middleware: ['auth', 'admin'],
});

const events = ref<
  Array<{ time: string; user: string; action: string; object: string; result: string }>
>([]);
</script>

<template>
  <section class="page-sheet">
    <div class="sheet">
      <header class="sheet__header">
        <nav class="curriculum-breadcrumbs" aria-label="Навигация">
          <NuxtLink to="/admin" class="curriculum-breadcrumbs__link">Админ</NuxtLink>
          <span class="curriculum-breadcrumbs__sep">/</span>
          <span>Журнал аудита</span>
        </nav>
        <h1 class="page-title">Журнал аудита</h1>
        <p class="page-sub">
          Записи действий пользователей. Persistent audit log на сервере пока не подключён —
          экран готов к приёму событий.
        </p>
      </header>
      <div class="sheet__rule"></div>
      <div class="sheet__body">
        <p v-if="!events.length" class="curriculum-empty">
          Событий пока нет. После появления API `/admin/audit` записи появятся здесь.
        </p>
        <div v-else class="table">
          <div
            v-for="(ev, idx) in events"
            :key="idx"
            class="table__row"
          >
            <span>{{ ev.time }}</span>
            <span>{{ ev.user }}</span>
            <span>{{ ev.action }}</span>
            <span>{{ ev.object }}</span>
            <span>{{ ev.result }}</span>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.table__row {
  display: grid;
  grid-template-columns: 140px 1fr 1.2fr 1.2fr 100px;
  gap: 12px;
  padding: 12px 16px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  margin-bottom: 4px;
  background: var(--color-surface);
  font: 500 13px/1.3 var(--font-sans);
}
</style>
