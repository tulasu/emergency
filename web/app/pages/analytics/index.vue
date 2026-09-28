<script setup lang="ts">
import type { Lesson, Module, ModuleSummary } from '~/types/curriculum';
import { apiErrorMessage } from '~/utils/api-error';
import { formatSuccessRate } from '~/utils/curriculum-labels';

definePageMeta({
  middleware: ['auth', 'staff'],
});

const modulesApi = useModules();
const lessonsApi = useLessons();
const variantsApi = useVariants();
const route = useRoute();

const modules = ref<Module[]>([]);
const selectedModuleId = ref(String(route.query.module || ''));
const summary = ref<ModuleSummary | null>(null);
const lessons = ref<Lesson[]>([]);
const hardest = ref<
  Array<{ title: string; rate: number | null; variant: string; attempts: number }>
>([]);
const loading = ref(true);
const error = ref('');

async function load(): Promise<void> {
  loading.value = true;
  error.value = '';
  try {
    modules.value = await modulesApi.list();
    if (!selectedModuleId.value && modules.value[0]) {
      selectedModuleId.value = modules.value[0].id;
    }
    if (selectedModuleId.value) {
      await loadModule(selectedModuleId.value);
    }
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    loading.value = false;
  }
}

async function loadModule(moduleId: string): Promise<void> {
  const [sum, lessonList] = await Promise.all([
    modulesApi.summary(moduleId),
    lessonsApi.listByModule(moduleId),
  ]);
  summary.value = sum;
  lessons.value = lessonList.length ? lessonList : sum.lessons;
  const rows: typeof hardest.value = [];
  for (const lesson of lessons.value) {
    try {
      const variants = await variantsApi.list(lesson.id);
      for (const v of variants) {
        if (v.hardest_ticket_title) {
          rows.push({
            title: v.hardest_ticket_title,
            rate: v.hardest_ticket_rate ?? null,
            variant: v.title,
            attempts: v.attempt_count ?? 0,
          });
        }
      }
    } catch {
      /* skip */
    }
  }
  hardest.value = rows
    .sort((a, b) => (a.rate ?? 100) - (b.rate ?? 100))
    .slice(0, 12);
}

onMounted(() => {
  void load();
});

watch(selectedModuleId, (id) => {
  if (id) {
    void loadModule(id);
  }
});
</script>

<template>
  <section class="page-sheet page-sheet--wide">
    <div class="sheet">
      <header class="sheet__header">
        <div class="sheet__header-row">
          <div>
            <h1 class="page-title">Аналитика</h1>
            <p class="page-sub">
              Успешность по занятиям и проблемные карточки модуля
            </p>
          </div>
          <TbSelect
            v-model="selectedModuleId"
            :options="modules.map((m) => ({ value: m.id, label: m.title }))"
          />
        </div>
      </header>
      <div class="sheet__rule"></div>
      <div class="sheet__body">
        <p v-if="error" class="curriculum-error">{{ error }}</p>
        <p v-if="loading" class="curriculum-empty">Загрузка…</p>
        <template v-else>
          <div class="stats">
            <div class="stat">
              <span>Назначено</span>
              <strong>{{ summary?.assignment?.total_users ?? 0 }}</strong>
            </div>
            <div class="stat">
              <span>Занятий</span>
              <strong>{{ lessons.length }}</strong>
            </div>
            <div class="stat">
              <span>Успешность модуля</span>
              <strong>{{ formatSuccessRate(summary?.module.success_rate) }}</strong>
            </div>
            <div class="stat">
              <span>Внимание</span>
              <strong>{{ summary?.attention?.length ?? 0 }}</strong>
            </div>
          </div>

          <h2 class="sheet__section-title">Занятия × успешность</h2>
          <div class="table">
            <div class="table__head">
              <span>Занятие</span>
              <span>Открыто</span>
              <span>Сдали</span>
              <span>Среднее</span>
            </div>
            <div v-for="lesson in lessons" :key="lesson.id" class="table__row">
              <span>{{ lesson.title }}</span>
              <span>{{ lesson.opened_for ?? 0 }} / {{ lesson.opened_total ?? 0 }}</span>
              <span>{{ formatSuccessRate(lesson.passed_rate) }}</span>
              <span>{{ formatSuccessRate(lesson.avg_success) }}</span>
            </div>
          </div>

          <h2 class="sheet__section-title">Проблемные карточки</h2>
          <p class="page-sub">
            Карточки с низкой успешностью — скорее дефект контента, чем слабость группы
          </p>
          <div class="table">
            <div class="table__head">
              <span>Карточка</span>
              <span>Вариант</span>
              <span>Попыток</span>
              <span>Успешность</span>
            </div>
            <div v-for="(row, idx) in hardest" :key="idx" class="table__row">
              <span>{{ row.title }}</span>
              <span>{{ row.variant }}</span>
              <span>{{ row.attempts }}</span>
              <span>{{ formatSuccessRate(row.rate) }}</span>
            </div>
            <p v-if="!hardest.length" class="curriculum-empty">Данных пока недостаточно</p>
          </div>
        </template>
      </div>
    </div>
  </section>
</template>

<style scoped>
.stats {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.stat {
  padding: 16px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.stat span {
  color: var(--color-text-muted);
  font: 500 12px/1.3 var(--font-sans);
}

.stat strong {
  font: 700 22px/1.2 var(--font-sans);
}

.table {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.table__head,
.table__row {
  display: grid;
  grid-template-columns: 1.6fr 1fr 1fr 1fr;
  gap: 12px;
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
  font: 500 14px/1.3 var(--font-sans);
}

@media (max-width: 900px) {
  .stats {
    grid-template-columns: 1fr 1fr;
  }
}
</style>
