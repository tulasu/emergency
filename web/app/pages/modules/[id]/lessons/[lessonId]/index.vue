<script setup lang="ts">
import type { Lesson, Module, Ticket, Topic, Variant } from '~/types/curriculum';
import type { TbTabItem } from '~/types/ui';
import { apiErrorMessage } from '~/utils/api-error';
import {
  compareVariantHeader,
  formatSuccessRate,
  variantLabels,
} from '~/utils/curriculum-labels';
import { ruCount } from '~/utils/ru-count';

definePageMeta({
  middleware: ['auth', 'staff'],
});

const route = useRoute();
const moduleId = computed(() => String(route.params.id));
const lessonId = computed(() => String(route.params.lessonId));

const variantsApi = useVariants();
const lessonsApi = useLessons();
const modulesApi = useModules();
const ticketsApi = useTickets();
const { $api } = useNuxtApp();

const tab = ref('variants');
const lesson = ref<Lesson | null>(null);
const module = ref<Module | null>(null);
const moduleTitle = ref('');
const variants = ref<Variant[]>([]);
const ticketsByVariant = ref<Record<string, Ticket[]>>({});
const topicsById = ref<Record<string, string>>({});
const loading = ref(true);
const error = ref('');
const modalOpen = ref(false);
const newTitle = ref('');
const createMode = ref<'empty' | 'copy'>('empty');
const copyFromId = ref('');
const busy = ref(false);
const openModal = ref(false);

const totalAttempts = computed(() =>
  variants.value.reduce((s, v) => s + (v.attempt_count ?? 0), 0),
);

const tabs = computed<TbTabItem[]>(() => [
  {
    id: 'progress',
    label: 'Прохождение',
    count: totalAttempts.value || undefined,
  },
  { id: 'variants', label: 'Варианты и задания', count: variants.value.length },
  { id: 'settings', label: 'Настройки' },
]);

const lessonSubtitle = computed(() => {
  const mins = lesson.value?.duration_seconds
    ? Math.round(lesson.value.duration_seconds / 60)
    : null;
  const threshold = module.value?.success_threshold;
  const parts = [
    mins != null ? `Норматив ${mins} мин` : 'Норматив —',
    threshold != null ? `порог ${threshold}%` : null,
    `варианты ${variantLabels(variants.value, 'short')}`,
  ].filter(Boolean);
  return parts.join(' · ');
});

const lessonNormMinutes = computed(() =>
  lesson.value?.duration_seconds
    ? Math.round(lesson.value.duration_seconds / 60)
    : null,
);

const copyOptions = computed(() =>
  variants.value.map((v) => ({
    value: v.id,
    label: `${v.title}${v.is_primary ? ' ★' : ''}${v.status === 'draft' ? ' (черн.)' : ''}`,
  })),
);

function hardestLabel(variant: Variant): string {
  if (variant.hardest_ticket_title) {
    if (variant.hardest_ticket_rate != null) {
      return `«${variant.hardest_ticket_title} — ${Math.round(variant.hardest_ticket_rate)}%»`;
    }
    return `«${variant.hardest_ticket_title}»`;
  }
  const tickets = ticketsByVariant.value[variant.id] || [];
  if (!tickets.length) {
    return '—';
  }
  return tickets[0]?.title ? `«${tickets[0].title}»` : '—';
}

function topicLabel(topicId: string): string {
  return topicsById.value[topicId] || topicId.slice(0, 8);
}

function topicsForVariant(variant: Variant): { topics: string[]; missing: string[] } {
  const tickets = ticketsByVariant.value[variant.id] || [];
  const coveredIds = [...new Set(tickets.map((t) => t.topic_id))];
  const topics = coveredIds.map(topicLabel);
  const allCatalog = Object.entries(topicsById.value);
  const missing: string[] = [];
  if (variant.status === 'draft' && allCatalog.length) {
    for (const [id, title] of allCatalog) {
      if (!coveredIds.includes(id) && missing.length < 1) {
        missing.push(title);
      }
    }
  }
  return { topics, missing };
}

const compareRows = computed(() => {
  const cols = variants.value;
  return [
    {
      label: 'Суммарная сложность',
      values: cols.map((v) => {
        const n = v.ticket_count ?? ticketsByVariant.value[v.id]?.length;
        return n != null ? String(n * 3) : '—';
      }),
    },
    {
      label: 'Доля успешных ответов',
      values: cols.map((v) =>
        v.avg_success != null ? `${Math.round(v.avg_success)}%` : '—',
      ),
    },
    {
      label: 'Самое трудное задание',
      values: cols.map((v) => hardestLabel(v)),
    },
  ];
});

async function load(): Promise<void> {
  loading.value = true;
  error.value = '';
  try {
    const [list, variantList, mod, topicList] = await Promise.all([
      lessonsApi.listByModule(moduleId.value),
      variantsApi.list(lessonId.value),
      modulesApi.get(moduleId.value),
      $api<Topic[]>('/topics').catch(() => [] as Topic[]),
    ]);
    lesson.value = list.find((item) => item.id === lessonId.value) ?? null;
    variants.value = variantList;
    module.value = mod;
    moduleTitle.value = mod.title;
    topicsById.value = Object.fromEntries(topicList.map((t) => [t.id, t.title]));
    if (!copyFromId.value && variantList[0]) {
      copyFromId.value = variantList.find((v) => v.is_primary)?.id || variantList[0].id;
    }
    const ticketEntries = await Promise.all(
      variantList.map(async (variant) => {
        try {
          const tickets = await ticketsApi.listByVariant(variant.id);
          return [variant.id, tickets] as const;
        } catch {
          return [variant.id, [] as Ticket[]] as const;
        }
      }),
    );
    ticketsByVariant.value = Object.fromEntries(ticketEntries);
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    loading.value = false;
  }
}

function openCreateModal(): void {
  createMode.value = 'empty';
  newTitle.value = `Вариант ${String.fromCharCode(65 + variants.value.length)}`;
  copyFromId.value =
    variants.value.find((v) => v.is_primary)?.id || variants.value[0]?.id || '';
  modalOpen.value = true;
}

async function createVariant(): Promise<void> {
  busy.value = true;
  error.value = '';
  try {
    let created: Variant;
    if (createMode.value === 'copy' && copyFromId.value) {
      created = await variantsApi.clone(
        copyFromId.value,
        newTitle.value.trim() || undefined,
      );
    } else {
      if (!newTitle.value.trim()) {
        return;
      }
      created = await variantsApi.create(lessonId.value, {
        title: newTitle.value.trim(),
        position: variants.value.length,
      });
    }
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
        :subtitle="lessonSubtitle"
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
          <TbButton
            variant="ghost"
            @click="navigateTo(`/modules/${moduleId}/lessons/${lessonId}/review`)"
          >
            Разбор / сетка
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
              :ticket-count="variant.ticket_count ?? ticketsByVariant[variant.id]?.length"
              :attempt-count="variant.attempt_count"
              :avg-success="variant.avg_success"
              :duration-minutes="
                lessonNormMinutes
                  ? Math.max(
                      1,
                      Math.round(
                        (lessonNormMinutes * (variant.ticket_count || ticketsByVariant[variant.id]?.length || 1)) /
                          Math.max(
                            ...variants.map(
                              (v) =>
                                v.ticket_count ||
                                ticketsByVariant[v.id]?.length ||
                                1,
                            ),
                            1,
                          ),
                      ),
                    )
                  : (variant.ticket_count || ticketsByVariant[variant.id]?.length || 0) > 0
                    ? (variant.ticket_count || ticketsByVariant[variant.id]?.length || 0) * 3
                    : null
              "
              :topics="topicsForVariant(variant).topics"
              :missing-topics="topicsForVariant(variant).missing"
              @open="
                navigateTo(
                  `/modules/${moduleId}/lessons/${lessonId}/variants/${variant.id}`,
                )
              "
              @make-primary="makePrimary(variant)"
            />
            <button type="button" class="new-variant" @click="openCreateModal">
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
                    {{ compareVariantHeader(variant) }}
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
          <div v-if="!variants.length" class="curriculum-empty">
            Вариантов пока нет — сводка появится после их создания.
          </div>
          <div v-else class="progress-card">
            <div class="curriculum-table-head progress-head">
              <span>Вариант</span>
              <span>Статус</span>
              <span>Попытки</span>
              <span>Средний результат</span>
            </div>
            <div
              v-for="variant in variants"
              :key="variant.id"
              class="progress-row"
            >
              <div>
                <strong>{{ variant.title }}</strong>
                <p v-if="variant.is_primary" class="page-sub">основной</p>
              </div>
              <div>{{ variant.status === 'approved' ? 'Утверждён' : 'Черновик' }}</div>
              <div>
                {{
                  ruCount(variant.attempt_count ?? 0, 'попытка', 'попытки', 'попыток')
                }}
              </div>
              <div
                :class="{
                  'is-success':
                    variant.avg_success != null && variant.avg_success >= 70,
                }"
              >
                {{ formatSuccessRate(variant.avg_success) }}
              </div>
              <TbButton
                variant="ghost"
                @click="
                  navigateTo(
                    `/modules/${moduleId}/lessons/${lessonId}/review?variant=${variant.id}`,
                  )
                "
              >
                Сетка
              </TbButton>
            </div>
          </div>
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
      <div class="curriculum-form">
        <div class="mode-row">
          <TbChip :active="createMode === 'empty'" @click="createMode = 'empty'">
            Пустой
          </TbChip>
          <TbChip
            :active="createMode === 'copy'"
            :disabled="!variants.length"
            @click="createMode = 'copy'"
          >
            Копия
          </TbChip>
        </div>
        <TbField v-if="createMode === 'copy'" label="Скопировать из">
          <TbSelect
            v-model="copyFromId"
            :options="copyOptions"
            placeholder="Выберите вариант"
          />
        </TbField>
        <TbField label="Название">
          <TbInput v-model="newTitle" placeholder="Вариант A" />
        </TbField>
      </div>
      <template #footer>
        <TbButton variant="secondary" @click="modalOpen = false">Отмена</TbButton>
        <TbButton
          :busy="busy"
          :disabled="createMode === 'copy' ? !copyFromId : !newTitle.trim()"
          @click="createVariant"
        >
          Создать и открыть редактор
        </TbButton>
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

.mode-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.progress-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 8px 0;
}

.progress-head,
.progress-row {
  display: grid;
  grid-template-columns: minmax(180px, 1.4fr) 140px 160px 140px;
  gap: 16px;
  align-items: center;
}

.progress-row {
  padding: 14px 16px;
  border-radius: var(--radius-sm);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
}

.progress-row strong {
  font: var(--font-block);
}

.progress-row .is-success {
  color: var(--color-success);
  font-weight: 600;
}

@media (max-width: 900px) {
  .progress-head,
  .progress-row {
    grid-template-columns: 1fr 1fr;
  }
}
</style>
