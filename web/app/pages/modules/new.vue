<script setup lang="ts">
import type { Lesson } from '~/types/curriculum';
import { apiErrorMessage } from '~/utils/api-error';

definePageMeta({
  middleware: ['auth', 'staff'],
});

const draft = useModuleDraftStore();
const modulesApi = useModules();
const lessonsApi = useLessons();
const variantsApi = useVariants();
const { groupOptions, load: loadGroups } = useGroups();
const { $api } = useNuxtApp();
const steps = ['Основное и занятия', 'Назначение', 'Проверка и создание'];
const poolOpen = ref(false);
const poolQuery = ref('');
const poolItems = ref<Lesson[]>([]);
const poolLoading = ref(false);
const busy = ref(false);
const error = ref('');

const allLessonsReady = computed(
  () => draft.lessons.length > 0 && draft.lessons.every((lesson) => Boolean(lesson.sourceLessonId)),
);

const readiness = computed(() => [
  { label: 'Название и описание', done: Boolean(draft.title.trim()) },
  { label: 'Хотя бы одно занятие', done: draft.lessons.length > 0 },
  {
    label: 'У каждого занятия есть утверждённый вариант',
    done: allLessonsReady.value,
  },
]);

const readinessNote = computed(() => {
  const pending = draft.lessons.find((lesson) => !lesson.sourceLessonId);
  if (pending) {
    return `Модуль можно назначить и сейчас. «${pending.title || 'Занятие'}» не получится открыть ученикам, пока у него нет утверждённого варианта.`;
  }
  return 'Черновик можно сохранить в любой момент. Ученики увидят модуль после назначения и открытия занятий.';
});

const nextLabel = computed(() => {
  if (draft.step === 0) {
    return 'Далее: назначение →';
  }
  if (draft.step === 1) {
    return 'Далее: проверка →';
  }
  return 'Создать и назначить';
});

onMounted(async () => {
  draft.reset();
  try {
    await loadGroups();
  } catch {
    // optional
  }
});

async function openPool(): Promise<void> {
  poolOpen.value = true;
  poolLoading.value = true;
  try {
    poolItems.value = await lessonsApi.pool(poolQuery.value.trim());
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    poolLoading.value = false;
  }
}

async function searchPool(): Promise<void> {
  poolLoading.value = true;
  try {
    poolItems.value = await lessonsApi.pool(poolQuery.value.trim());
  } finally {
    poolLoading.value = false;
  }
}

function pickPool(lesson: Lesson): void {
  draft.addFromPool(lesson.id, lesson.title);
  poolOpen.value = false;
}

function next(): void {
  error.value = '';
  if (draft.step === 0) {
    if (!draft.title.trim()) {
      error.value = 'Укажите название модуля';
      return;
    }
    draft.step = 1;
    return;
  }
  if (draft.step === 1) {
    draft.step = 2;
    return;
  }
  void create();
}

function back(): void {
  draft.step = Math.max(draft.step - 1, 0);
}

async function create(): Promise<void> {
  busy.value = true;
  error.value = '';
  try {
    const mod = await modulesApi.create({
      title: draft.title.trim(),
      description: draft.description.trim(),
    });
    await modulesApi.update(mod.id, {
      title: draft.title.trim(),
      description: draft.description.trim(),
      success_threshold: draft.successThreshold,
      status: 'draft',
    });
    const createdLessons: Lesson[] = [];
    let position = 0;
    for (const lesson of draft.lessons) {
      if (lesson.sourceLessonId) {
        createdLessons.push(
          await lessonsApi.copyFromPool(mod.id, lesson.sourceLessonId, position),
        );
      } else {
        createdLessons.push(
          await lessonsApi.create(mod.id, {
            title: lesson.title.trim() || `Занятие ${position + 1}`,
            position,
          }),
        );
      }
      position += 1;
    }
    if (draft.assignGroupId && createdLessons.length) {
      const picks: Array<{ lesson_id: string; variant_id: string }> = [];
      for (const lesson of createdLessons) {
        const variants = await variantsApi.list(lesson.id);
        const primary = variants.find((v) => v.is_primary) ?? variants[0];
        if (primary) {
          picks.push({ lesson_id: lesson.id, variant_id: primary.id });
        }
      }
      if (picks.length) {
        await $api(`/groups/${draft.assignGroupId}/modules`, {
          method: 'POST',
          body: { module_id: mod.id, lessons: picks },
        });
      }
    }
    draft.reset();
    await navigateTo(`/modules/${mod.id}`);
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    busy.value = false;
  }
}
</script>

<template>
  <section class="wizard-page">
    <header class="wizard-page__header">
      <div class="wizard-page__titles">
        <nav class="curriculum-breadcrumbs" aria-label="Навигация">
          <NuxtLink to="/modules" class="curriculum-breadcrumbs__link">Модули</NuxtLink>
          <span class="curriculum-breadcrumbs__sep">/</span>
          <span class="curriculum-breadcrumbs__current">Новый модуль</span>
        </nav>
        <h1 class="page-title">Новый модуль</h1>
      </div>
      <div class="wizard-page__actions">
        <TbButton variant="ghost" @click="navigateTo('/modules')">Отмена</TbButton>
        <TbButton variant="secondary" :busy="busy" @click="create">Сохранить черновик</TbButton>
      </div>
    </header>

    <TbStepper :steps="steps" :current="draft.step" />

    <div class="curriculum-layout">
      <div class="wizard-card">
        <div v-if="draft.step === 0" class="curriculum-form curriculum-form--wide">
          <div class="title-threshold">
            <TbField label="Название модуля" class="grow">
              <TbInput v-model="draft.title" placeholder="Название модуля" />
            </TbField>
            <TbField label="Порог успешности">
              <TbInput
                :model-value="String(draft.successThreshold)"
                type="number"
                @update:model-value="draft.successThreshold = Number($event) || 0"
              />
            </TbField>
          </div>
          <TbField label="Описание" hint="видят ученики на странице модуля">
            <TbTextarea
              v-model="draft.description"
              :rows="3"
              :maxlength="255"
              show-counter
            />
          </TbField>

          <div class="lessons-block">
            <div class="lessons-block__head">
              <div>
                <h2 class="sheet__section-title">Занятия · {{ draft.lessons.length }}</h2>
                <p class="page-sub">
                  Порядок — как ученики увидят занятия в модуле. Открываете их вы.
                </p>
              </div>
              <div class="curriculum-actions">
                <TbButton variant="secondary" @click="openPool">+ Из пула занятий</TbButton>
                <TbButton @click="draft.addEmptyLesson()">+ Новое занятие</TbButton>
              </div>
            </div>

            <p v-if="!draft.lessons.length" class="curriculum-empty">Пока нет занятий</p>
            <div v-else class="lessons-list">
              <div
                v-for="lesson in draft.lessons"
                :key="lesson.key"
                class="draft-lesson"
                :class="{ 'draft-lesson--warn': !lesson.sourceLessonId }"
              >
                <span class="draft-lesson__drag" aria-hidden="true">⋮⋮</span>
                <div class="draft-lesson__text">
                  <TbInput v-model="lesson.title" placeholder="Название занятия" />
                  <p class="draft-lesson__meta">
                    {{ lesson.sourceLessonId ? 'из пула занятий' : 'новое занятие' }}
                  </p>
                </div>
                <span class="draft-lesson__variants">
                  {{ lesson.sourceLessonId ? 'A★' : '—' }}
                </span>
                <div class="draft-lesson__status">
                  <TbBadge :tone="lesson.sourceLessonId ? 'success' : 'warning'">
                    {{
                      lesson.sourceLessonId ? 'Готово к открытию' : 'Нет утверждённого варианта'
                    }}
                  </TbBadge>
                </div>
                <div class="draft-lesson__actions">
                  <button
                    v-if="!lesson.sourceLessonId"
                    type="button"
                    class="draft-lesson__fix"
                    @click="openPool"
                  >
                    Собрать вариант
                  </button>
                  <TbButton
                    variant="icon"
                    aria-label="Удалить"
                    @click="draft.removeLesson(lesson.key)"
                  >
                    <TbIcon name="x" />
                  </TbButton>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div v-else-if="draft.step === 1" class="curriculum-form">
          <h2 class="sheet__section-title">Назначение</h2>
          <p class="page-sub">Можно назначить сразу или позже на странице модуля.</p>
          <TbField label="Группа">
            <TbSelect
              v-model="draft.assignGroupId"
              :options="[{ value: '', label: 'Не назначать сейчас' }, ...groupOptions]"
            />
          </TbField>
        </div>

        <div v-else class="curriculum-form">
          <h2 class="sheet__section-title">Проверка и создание</h2>
          <p><strong>{{ draft.title }}</strong></p>
          <p class="page-sub">{{ draft.description || 'Без описания' }}</p>
          <p class="page-sub">Порог успеха: {{ draft.successThreshold }}%</p>
          <p class="page-sub">Занятий: {{ draft.lessons.length }}</p>
          <ul v-if="draft.lessons.length" class="review-list">
            <li v-for="lesson in draft.lessons" :key="lesson.key">
              {{ lesson.title }}
              <span v-if="lesson.sourceLessonId">(из пула)</span>
            </li>
          </ul>
        </div>

        <p v-if="error" class="curriculum-error">{{ error }}</p>
        <div v-if="draft.step > 0" class="curriculum-actions">
          <TbButton variant="secondary" @click="back">Назад</TbButton>
        </div>
      </div>

      <div class="curriculum-layout__side">
        <ReadinessPanel
          :items="readiness"
          :note="readinessNote"
          :next-label="nextLabel"
          :next-busy="busy"
          :next-disabled="draft.step === 0 && !draft.title.trim()"
          @next="next"
        />
      </div>
    </div>

    <TbModal v-model:open="poolOpen" title="Пул занятий">
      <div class="curriculum-form">
        <TbSearch v-model="poolQuery" @update:model-value="searchPool" />
        <p v-if="poolLoading" class="curriculum-empty">Загрузка…</p>
        <p v-else-if="!poolItems.length" class="curriculum-empty">Ничего не найдено</p>
        <button
          v-for="lesson in poolItems"
          :key="lesson.id"
          type="button"
          class="curriculum-list-item pool-item"
          @click="pickPool(lesson)"
        >
          <span>{{ lesson.title }}</span>
          <TbIcon name="plus" />
        </button>
      </div>
    </TbModal>
  </section>
</template>

<style scoped>
.wizard-page {
  display: flex;
  flex-direction: column;
  gap: 20px;
  width: 100%;
  min-height: 100%;
}

.wizard-page__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
}

.wizard-page__titles {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}

.wizard-page__actions {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
}

.wizard-card {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 24px;
  background: var(--color-surface);
  border-radius: var(--radius-md);
}

.title-threshold {
  display: grid;
  grid-template-columns: 1fr 220px;
  gap: 16px;
}

.grow {
  min-width: 0;
}

.lessons-block {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.lessons-block__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.lessons-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.draft-lesson {
  display: grid;
  grid-template-columns: 16px minmax(180px, 1fr) 120px 210px auto;
  gap: 14px;
  align-items: center;
  padding: 12px 14px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-surface);
  box-sizing: border-box;
}

.draft-lesson--warn {
  background: var(--color-warning-soft);
  border-color: var(--color-warning);
}

.draft-lesson__drag {
  color: var(--color-text-subtle);
  font-size: 12px;
  letter-spacing: -2px;
  line-height: 1;
}

.draft-lesson__text {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.draft-lesson__meta {
  margin: 0;
  color: var(--color-text-subtle);
  font: 400 12px/1.3 var(--font-sans);
}

.draft-lesson__variants {
  color: var(--color-text-muted);
  font: 500 13px/1.3 var(--font-sans);
}

.draft-lesson__status {
  justify-self: start;
}

.draft-lesson__actions {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  justify-self: end;
}

.draft-lesson__fix {
  border: 0;
  background: transparent;
  color: var(--color-primary);
  font: 600 13px/1.3 var(--font-sans);
  cursor: pointer;
  padding: 0;
  white-space: nowrap;
}

.review-list {
  margin: 0;
  padding-left: 18px;
  color: var(--color-text-muted);
  font: var(--font-mute);
}

.pool-item {
  width: 100%;
  background: transparent;
  cursor: pointer;
  font: inherit;
  text-align: left;
}

@media (max-width: 1100px) {
  .draft-lesson {
    grid-template-columns: 16px 1fr auto;
  }

  .draft-lesson__variants,
  .draft-lesson__status {
    display: none;
  }
}

@media (max-width: 700px) {
  .title-threshold {
    grid-template-columns: 1fr;
  }

  .wizard-page__header {
    flex-direction: column;
  }
}
</style>
