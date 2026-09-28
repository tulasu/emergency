<script setup lang="ts">
import type { Attempt } from '~/types/curriculum';
import { apiErrorMessage } from '~/utils/api-error';
import { attemptStatusLabel, isOpenAttemptStatus } from '~/utils/curriculum-labels';

definePageMeta({
  middleware: 'auth',
});

const attemptsApi = useAttempts();
const rows = ref<
  Array<{
    moduleTitle: string;
    lessonTitle: string;
    attempt: Attempt;
  }>
>([]);
const loading = ref(true);
const error = ref('');

async function load(): Promise<void> {
  loading.value = true;
  error.value = '';
  try {
    const modules = await attemptsApi.listMyModules();
    const out: typeof rows.value = [];
    for (const mod of modules) {
      for (const lesson of mod.lessons) {
        try {
          const mine = await attemptsApi.listMine(lesson.variant.id);
          for (const at of mine) {
            out.push({
              moduleTitle: mod.module.title,
              lessonTitle: lesson.lesson.title,
              attempt: at,
            });
          }
        } catch {
          /* skip */
        }
      }
    }
    out.sort((a, b) => (b.attempt.attempt_no || 0) - (a.attempt.attempt_no || 0));
    rows.value = out;
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    loading.value = false;
  }
}

function openAttempt(at: Attempt): void {
  if (isOpenAttemptStatus(at.status) || at.status === 'available') {
    void navigateTo(`/learn/attempts/${at.id}`);
  } else {
    void navigateTo(`/learn/attempts/${at.id}/result`);
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
        <h1 class="page-title">Мои результаты</h1>
        <p class="page-sub">История попыток по назначенным модулям</p>
      </header>
      <div class="sheet__rule"></div>
      <div class="sheet__body">
        <p v-if="error" class="curriculum-error">{{ error }}</p>
        <p v-if="loading" class="curriculum-empty">Загрузка…</p>
        <p v-else-if="!rows.length" class="curriculum-empty">Попыток пока нет</p>
        <div v-else class="table">
          <div class="table__head">
            <span>Модуль</span>
            <span>Занятие</span>
            <span>Попытка</span>
            <span>Статус</span>
            <span>Оценка</span>
            <span></span>
          </div>
          <button
            v-for="row in rows"
            :key="row.attempt.id"
            type="button"
            class="table__row"
            @click="openAttempt(row.attempt)"
          >
            <span>{{ row.moduleTitle }}</span>
            <span>{{ row.lessonTitle }}</span>
            <span>№ {{ row.attempt.attempt_no }}</span>
            <span>{{ attemptStatusLabel(row.attempt.status) }}</span>
            <span>{{ row.attempt.score != null ? `${row.attempt.score}%` : '—' }}</span>
            <span class="link">Открыть</span>
          </button>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.table {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.table__head,
.table__row {
  display: grid;
  grid-template-columns: 1.4fr 1.4fr 90px 140px 80px 90px;
  gap: 12px;
  align-items: center;
  padding: 12px 16px;
}

.table__head {
  color: var(--color-text-muted);
  font: 500 12px/1.3 var(--font-sans);
}

.table__row {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  text-align: left;
  cursor: pointer;
  color: inherit;
  font: 500 14px/1.3 var(--font-sans);
}

.table__row:hover {
  border-color: color-mix(in srgb, var(--color-primary) 30%, transparent);
}

.link {
  color: var(--color-primary);
  font: 500 13px/1.3 var(--font-sans);
}

@media (max-width: 900px) {
  .table__head {
    display: none;
  }

  .table__row {
    grid-template-columns: 1fr 1fr;
  }
}
</style>
