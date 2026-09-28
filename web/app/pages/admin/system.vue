<script setup lang="ts">
import { apiErrorMessage } from '~/utils/api-error';

definePageMeta({
  middleware: ['auth', 'admin'],
});

const { $api } = useNuxtApp();
const health = ref<Record<string, unknown> | null>(null);
const error = ref('');
const loading = ref(true);

async function load(): Promise<void> {
  loading.value = true;
  try {
    health.value = await $api<Record<string, unknown>>('/health');
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  void load();
});
</script>

<template>
  <section class="page-sheet">
    <div class="sheet">
      <header class="sheet__header">
        <nav class="curriculum-breadcrumbs" aria-label="Навигация">
          <NuxtLink to="/admin" class="curriculum-breadcrumbs__link">Админ</NuxtLink>
          <span class="curriculum-breadcrumbs__sep">/</span>
          <span>Состояние системы</span>
        </nav>
        <h1 class="page-title">Состояние системы</h1>
        <p class="page-sub">API health check и служебные метрики</p>
      </header>
      <div class="sheet__rule"></div>
      <div class="sheet__body">
        <p v-if="error" class="curriculum-error">{{ error }}</p>
        <p v-if="loading" class="curriculum-empty">Загрузка…</p>
        <template v-else>
          <div class="card">
            <h2>API и веб-сервер</h2>
            <p>Статус: {{ health?.status ?? health?.ok ?? 'ok' }}</p>
            <p v-if="health?.version">Версия: {{ health.version }}</p>
            <pre>{{ JSON.stringify(health, null, 2) }}</pre>
          </div>
          <div class="card">
            <h2>Резервное копирование</h2>
            <p class="page-sub">
              Управление бэкапами выполняется на инфраструктурном уровне (вне приложения).
            </p>
          </div>
        </template>
      </div>
    </div>
  </section>
</template>

<style scoped>
.card {
  padding: 20px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.card h2 {
  margin: 0;
  font: 700 16px/1.3 var(--font-sans);
}

.card p {
  margin: 0;
}

.card pre {
  margin: 8px 0 0;
  padding: 12px;
  background: var(--color-bg);
  border-radius: var(--radius-sm);
  overflow: auto;
  font: 400 12px/1.4 ui-monospace, monospace;
}
</style>
