<script setup lang="ts">
import type { Lesson, Module, ModuleStatus } from '~/types/curriculum';
import { apiErrorMessage } from '~/utils/api-error';

definePageMeta({
  middleware: ['auth', 'staff'],
});

const route = useRoute();
const moduleId = computed(() => String(route.params.id));

const modulesApi = useModules();
const lessonsApi = useLessons();

const title = ref('');
const description = ref('');
const status = ref('draft');
const threshold = ref(70);
const lessons = ref<Lesson[]>([]);
const newLessonTitle = ref('');
const busy = ref(false);
const error = ref('');
const loading = ref(true);

const statusOptions = [
  { value: 'draft', label: 'Черновик' },
  { value: 'active', label: 'Активен' },
  { value: 'archived', label: 'Архив' },
];

async function load(): Promise<void> {
  loading.value = true;
  error.value = '';
  try {
    const [mod, list] = await Promise.all([
      modulesApi.get(moduleId.value),
      lessonsApi.listByModule(moduleId.value),
    ]);
    applyModule(mod);
    lessons.value = list;
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    loading.value = false;
  }
}

function applyModule(mod: Module): void {
  title.value = mod.title;
  description.value = mod.description;
  status.value = mod.status;
  threshold.value = mod.success_threshold;
}

async function save(): Promise<void> {
  busy.value = true;
  error.value = '';
  try {
    const mod = await modulesApi.update(moduleId.value, {
      title: title.value.trim(),
      description: description.value.trim(),
      status: status.value as ModuleStatus,
      success_threshold: threshold.value,
    });
    applyModule(mod);
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    busy.value = false;
  }
}

async function addLesson(): Promise<void> {
  if (!newLessonTitle.value.trim()) {
    return;
  }
  try {
    await lessonsApi.create(moduleId.value, {
      title: newLessonTitle.value.trim(),
      position: lessons.value.length,
    });
    newLessonTitle.value = '';
    lessons.value = await lessonsApi.listByModule(moduleId.value);
  } catch (err) {
    error.value = apiErrorMessage(err);
  }
}

async function archiveLesson(lessonId: string): Promise<void> {
  try {
    await lessonsApi.archive(lessonId);
    lessons.value = await lessonsApi.listByModule(moduleId.value);
  } catch (err) {
    error.value = apiErrorMessage(err);
  }
}

onMounted(() => {
  void load();
});
</script>

<template>
  <section class="page-sheet page-sheet--wide">
    <div class="sheet">
      <CurriculumPageHeader
        title="Изменение модуля"
        :breadcrumbs="[
          { label: 'Модули', to: '/modules' },
          { label: title || 'Модуль', to: `/modules/${moduleId}` },
          { label: 'Изменение' },
        ]"
      >
        <template #actions>
          <TbButton variant="secondary" @click="navigateTo(`/modules/${moduleId}`)">
            К модулю
          </TbButton>
          <TbButton :busy="busy" @click="save">Сохранить изменения</TbButton>
        </template>
      </CurriculumPageHeader>
      <div class="sheet__rule"></div>
      <div class="sheet__body">
        <p v-if="loading" class="curriculum-empty">Загрузка…</p>
        <template v-else>
          <div class="curriculum-form">
            <TbField label="Название">
              <TbInput v-model="title" />
            </TbField>
            <TbField label="Описание">
              <TbTextarea v-model="description" :rows="4" :maxlength="1000" show-counter />
            </TbField>
            <TbField label="Статус">
              <TbSelect v-model="status" :options="statusOptions" />
            </TbField>
            <TbField label="Порог успеха, %">
              <TbInput
                :model-value="String(threshold)"
                type="number"
                @update:model-value="threshold = Number($event) || 0"
              />
            </TbField>
          </div>

          <div class="lessons">
            <h2 class="sheet__section-title">Занятия</h2>
            <div class="add-row">
              <TbInput v-model="newLessonTitle" placeholder="Новое занятие" />
              <TbButton variant="secondary" @click="addLesson">Добавить</TbButton>
            </div>
            <p v-if="!lessons.length" class="curriculum-empty">Нет занятий</p>
            <div v-for="lesson in lessons" :key="lesson.id" class="curriculum-list-item">
              <div>
                <strong>{{ lesson.title }}</strong>
                <p class="page-sub">Позиция {{ lesson.position }}</p>
              </div>
              <div class="curriculum-actions">
                <TbButton
                  variant="secondary"
                  @click="navigateTo(`/modules/${moduleId}/lessons/${lesson.id}`)"
                >
                  Открыть
                </TbButton>
                <TbButton
                  variant="icon"
                  aria-label="Убрать в архив"
                  @click="archiveLesson(lesson.id)"
                >
                  <TbIcon name="archive" />
                </TbButton>
              </div>
            </div>
          </div>
        </template>
        <p v-if="error" class="curriculum-error">{{ error }}</p>
      </div>
    </div>
  </section>
</template>

<style scoped>
.lessons {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.add-row {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 12px;
}
</style>
