<script setup lang="ts">
import type { Lesson, Module, ModuleSummary, Variant } from '~/types/curriculum';
import type { TbTabItem } from '~/types/ui';
import { apiErrorMessage } from '~/utils/api-error';
import {
  formatOpenedFor,
  formatPassedFraction,
  formatSuccessRate,
  lessonCountLabel,
  moduleDisplayStatus,
  moduleStatusLabel,
  moduleStatusTone,
} from '~/utils/curriculum-labels';

definePageMeta({
  middleware: ['auth', 'staff'],
});

const route = useRoute();
const moduleId = computed(() => String(route.params.id));

const modulesApi = useModules();
const lessonsApi = useLessons();
const variantsApi = useVariants();
const { groupOptions, load: loadGroups } = useGroups();

const tab = ref('lessons');
const summary = ref<ModuleSummary | null>(null);
const module = ref<Module | null>(null);
const lessons = ref<Lesson[]>([]);
const openModal = ref(false);
const openVariants = ref<Variant[]>([]);
const toast = ref('');
const error = ref('');
const loading = ref(true);
const assignGroupId = ref('');
const assignBusy = ref(false);
const selectedLessonId = ref('');

const tabs = computed<TbTabItem[]>(() => [
  { id: 'lessons', label: 'Занятия', count: lessons.value.length },
  {
    id: 'assign',
    label: 'Назначение',
    count: summary.value?.assignment?.total_users,
  },
  { id: 'summary', label: 'Сводка и допуск' },
]);

const attentionItems = computed(() => summary.value?.attention ?? []);

const displayStatus = computed(() =>
  module.value ? moduleDisplayStatus(module.value) : 'draft',
);

const headerSubtitle = computed(() => {
  if (!module.value) {
    return 'Загрузка…';
  }
  const parts = [
    module.value.description || '',
    lessonCountLabel(lessons.value.length),
  ];
  const assigned = summary.value?.assignment?.total_users ?? 0;
  if (assigned > 0) {
    parts.push(`назначен ${assigned} ученикам`);
  }
  return parts.filter(Boolean).join(' · ');
});

async function load(): Promise<void> {
  loading.value = true;
  error.value = '';
  try {
    const [sum, list] = await Promise.all([
      modulesApi.summary(moduleId.value),
      lessonsApi.listByModule(moduleId.value),
    ]);
    summary.value = sum;
    module.value = sum.module;
    lessons.value = list.length
      ? list.map((lesson) => {
          const fromSum = sum.lessons.find((l) => l.id === lesson.id);
          return { ...lesson, ...fromSum };
        })
      : sum.lessons;
    if (!selectedLessonId.value && lessons.value[0]) {
      selectedLessonId.value = lessons.value[0].id;
    }
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    loading.value = false;
  }
}

async function prepareOpen(): Promise<void> {
  openVariants.value = [];
  toast.value = '';
  try {
    for (const lesson of lessons.value) {
      const variants = await variantsApi.list(lesson.id);
      openVariants.value.push(...variants.filter((v) => v.status === 'approved'));
    }
    if (!openVariants.value.length) {
      for (const lesson of lessons.value) {
        openVariants.value.push(...(await variantsApi.list(lesson.id)));
      }
    }
    openModal.value = true;
  } catch (err) {
    error.value = apiErrorMessage(err);
  }
}

async function assignToGroup(): Promise<void> {
  if (!assignGroupId.value || !module.value) {
    return;
  }
  assignBusy.value = true;
  error.value = '';
  try {
    const picks: Array<{ lesson_id: string; variant_id: string }> = [];
    for (const lesson of lessons.value) {
      const variants = await variantsApi.list(lesson.id);
      const primary = variants.find((v) => v.is_primary) ?? variants[0];
      if (primary) {
        picks.push({ lesson_id: lesson.id, variant_id: primary.id });
      }
    }
    const { $api } = useNuxtApp();
    await $api(`/groups/${assignGroupId.value}/modules`, {
      method: 'POST',
      body: { module_id: module.value.id, lessons: picks },
    });
    toast.value = 'Модуль назначен группе';
    await load();
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    assignBusy.value = false;
  }
}

function openLesson(lesson: Lesson): void {
  selectedLessonId.value = lesson.id;
  void navigateTo(`/modules/${moduleId.value}/lessons/${lesson.id}`);
}

onMounted(async () => {
  await load();
  try {
    await loadGroups();
  } catch {
    // optional
  }
});
</script>

<template>
  <section class="module-page">
    <header class="module-page__header">
      <div class="module-page__titles">
        <nav class="curriculum-breadcrumbs" aria-label="Навигация">
          <NuxtLink to="/modules" class="curriculum-breadcrumbs__link">Модули</NuxtLink>
          <span class="curriculum-breadcrumbs__sep">/</span>
          <span class="curriculum-breadcrumbs__current">{{ module?.title || 'Модуль' }}</span>
        </nav>
        <div class="module-page__title-row">
          <h1 class="page-title">{{ module?.title || 'Модуль' }}</h1>
          <TbBadge v-if="module" :tone="moduleStatusTone(displayStatus)">
            {{ moduleStatusLabel(displayStatus) }}
          </TbBadge>
        </div>
        <p class="page-sub">{{ headerSubtitle }}</p>
      </div>
      <div class="module-page__actions">
        <TbButton variant="secondary" @click="navigateTo(`/modules/${moduleId}/edit`)">
          Изменить модуль
        </TbButton>
        <TbButton @click="prepareOpen">Открыть занятие</TbButton>
      </div>
    </header>

    <TbTabs v-model="tab" :tabs="tabs" />

    <p v-if="error" class="curriculum-error">{{ error }}</p>
    <p v-if="toast" class="toast">{{ toast }}</p>
    <p v-if="loading" class="curriculum-empty">Загрузка…</p>

    <div v-else-if="tab === 'lessons'" class="curriculum-layout">
      <div class="lessons-card">
        <div class="curriculum-table-head curriculum-table-head--lessons">
          <span></span>
          <span>Занятие</span>
          <span>Варианты</span>
          <span>Открыто для</span>
          <span>Сдали</span>
          <span>Средняя</span>
          <span></span>
        </div>
        <p v-if="!lessons.length" class="curriculum-empty">Занятий пока нет</p>
        <div v-else class="lessons-card__list">
          <LessonTableRow
            v-for="lesson in lessons"
            :key="lesson.id"
            :title="lesson.title"
            :variants-label="lesson.variants_label"
            :opened-label="formatOpenedFor(lesson.opened_for, lesson.opened_total).label"
            :opened-tone="formatOpenedFor(lesson.opened_for, lesson.opened_total).tone"
            :passed-label="formatPassedFraction(lesson.passed_rate, lesson.opened_for ?? lesson.opened_total)"
            :average-label="formatSuccessRate(lesson.avg_success)"
            :hint="lesson.attention"
            :selected="selectedLessonId === lesson.id"
            @click="openLesson(lesson)"
          />
        </div>
        <div class="lessons-card__add">
          <button type="button" class="link-action" @click="navigateTo('/lessons')">
            + Добавить занятие из пула
          </button>
          <button
            type="button"
            class="link-action"
            @click="navigateTo(`/modules/${moduleId}/edit`)"
          >
            + Создать новое занятие
          </button>
        </div>
      </div>
      <div class="curriculum-layout__side">
        <AssignmentSidePanel
          :groups="summary?.assignment?.groups"
          :individuals="summary?.assignment?.individuals"
          @edit="tab = 'assign'"
        />
        <AttentionPanel :items="attentionItems" />
      </div>
    </div>

    <div v-else-if="tab === 'assign'" class="module-page__panel">
      <h2 class="sheet__section-title">Назначение модуля</h2>
      <p class="page-sub">
        Назначьте модуль группе — студенты увидят его в личном кабинете.
      </p>
      <TbField label="Группа">
        <TbSelect
          v-model="assignGroupId"
          :options="groupOptions"
          placeholder="Выберите группу"
        />
      </TbField>
      <TbButton :busy="assignBusy" :disabled="!assignGroupId" @click="assignToGroup">
        Назначить группе
      </TbButton>
    </div>

    <div v-else class="module-page__panel">
      <h2 class="sheet__section-title">Сводка и допуск</h2>
      <p class="page-sub">
        Порог допуска модуля: {{ module?.success_threshold }}%. Ниже порога — нужна повторная
        попытка.
      </p>
      <ul class="side-panel__list">
        <li class="side-panel__row">
          <span>Занятий</span>
          <strong>{{ lessons.length }}</strong>
        </li>
        <li class="side-panel__row">
          <span>Назначено учеников</span>
          <strong>{{ summary?.assignment?.total_users ?? 0 }}</strong>
        </li>
        <li class="side-panel__row">
          <span>Успешность модуля</span>
          <strong>{{ formatSuccessRate(summary?.module.success_rate) }}</strong>
        </li>
        <li class="side-panel__row">
          <span>Требует внимания</span>
          <strong>{{ attentionItems.length }}</strong>
        </li>
      </ul>
      <div class="admit-table">
        <div class="admit-table__head">
          <span>Занятие</span>
          <span>Открыто</span>
          <span>Сдали</span>
          <span>Среднее</span>
          <span>Допуск</span>
        </div>
        <div v-for="lesson in lessons" :key="lesson.id" class="admit-table__row">
          <span>{{ lesson.title }}</span>
          <span>{{ lesson.opened_for ?? 0 }}/{{ lesson.opened_total ?? 0 }}</span>
          <span>{{ formatPassedFraction(lesson.passed_rate) }}</span>
          <span>{{ formatSuccessRate(lesson.avg_success) }}</span>
          <span>
            {{
              lesson.avg_success == null
                ? '—'
                : lesson.avg_success >= (module?.success_threshold ?? 70)
                  ? 'Открыт'
                  : 'Ниже порога'
            }}
          </span>
        </div>
      </div>
      <TbButton
        variant="secondary"
        @click="navigateTo(`/analytics?module=${moduleId}`)"
      >
        Открыть аналитику модуля
      </TbButton>
    </div>

    <OpenLessonModal
      v-model:open="openModal"
      :module-id="moduleId"
      :module-title="module?.title || ''"
      :lessons="lessons"
      :variants="openVariants"
      @done="toast = `Выдано попыток: ${$event}`"
    />
  </section>
</template>

<style scoped>
.module-page {
  display: flex;
  flex-direction: column;
  gap: 24px;
  width: 100%;
  min-height: 100%;
}

.module-page__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
}

.module-page__titles {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
  max-width: 720px;
}

.module-page__title-row {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.module-page__title-row .page-title {
  min-width: 0;
}

.module-page__actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  flex-shrink: 0;
}

.lessons-card {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 24px;
  background: var(--color-surface);
  border-radius: var(--radius-md);
}

.lessons-card__list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.lessons-card__add {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  padding: 12px 16px;
}

.link-action {
  border: 0;
  background: transparent;
  color: var(--color-primary);
  font: 500 14px/1.3 var(--font-sans);
  cursor: pointer;
  padding: 0;
}

.module-page__panel {
  padding: 24px;
  background: var(--color-surface);
  border-radius: var(--radius-md);
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-width: 960px;
}

.admit-table {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.admit-table__head,
.admit-table__row {
  display: grid;
  grid-template-columns: 1.6fr 1fr 1fr 1fr 1fr;
  gap: 10px;
  padding: 10px 12px;
  font: 500 13px/1.3 var(--font-sans);
}

.admit-table__head {
  color: var(--color-text-muted);
  font: 500 12px/1.3 var(--font-sans);
}

.admit-table__row {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg);
}

.toast {
  margin: 0;
  color: var(--color-success);
  font: var(--font-mute);
}

.curriculum-table-head--lessons {
  grid-template-columns: 16px minmax(200px, 344px) 200px 250px 120px 80px 18px;
  gap: 16px;
  padding: 0 16px 8px;
}

@media (max-width: 1100px) {
  .module-page__header {
    flex-direction: column;
  }

  .module-page__actions {
    width: 100%;
  }
}
</style>
