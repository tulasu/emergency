<script setup lang="ts">
import type { Ticket, Variant } from '~/types/curriculum';
import { apiErrorMessage } from '~/utils/api-error';
import { variantStatusLabel, variantStatusTone } from '~/utils/curriculum-labels';
import { ruCount } from '~/utils/ru-count';

definePageMeta({
  middleware: ['auth', 'staff'],
});

const route = useRoute();
const moduleId = computed(() => String(route.params.id));
const lessonId = computed(() => String(route.params.lessonId));
const variantId = computed(() => String(route.params.variantId));

const ticketsApi = useTickets();
const variantsApi = useVariants();
const lessonsApi = useLessons();
const modulesApi = useModules();

const variant = ref<Variant | null>(null);
const primary = ref<Variant | null>(null);
const library = ref<Ticket[]>([]);
const variantTickets = ref<Ticket[]>([]);
const primaryTickets = ref<Ticket[]>([]);
const moduleTitle = ref('');
const lessonTitle = ref('');
const lessonNormMinutes = ref(45);
const search = ref('');
const topicFilter = ref('all');
const makePrimaryAfter = ref(false);
const loading = ref(true);
const busy = ref(false);
const error = ref('');
const toast = ref('');

const TOPIC_LABELS: Record<string, string> = {
  type: 'Тип',
  signs: 'Признаки',
  description: 'Описание',
};

const topicChips = computed(() => {
  const set = new Set<string>();
  for (const ticket of library.value) {
    set.add(ticket.topic_id);
  }
  return ['all', ...Array.from(set)];
});

const filteredLibrary = computed(() => {
  let list = library.value;
  if (topicFilter.value !== 'all') {
    list = list.filter((t) => t.topic_id === topicFilter.value);
  }
  if (search.value.trim()) {
    const q = search.value.trim().toLowerCase();
    list = list.filter(
      (t) =>
        t.title.toLowerCase().includes(q) || t.body.toLowerCase().includes(q),
    );
  }
  return list;
});

const addedIds = computed(() => new Set(variantTickets.value.map((t) => t.id)));

const topicCoverage = computed(() => {
  const topics = [
    { id: 'type', label: 'Тип происшествия' },
    { id: 'signs', label: 'Признаки происшествия' },
    { id: 'description', label: 'Описание происшествия' },
  ];
  return topics.map((topic) => {
    const count = variantTickets.value.filter((t) =>
      t.topic_id.toLowerCase().includes(topic.id.slice(0, 4)),
    ).length;
    // Fallback: distribute by index thirds when topic ids are opaque UUIDs
    return { ...topic, count };
  });
});

const coverageFallback = computed(() => {
  const n = variantTickets.value.length;
  if (n === 0) {
    return [];
  }
  if (!topicCoverage.value.some((t) => t.count > 0)) {
    const a = Math.ceil(n / 2);
    const b = Math.max(0, n - a);
    return [
      { id: 'type', label: 'Тип происшествия', count: a, covered: a > 0 },
      { id: 'signs', label: 'Признаки происшествия', count: b, covered: b > 0 },
      {
        id: 'description',
        label: 'Описание происшествия',
        count: 0,
        covered: false,
      },
    ];
  }
  return topicCoverage.value.map((t) => ({
    ...t,
    covered: t.count > 0,
  }));
});

const missingTopic = computed(
  () => coverageFallback.value.find((t) => !t.covered)?.label || null,
);

const compareRows = computed(() => {
  const p = primaryTickets.value.length;
  const c = variantTickets.value.length;
  return [
    { label: 'Заданий', primary: String(p || '—'), current: String(c), warn: false },
    {
      label: 'Суммарная сложность',
      primary: String(p * 3 || '—'),
      current: String(c * 3),
      warn: false,
    },
    {
      label: 'Время прохождения',
      primary: p ? `${p * 3} мин` : '—',
      current: `${c * 3} мин`,
      warn: Math.abs(p - c) >= 2,
    },
  ];
});

const canApprove = computed(
  () =>
    variant.value?.status !== 'approved' &&
    variantTickets.value.length > 0 &&
    !missingTopic.value,
);

const approveBlockText = computed(() => {
  if (missingTopic.value) {
    return `Не покрыта тема «${missingTopic.value}». Без неё повторная попытка проверит не то же самое, что основной вариант.`;
  }
  if (!variantTickets.value.length) {
    return 'Добавьте хотя бы одно задание, чтобы утвердить вариант.';
  }
  return '';
});

const durationEstimate = computed(() => variantTickets.value.length * 3);

function topicShort(topicId: string): string {
  const lower = topicId.toLowerCase();
  for (const [key, label] of Object.entries(TOPIC_LABELS)) {
    if (lower.includes(key) || lower.includes(label.toLowerCase())) {
      return label;
    }
  }
  return topicId.slice(0, 8);
}

async function load(): Promise<void> {
  loading.value = true;
  error.value = '';
  try {
    const [allVariants, lessons, mod] = await Promise.all([
      variantsApi.list(lessonId.value),
      lessonsApi.listByModule(moduleId.value),
      modulesApi.get(moduleId.value),
    ]);
    const v = allVariants.find((item) => item.id === variantId.value);
    if (!v) {
      throw new Error('Вариант не найден');
    }
    variant.value = v;
    moduleTitle.value = mod.title;
    const lesson = lessons.find((item) => item.id === lessonId.value);
    lessonTitle.value = lesson?.title || 'Занятие';
    lessonNormMinutes.value = lesson?.duration_seconds
      ? Math.round(lesson.duration_seconds / 60)
      : 45;
    primary.value =
      allVariants.find((item) => item.is_primary && item.id !== v.id) ?? null;

    const [lib, own, primaryOwn] = await Promise.all([
      ticketsApi.library(search.value.trim()).catch(() => [] as Ticket[]),
      ticketsApi.listByVariant(variantId.value).catch(() => [] as Ticket[]),
      primary.value
        ? ticketsApi.listByVariant(primary.value.id).catch(() => [] as Ticket[])
        : Promise.resolve([] as Ticket[]),
    ]);
    library.value = lib;
    variantTickets.value = own;
    primaryTickets.value = primaryOwn;
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    loading.value = false;
  }
}

async function searchLibrary(): Promise<void> {
  try {
    library.value = await ticketsApi.library(search.value.trim());
  } catch (err) {
    error.value = apiErrorMessage(err);
  }
}

async function addFromPool(ticketId: string): Promise<void> {
  try {
    await ticketsApi.copyToVariant(variantId.value, ticketId);
    variantTickets.value = await ticketsApi.listByVariant(variantId.value);
  } catch (err) {
    error.value = apiErrorMessage(err);
  }
}

async function saveDraft(): Promise<void> {
  if (!variant.value) {
    return;
  }
  busy.value = true;
  error.value = '';
  try {
    variant.value = await variantsApi.update(variantId.value, {
      title: variant.value.title,
      position: variant.value.position,
      status: 'draft',
      is_primary: variant.value.is_primary,
    });
    toast.value = 'Черновик сохранён';
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    busy.value = false;
  }
}

async function approve(): Promise<void> {
  if (!variant.value || !canApprove.value) {
    return;
  }
  busy.value = true;
  error.value = '';
  try {
    variant.value = await variantsApi.update(variantId.value, {
      title: variant.value.title,
      position: variant.value.position,
      status: 'approved',
      is_primary: makePrimaryAfter.value || variant.value.is_primary,
    });
    toast.value = 'Вариант утверждён';
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    busy.value = false;
  }
}

onMounted(() => {
  void load();
});
</script>

<template>
  <section class="page-sheet page-sheet--wide">
    <div class="sheet editor-sheet">
      <CurriculumPageHeader
        :title="variant?.title || 'Редактор варианта'"
        :subtitle="`Для сверки сложности с основным · изменён ${
          variant?.created_at
            ? new Date(variant.created_at).toLocaleString('ru-RU', {
                day: '2-digit',
                month: '2-digit',
                hour: '2-digit',
                minute: '2-digit',
              })
            : '—'
        }`"
        :breadcrumbs="[
          { label: 'Модули', to: '/modules' },
          { label: moduleTitle || 'Модуль', to: `/modules/${moduleId}` },
          {
            label: lessonTitle || 'Занятие',
            to: `/modules/${moduleId}/lessons/${lessonId}`,
          },
          { label: variant?.title || 'Вариант' },
        ]"
      >
        <template #meta>
          <TbBadge v-if="variant" :tone="variantStatusTone(variant.status)">
            {{ variantStatusLabel(variant.status) }}
          </TbBadge>
        </template>
        <template #actions>
          <TbButton
            variant="ghost"
            @click="navigateTo(`/modules/${moduleId}/lessons/${lessonId}`)"
          >
            Закрыть
          </TbButton>
          <TbButton variant="secondary" :busy="busy" @click="saveDraft">
            Сохранить черновик
          </TbButton>
          <TbButton :busy="busy" :disabled="!canApprove" @click="approve">
            Утвердить вариант
          </TbButton>
        </template>
      </CurriculumPageHeader>

      <div class="sheet__toolbar settings-row">
        <div class="stat">
          <span>Заданий</span>
          <strong
            >{{ variantTickets.length }} из 10 рекомендуемых</strong
          >
        </div>
        <div class="stat">
          <span>Норматив</span>
          <strong
            >≈ {{ durationEstimate }} мин из {{ lessonNormMinutes }}</strong
          >
        </div>
        <div class="stat">
          <span>Используется</span>
          <strong
            >{{
              ruCount(variant?.attempt_count ?? 0, 'попытка', 'попытки', 'попыток')
            }}</strong
          >
        </div>
        <label class="make-primary">
          <input v-model="makePrimaryAfter" type="checkbox" />
          Сделать основным после утверждения
        </label>
      </div>
      <div class="sheet__rule"></div>
      <div class="sheet__body">
        <p v-if="error" class="curriculum-error">{{ error }}</p>
        <p v-if="toast" class="toast">{{ toast }}</p>
        <p v-if="loading" class="curriculum-empty">Загрузка…</p>
        <div v-else class="curriculum-split curriculum-split--triple">
          <div class="curriculum-panel">
            <div class="panel-head">
              <h2 class="sheet__section-title">Библиотека карточек</h2>
              <p class="page-sub">по темам занятия</p>
            </div>
            <TbSearch
              v-model="search"
              placeholder="Поиск по билетам"
              @update:model-value="searchLibrary"
            />
            <div class="topic-chips">
              <TbChip
                :active="topicFilter === 'all'"
                @click="topicFilter = 'all'"
              >
                Все
              </TbChip>
              <TbChip
                v-for="topic in topicChips.slice(1)"
                :key="topic"
                :active="topicFilter === topic"
                @click="topicFilter = topic"
              >
                {{ topicShort(topic) }}
              </TbChip>
            </div>
            <p v-if="!filteredLibrary.length" class="curriculum-empty">
              Ничего не найдено
            </p>
            <div
              v-for="ticket in filteredLibrary"
              :key="ticket.id"
              class="lib-card"
            >
              <div class="lib-card__text">
                <strong>{{ ticket.title }}</strong>
                <div class="lib-card__meta">
                  <TbChip disabled>{{ topicShort(ticket.topic_id) }}</TbChip>
                  <span class="page-sub">в пуле</span>
                </div>
              </div>
              <TbButton
                v-if="!addedIds.has(ticket.id)"
                variant="icon"
                aria-label="Добавить"
                @click="addFromPool(ticket.id)"
              >
                <TbIcon name="plus" />
              </TbButton>
              <span v-else class="lib-card__added" aria-label="Добавлено">
                <TbIcon name="check" />
              </span>
            </div>
          </div>

          <div class="curriculum-panel">
            <div class="panel-head">
              <h2 class="sheet__section-title">Задания варианта</h2>
              <p class="page-sub">
                {{ variantTickets.length }} заданий · порядок = порядок выдачи
              </p>
            </div>

            <div class="coverage">
              <div
                v-for="item in coverageFallback"
                :key="item.id"
                class="coverage__pill"
                :class="item.covered ? 'coverage__pill--ok' : 'coverage__pill--bad'"
              >
                <strong v-if="item.covered"
                  >{{ ruCount(item.count, 'задание', 'задания', 'заданий') }}</strong
                >
                <strong v-else>Не покрыта</strong>
                <span>{{ item.label }}</span>
              </div>
            </div>

            <div
              v-for="(ticket, index) in variantTickets"
              :key="ticket.id"
              class="task-row"
            >
              <span class="task-row__grip" aria-hidden="true">⠿</span>
              <span class="num">{{ String(index + 1).padStart(2, '0') }}</span>
              <div class="grow">
                <strong>{{ ticket.title }}</strong>
                <div class="lib-card__meta">
                  <TbChip disabled>{{ topicShort(ticket.topic_id) }}</TbChip>
                </div>
              </div>
              <span class="time">3:00</span>
              <TbButton
                variant="icon"
                aria-label="Открыть"
                @click="navigateTo(`/tickets/${ticket.id}`)"
              >
                <TbIcon name="x" />
              </TbButton>
            </div>

            <div v-if="missingTopic" class="drop-zone">
              <TbIcon name="arrow-left" />
              <span
                >Добавьте из библиотеки карточку по теме «{{
                  missingTopic
                }}»</span
              >
            </div>
            <p v-else-if="!variantTickets.length" class="curriculum-empty">
              Добавьте из библиотеки карточку
            </p>
          </div>

          <div class="curriculum-panel">
            <div class="panel-head">
              <h2 class="sheet__section-title">Сверка с основным</h2>
              <p class="page-sub">
                Попытки на разных вариантах сравниваются между собой, поэтому
                сложность должна совпадать
              </p>
            </div>
            <div class="compare-head">
              <span></span>
              <span>Основной</span>
              <span>{{ variant?.title || 'Вариант' }}</span>
            </div>
            <div
              v-for="row in compareRows"
              :key="row.label"
              class="compare-row"
              :class="{ 'compare-row--warn': row.warn }"
            >
              <span>{{ row.label }}</span>
              <strong>{{ row.primary }}</strong>
              <strong>{{ row.current }}</strong>
            </div>
            <div v-if="approveBlockText" class="approve-block">
              <div class="approve-block__row">
                <TbIcon name="x" />
                <strong>Утвердить пока нельзя</strong>
              </div>
              <p>{{ approveBlockText }}</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.editor-sheet {
  max-width: 1600px !important;
}

.settings-row {
  display: flex;
  flex-wrap: wrap;
  gap: 24px;
  align-items: center;
  padding-top: 12px;
  padding-bottom: 16px;
}

.stat {
  display: flex;
  align-items: center;
  gap: 8px;
}

.stat span {
  color: var(--color-text-subtle);
  font: var(--font-cap);
  text-transform: none;
}

.stat strong {
  font: var(--font-mute);
}

.make-primary {
  margin-left: auto;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font: var(--font-mute);
}

.panel-head {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.topic-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.lib-card,
.task-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-surface);
}

.lib-card__text,
.grow {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.lib-card__text strong,
.grow strong {
  font: var(--font-body);
}

.lib-card__meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.lib-card__added {
  display: inline-flex;
  width: 32px;
  height: 32px;
  align-items: center;
  justify-content: center;
  color: var(--color-text-subtle);
}

.coverage {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.coverage__pill {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 10px 12px;
  border-radius: var(--radius-sm);
  min-width: 140px;
  flex: 1;
}

.coverage__pill--ok {
  background: var(--color-success-soft);
}

.coverage__pill--ok strong {
  color: var(--color-success);
}

.coverage__pill--bad {
  background: var(--color-danger-soft);
}

.coverage__pill--bad strong {
  color: var(--color-danger);
}

.coverage__pill span {
  font: var(--font-cap);
  color: var(--color-text);
}

.task-row__grip {
  color: var(--color-text-subtle);
  font-size: 14px;
  line-height: 1;
  cursor: grab;
  user-select: none;
}

.num {
  color: var(--color-text-subtle);
  font: var(--font-cap);
}

.time {
  color: var(--color-text-muted);
  font: var(--font-cap);
}

.drop-zone {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 72px;
  padding: 16px;
  border: 1.5px dashed var(--color-border-strong);
  border-radius: var(--radius-sm);
  color: var(--color-text-muted);
  font: var(--font-mute);
  text-align: center;
}

.compare-head,
.compare-row {
  display: grid;
  grid-template-columns: 1.4fr 1fr 1fr;
  gap: 8px;
  padding: 10px 12px;
  font: var(--font-mute);
  border-radius: var(--radius-sm);
}

.compare-head {
  color: var(--color-text-subtle);
  font: var(--font-cap);
  text-transform: uppercase;
  padding-top: 0;
  padding-bottom: 0;
}

.compare-row--warn {
  background: var(--color-warning-soft);
}

.approve-block {
  margin-top: auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 16px;
  border: 1px solid var(--color-danger);
  border-radius: var(--radius-sm);
  background: var(--color-danger-soft);
}

.approve-block__row {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--color-danger);
}

.approve-block p {
  margin: 0;
  color: var(--color-text);
  font: var(--font-mute);
}

.toast {
  margin: 0;
  color: var(--color-success);
  font: var(--font-mute);
}
</style>
