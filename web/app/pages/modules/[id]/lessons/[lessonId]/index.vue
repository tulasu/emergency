<script setup lang="ts">
import type { Lesson, Variant } from '~/types/curriculum';
import type { TbTabItem } from '~/types/ui';
import { apiErrorMessage } from '~/utils/api-error';
import { variantLabels } from '~/utils/curriculum-labels';

definePageMeta({
  middleware: ['auth', 'staff'],
});

const route = useRoute();
const moduleId = computed(() => String(route.params.id));
const lessonId = computed(() => String(route.params.lessonId));

const variantsApi = useVariants();
const lessonsApi = useLessons();
const modulesApi = useModules();

const tab = ref('variants');
const lesson = ref<Lesson | null>(null);
const moduleTitle = ref('');
const variants = ref<Variant[]>([]);
const loading = ref(true);
const error = ref('');
const modalOpen = ref(false);
const newTitle = ref('');
const busy = ref(false);
const openModal = ref(false);

const tabs = computed<TbTabItem[]>(() => [
  { id: 'progress', label: 'Прохождение', count: variants.value.reduce((s, v) => s + (v.attempt_count ?? 0), 0) || undefined },
  { id: 'variants', label: 'Варианты и задания', count: variants.value.length },
  { id: 'settings', label: 'Настройки' },
]);

const lessonNormMinutes = computed(() =>
  lesson.value?.duration_seconds
    ? Math.round(lesson.value.duration_seconds / 60)
    : null,
);

const compareRows = computed(() => {
  const cols = variants.value;
  return [
    {
      label: 'Суммарная сложность',
      values: cols.map((v) =>
        v.ticket_count != null ? String((v.ticket_count || 0) * 3) : '—',
      ),
    },
    {
      label: 'Доля успешных ответов',
      values: cols.map((v) =>
        v.avg_success != null ? `${Math.round(v.avg_success)}%` : '—',
      ),
    },
    {
      label: 'Самое трудное задание',
      values: cols.map(() => '—'),
    },
  ];
});

async function load(): Promise<void> {
  loading.value = true;
  error.value = '';
  try {
    const [list, variantList, mod] = await Promise.all([
      lessonsApi.listByModule(moduleId.value),
      variantsApi.list(lessonId.value),
      modulesApi.get(moduleId.value),
    ]);
    lesson.value = list.find((item) => item.id === lessonId.value) ?? null;
    variants.value = variantList;
    moduleTitle.value = mod.title;
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    loading.value = false;
  }
}

async function createVariant(): Promise<void> {
  if (!newTitle.value.trim()) {
    return;
  }
  busy.value = true;
  error.value = '';
  try {
    const created = await variantsApi.create(lessonId.value, {
      title: newTitle.value.trim(),
      position: variants.value.length,
    });
    newTitle.value = '';
    modalOpen.value = false;
    await navigateTo(
      `/modules/${moduleId.value}/lessons/${lessonId.value}/variants/${created.id}`,
    );
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    busy.value = false;
  }
}

async function makePrimary(variant: Variant): Promise<void> {
  try {
    await variantsApi.update(variant.id, {
      title: variant.title,
      position: variant.position,
      status: variant.status,
      is_primary: true,
    });
    await load();
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
        :title="lesson?.title || 'Занятие'"
        :subtitle="`Норматив ${
          lesson?.duration_seconds ? Math.round(lesson.duration_seconds / 60) + ' мин' : '—'
        } · варианты ${variantLabels(variants)}`"
        :breadcrumbs="[
          { label: 'Модули', to: '/modules' },
          { label: moduleTitle || 'Модуль', to: `/modules/${moduleId}` },
          { label: lesson?.title || 'Занятие' },
        ]"
      >
        <template #actions>
          <TbButton
            variant="secondary"
            @click="navigateTo(`/modules/${moduleId}/edit`)"
          >
            Изменить занятие
          </TbButton>
          <TbButton @click="openModal = true">Открыть занятие</TbButton>
        </template>
      </CurriculumPageHeader>
      <div class="sheet__toolbar">
        <TbTabs v-model="tab" :tabs="tabs" />
      </div>
      <div class="sheet__rule"></div>
      <div class="sheet__body">
        <p v-if="error" class="curriculum-error">{{ error }}</p>
        <p v-if="loading" class="curriculum-empty">Загрузка…</p>

        <template v-else-if="tab === 'variants'">
          <div class="variants-row">
            <VariantOverviewCard
              v-for="variant in variants"
              :key="variant.id"
              :variant="variant"
              :ticket-count="variant.ticket_count"
              :attempt-count="variant.attempt_count"
              :avg-success="variant.avg_success"
              :duration-minutes="
                lessonNormMinutes
                  ? Math.max(
                      1,
                      Math.round(
                        (lessonNormMinutes * (variant.ticket_count || 1)) /
                          Math.max(
                            ...variants.map((v) => v.ticket_count || 1),
                            1,
                          ),
                      ),
                    )
                  : variant.ticket_count
                    ? (variant.ticket_count || 0) * 3
                    : null
              "
              :topics="
                variant.status === 'draft' && (variant.ticket_count || 0) < 8
                  ? ['Тип', 'Признаки']
                  : ['Тип', 'Признаки', 'Описание']
              "
              :missing-topics="
                variant.status === 'draft' && (variant.ticket_count || 0) < 8
                  ? ['Описание']
                  : []
              "
              @open="
                navigateTo(
                  `/modules/${moduleId}/lessons/${lessonId}/variants/${variant.id}`,
                )
              "
              @make-primary="makePrimary(variant)"
            />
            <button type="button" class="new-variant" @click="modalOpen = true">
              <TbIcon name="plus" />
              <strong>Новый вариант</strong>
              <span>пустой или копия</span>
            </button>
          </div>

          <div v-if="variants.length" class="curriculum-compare">
            <div class="curriculum-compare__head">
              <h3>Сравнение вариантов по заданиям</h3>
              <p>Попытки на разных вариантах сравнимы, если сложность близка</p>
            </div>
            <table>
              <thead>
                <tr>
                  <th>Показатель</th>
                  <th v-for="variant in variants" :key="variant.id">
                    {{
                      variant.title.replace(/^Вариант\s+/i, '').trim() ||
                      variant.title
                    }}{{ variant.is_primary ? ' ★' : '' }}
                  </th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="row in compareRows" :key="row.label">
                  <td>{{ row.label }}</td>
                  <td v-for="(value, index) in row.values" :key="`${row.label}-${index}`">
                    {{ value }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </template>

        <template v-else-if="tab === 'progress'">
          <p class="page-sub">
            Сводка прохождения появится после выдачи попыток. Сейчас вариантов:
            {{ variants.length }}.
          </p>
        </template>

        <template v-else>
          <div class="curriculum-form">
            <TbField label="Название занятия">
              <TbInput :model-value="lesson?.title || ''" disabled />
            </TbField>
            <p class="page-sub">
              Полное редактирование названия и архив — на странице изменения модуля.
            </p>
            <TbButton variant="secondary" @click="navigateTo(`/modules/${moduleId}/edit`)">
              К изменению модуля
            </TbButton>
          </div>
        </template>
      </div>
    </div>

    <TbModal v-model:open="modalOpen" title="Новый вариант">
      <TbField label="Название">
        <TbInput v-model="newTitle" placeholder="Вариант A" />
      </TbField>
      <template #footer>
        <TbButton variant="secondary" @click="modalOpen = false">Отмена</TbButton>
        <TbButton :busy="busy" @click="createVariant">Создать и открыть редактор</TbButton>
      </template>
    </TbModal>

    <OpenLessonModal
      v-model:open="openModal"
      :module-id="moduleId"
      :module-title="moduleTitle"
      :lessons="lesson ? [lesson] : []"
      :variants="variants"
    />
  </section>
</template>

<style scoped>
.variants-row {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr)) minmax(200px, 240px);
  gap: 16px;
  align-items: stretch;
}

.new-variant {
  min-height: 260px;
  width: 100%;
  max-width: none;
  padding: 20px;
  border: 1.5px dashed var(--color-border-strong);
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--color-primary);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  cursor: pointer;
  font: inherit;
}

@media (max-width: 1100px) {
  .variants-row {
    grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  }
}

.new-variant strong {
  font: var(--font-block);
  color: var(--color-primary);
}

.new-variant span {
  color: var(--color-text-muted);
  font: var(--font-cap);
}
</style>
