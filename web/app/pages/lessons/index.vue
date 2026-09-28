<script setup lang="ts">
import type { Lesson, Module } from '~/types/curriculum';
import { apiErrorMessage } from '~/utils/api-error';
import { ruCount } from '~/utils/ru-count';

definePageMeta({
  middleware: ['auth', 'staff'],
});

const lessonsApi = useLessons();
const modulesApi = useModules();
const query = ref('');
const lessons = ref<Lesson[]>([]);
const modules = ref<Module[]>([]);
const loading = ref(true);
const error = ref('');
const toast = ref('');
const assignOpen = ref(false);
const assignBusy = ref(false);
const assignLesson = ref<Lesson | null>(null);
const targetModuleId = ref('');

let timer: ReturnType<typeof setTimeout> | null = null;

const moduleTitle = computed(() => {
  const map = new Map(modules.value.map((m) => [m.id, m.title]));
  return (id: string) => map.get(id) || `Модуль ${id.slice(0, 8)}…`;
});

const folders = computed(() => {
  const map = new Map<string, Lesson[]>();
  for (const lesson of lessons.value) {
    const list = map.get(lesson.module_id) || [];
    list.push(lesson);
    map.set(lesson.module_id, list);
  }
  return Array.from(map.entries()).map(([moduleId, items], index) => ({
    moduleId,
    title: `Модуль ${index + 1} · ${moduleTitle.value(moduleId)}`,
    items,
  }));
});

const moduleOptions = computed(() =>
  modules.value.map((m) => ({ value: m.id, label: m.title })),
);

function lessonMeta(lesson: Lesson): string {
  if (lesson.ticket_count != null) {
    return ruCount(lesson.ticket_count, 'карта', 'карты', 'карт');
  }
  if (lesson.archived_at) {
    return 'Архив';
  }
  return lesson.variants_label || '';
}

function lessonTags(lesson: Lesson): string[] {
  const tags: string[] = [];
  if (lesson.variants_label) {
    const parts = lesson.variants_label
      .split(/[·,]/)
      .map((part) => part.trim())
      .filter(Boolean)
      .slice(0, 2);
    tags.push(...parts);
  }
  if (!tags.length && lesson.attention) {
    tags.push(lesson.attention);
  }
  if (lesson.archived_at) {
    tags.push('Архив');
  }
  if (!tags.length) {
    tags.push('Пул');
  }
  return tags.slice(0, 2);
}

async function load(): Promise<void> {
  loading.value = true;
  error.value = '';
  try {
    const [pool, mods] = await Promise.all([
      lessonsApi.pool(query.value.trim()),
      modulesApi.list('', 'all'),
    ]);
    lessons.value = pool;
    modules.value = mods;
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    loading.value = false;
  }
}

function onSearch(): void {
  if (timer) {
    clearTimeout(timer);
  }
  timer = setTimeout(() => {
    void load();
  }, 250);
}

function openAssign(lesson: Lesson): void {
  assignLesson.value = lesson;
  targetModuleId.value =
    modules.value.find((m) => m.id !== lesson.module_id)?.id || modules.value[0]?.id || '';
  assignOpen.value = true;
  error.value = '';
  toast.value = '';
}

async function assignToModule(): Promise<void> {
  if (!assignLesson.value || !targetModuleId.value) {
    return;
  }
  assignBusy.value = true;
  error.value = '';
  try {
    const existing = await lessonsApi.listByModule(targetModuleId.value);
    await lessonsApi.copyFromPool(
      targetModuleId.value,
      assignLesson.value.id,
      existing.length,
    );
    assignOpen.value = false;
    toast.value = 'Занятие добавлено в модуль';
    await navigateTo(`/modules/${targetModuleId.value}`);
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    assignBusy.value = false;
  }
}

onMounted(() => {
  void load();
});
</script>

<template>
  <section class="pool-page">
    <header class="pool-page__header">
      <h1 class="page-title">Занятия</h1>
      <div class="pool-page__actions">
        <div class="search-wrap">
          <TbSearch v-model="query" placeholder="Поиск" @update:model-value="onSearch" />
        </div>
      </div>
    </header>

    <div class="pool-page__body">
      <p v-if="error && !assignOpen" class="curriculum-error">{{ error }}</p>
      <p v-if="toast" class="toast">{{ toast }}</p>
      <p v-if="loading" class="curriculum-empty">Загрузка…</p>
      <p v-else-if="!lessons.length" class="curriculum-empty">Занятия не найдены</p>
      <div v-else class="curriculum-folders">
        <section v-for="folder in folders" :key="folder.moduleId" class="curriculum-folder">
          <h2 class="curriculum-folder__title">{{ folder.title }}</h2>
          <div class="curriculum-grid">
            <TbLessonCard
              v-for="lesson in folder.items"
              :key="lesson.id"
              :title="lesson.title"
              :meta="lessonMeta(lesson)"
              :tags="lessonTags(lesson)"
              :progress="lesson.passed_rate"
              action-label="Назначить"
              @action="openAssign(lesson)"
            />
          </div>
        </section>
      </div>
    </div>

    <TbModal v-model:open="assignOpen" title="Назначить занятие в модуль">
      <div class="curriculum-form">
        <p class="page-sub">
          «{{ assignLesson?.title }}» будет скопировано в выбранный модуль.
        </p>
        <TbField label="Модуль">
          <TbSelect
            v-model="targetModuleId"
            :options="moduleOptions"
            placeholder="Выберите модуль"
          />
        </TbField>
        <p v-if="error" class="curriculum-error">{{ error }}</p>
      </div>
      <template #footer>
        <TbButton variant="secondary" @click="assignOpen = false">Отмена</TbButton>
        <TbButton :busy="assignBusy" :disabled="!targetModuleId" @click="assignToModule">
          Назначить
        </TbButton>
      </template>
    </TbModal>
  </section>
</template>

<style scoped>
.pool-page {
  display: flex;
  flex-direction: column;
  gap: 24px;
  width: 100%;
  min-height: 100%;
}

.pool-page__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
}

.pool-page__actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  flex-shrink: 0;
}

.search-wrap {
  width: 280px;
  max-width: 100%;
}

.pool-page__body {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 28px;
  min-height: 0;
}

.toast {
  margin: 0;
  color: var(--color-success);
  font: var(--font-mute);
}

@media (max-width: 1100px) {
  .pool-page__header {
    flex-direction: column;
    align-items: stretch;
  }

  .pool-page__actions {
    width: 100%;
  }

  .search-wrap {
    flex: 1;
    min-width: 0;
    width: auto;
  }
}

@media (max-width: 640px) {
  .pool-page__actions {
    flex-direction: column;
    align-items: stretch;
  }

  .search-wrap {
    width: 100%;
  }

  .pool-page__actions :deep(.tb-btn) {
    width: 100%;
  }
}
</style>
