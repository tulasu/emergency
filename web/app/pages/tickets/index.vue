<script setup lang="ts">
import type { Ticket, Topic } from '~/types/curriculum';
import { apiErrorMessage } from '~/utils/api-error';
import { ruCount } from '~/utils/ru-count';

definePageMeta({
  middleware: ['auth', 'staff'],
});

type TicketStatusFilter = 'all' | 'ready' | 'pending' | 'draft';

const PAGE_SIZE = 12;

const ticketsApi = useTickets();
const { $api } = useNuxtApp();

const query = ref('');
const topicSearch = ref('');
const tickets = ref<Ticket[]>([]);
const topics = ref<Topic[]>([]);
const loading = ref(true);
const error = ref('');
const createOpen = ref(false);
const busy = ref(false);
const selectedId = ref('');
const selectedTopicIds = ref<string[]>([]);
const selectedIncidentTypes = ref<string[]>([]);
const statusFilter = ref<TicketStatusFilter>('all');
const unusedOnly = ref(false);
const visibleCount = ref(PAGE_SIZE);
const form = reactive({
  title: '',
  body: '',
  topic_id: '',
});

let timer: ReturnType<typeof setTimeout> | null = null;

const topicName = computed(() => {
  const map = new Map(topics.value.map((t) => [t.id, t.title]));
  return (id: string) => map.get(id) || 'Без темы';
});

const topicCounts = computed(() => {
  const counts = new Map<string, number>();
  for (const ticket of tickets.value) {
    counts.set(ticket.topic_id, (counts.get(ticket.topic_id) || 0) + 1);
  }
  return counts;
});

const filteredTopics = computed(() => {
  const q = topicSearch.value.trim().toLowerCase();
  const list = topics.value
    .map((topic) => ({
      ...topic,
      count: topicCounts.value.get(topic.id) || 0,
    }))
    .sort((a, b) => b.count - a.count || a.title.localeCompare(b.title, 'ru'));
  if (!q) {
    return list;
  }
  return list.filter((topic) => topic.title.toLowerCase().includes(q));
});

function ticketStatusKey(ticket: Ticket): TicketStatusFilter {
  const status = ticket.card_status || ticket.audio_status;
  switch (status) {
    case 'ready':
      return 'ready';
    case 'pending':
      return 'pending';
    default:
      return 'draft';
  }
}

function ticketStatusLabel(ticket: Ticket): string {
  switch (ticketStatusKey(ticket)) {
    case 'ready':
      return 'Готова';
    case 'pending':
      return 'В работе';
    default:
      return 'Черновик';
  }
}

function ticketStatusTone(ticket: Ticket): 'success' | 'info' | 'warning' {
  switch (ticketStatusKey(ticket)) {
    case 'ready':
      return 'success';
    case 'pending':
      return 'info';
    default:
      return 'warning';
  }
}

function ticketTypeLabel(ticket: Ticket): string {
  if (ticket.incident_type) {
    return ticket.incident_type;
  }
  if (ticket.incident_type_code) {
    return ticket.incident_type_code;
  }
  return '—';
}

function ticketSlotsLabel(ticket: Ticket): string {
  if (ticket.slots_total == null) {
    return '—';
  }
  const total = ticket.slots_total;
  const required = ticket.slots_required ?? total;
  return `${total} · ${required} обяз.`;
}

function ticketUsageLabel(ticket: Ticket): string {
  const n = ticket.variant_usage ?? (ticket.variant_id ? 1 : 0);
  if (!n) {
    return 'не используется';
  }
  return String(n);
}

function ticketSubtitle(ticket: Ticket): string {
  const parts = [topicName.value(ticket.topic_id)];
  if (ticket.incident_type_code) {
    parts.push(`вызов ${ticket.incident_type_code}`);
  }
  return parts.join(' · ');
}

const incidentTypeOptions = computed(() => {
  const map = new Map<string, string>();
  for (const t of tickets.value) {
    const code = t.incident_type_code;
    if (!code) {
      continue;
    }
    map.set(code, t.incident_type || code);
  }
  return Array.from(map.entries())
    .map(([code, label]) => ({ code, label }))
    .sort((a, b) => a.label.localeCompare(b.label, 'ru'));
});

const filteredTickets = computed(() => {
  let list = tickets.value;
  if (selectedTopicIds.value.length) {
    const set = new Set(selectedTopicIds.value);
    list = list.filter((t) => set.has(t.topic_id));
  }
  if (selectedIncidentTypes.value.length) {
    const set = new Set(selectedIncidentTypes.value);
    list = list.filter((t) => t.incident_type_code && set.has(t.incident_type_code));
  }
  if (statusFilter.value !== 'all') {
    list = list.filter((t) => ticketStatusKey(t) === statusFilter.value);
  }
  if (unusedOnly.value) {
    list = list.filter((t) => !(t.variant_usage ?? (t.variant_id ? 1 : 0)));
  }
  return [...list].sort(
    (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime(),
  );
});

const visibleTickets = computed(() => filteredTickets.value.slice(0, visibleCount.value));

const remainingCount = computed(() =>
  Math.max(0, filteredTickets.value.length - visibleTickets.value.length),
);

const activeFilters = computed(() => {
  const chips: Array<{ id: string; label: string; clear: () => void }> = [];
  for (const id of selectedTopicIds.value) {
    chips.push({
      id: `topic-${id}`,
      label: topicName.value(id),
      clear: () => {
        selectedTopicIds.value = selectedTopicIds.value.filter((x) => x !== id);
      },
    });
  }
  for (const code of selectedIncidentTypes.value) {
    const opt = incidentTypeOptions.value.find((o) => o.code === code);
    chips.push({
      id: `type-${code}`,
      label: opt?.label || code,
      clear: () => {
        selectedIncidentTypes.value = selectedIncidentTypes.value.filter((x) => x !== code);
      },
    });
  }
  if (statusFilter.value !== 'all') {
    const labels: Record<TicketStatusFilter, string> = {
      all: '',
      ready: 'Готова',
      pending: 'В работе',
      draft: 'Черновик',
    };
    chips.push({
      id: `status-${statusFilter.value}`,
      label: labels[statusFilter.value],
      clear: () => {
        statusFilter.value = 'all';
      },
    });
  }
  if (unusedOnly.value) {
    chips.push({
      id: 'unused',
      label: 'Не в вариантах',
      clear: () => {
        unusedOnly.value = false;
      },
    });
  }
  return chips;
});

function toggleIncidentType(code: string): void {
  if (selectedIncidentTypes.value.includes(code)) {
    selectedIncidentTypes.value = selectedIncidentTypes.value.filter((x) => x !== code);
  } else {
    selectedIncidentTypes.value = [...selectedIncidentTypes.value, code];
  }
  visibleCount.value = PAGE_SIZE;
}

const topicOptions = computed(() =>
  topics.value.map((t) => ({ value: t.id, label: t.title })),
);

function formatChanged(iso: string): string {
  if (!iso) {
    return '—';
  }
  return new Date(iso).toLocaleDateString('ru-RU', {
    day: '2-digit',
    month: '2-digit',
  });
}

function toggleTopic(id: string): void {
  if (selectedTopicIds.value.includes(id)) {
    selectedTopicIds.value = selectedTopicIds.value.filter((x) => x !== id);
  } else {
    selectedTopicIds.value = [...selectedTopicIds.value, id];
  }
  visibleCount.value = PAGE_SIZE;
}

function resetFilters(): void {
  selectedTopicIds.value = [];
  selectedIncidentTypes.value = [];
  statusFilter.value = 'all';
  unusedOnly.value = false;
  visibleCount.value = PAGE_SIZE;
}

async function load(): Promise<void> {
  loading.value = true;
  error.value = '';
  try {
    const [list, topicList] = await Promise.all([
      ticketsApi.library(query.value.trim()),
      topics.value.length ? Promise.resolve(topics.value) : $api<Topic[]>('/topics'),
    ]);
    tickets.value = list;
    topics.value = topicList;
    if (!selectedId.value && list[0]) {
      selectedId.value = list[0].id;
    }
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
    visibleCount.value = PAGE_SIZE;
    void load();
  }, 250);
}

async function openCreate(): Promise<void> {
  createOpen.value = true;
  error.value = '';
  if (!topics.value.length) {
    try {
      topics.value = await $api<Topic[]>('/topics');
      if (!form.topic_id && topics.value[0]) {
        form.topic_id = topics.value[0].id;
      }
    } catch (err) {
      error.value = apiErrorMessage(err);
    }
  } else if (!form.topic_id && topics.value[0]) {
    form.topic_id = topics.value[0].id;
  }
}

async function create(): Promise<void> {
  if (!form.title.trim() || !form.topic_id) {
    error.value = 'Укажите тему и название';
    return;
  }
  busy.value = true;
  error.value = '';
  try {
    const ticket = await ticketsApi.createLibrary({
      topic_id: form.topic_id,
      title: form.title.trim(),
      body: form.body.trim(),
    });
    createOpen.value = false;
    form.title = '';
    form.body = '';
    await navigateTo(`/tickets/${ticket.id}`);
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    busy.value = false;
  }
}

function openTicket(ticket: Ticket): void {
  selectedId.value = ticket.id;
  void navigateTo(`/tickets/${ticket.id}`);
}

onMounted(() => {
  void load();
});
</script>

<template>
  <section class="pool-page">
    <header class="pool-page__header">
      <h1 class="page-title">Пул карточек</h1>
      <div class="pool-page__actions">
        <div class="search-wrap">
          <TbSearch v-model="query" placeholder="Поиск" @update:model-value="onSearch" />
        </div>
        <TbButton @click="openCreate">+ Новая карточка</TbButton>
      </div>
    </header>

    <div class="pool-layout">
      <aside class="pool-filters">
        <div class="pool-filters__section">
          <div class="pool-filters__head">
            <h2>Темы</h2>
            <span class="pool-filters__count">{{ topics.length }}</span>
          </div>
          <TbSearch v-model="topicSearch" placeholder="Найти тему" />
          <button
            type="button"
            class="topic-row"
            :class="{ 'topic-row--active': !selectedTopicIds.length }"
            @click="selectedTopicIds = []"
          >
            <span>Все темы</span>
            <strong>{{ tickets.length }}</strong>
          </button>
          <button
            v-for="topic in filteredTopics"
            :key="topic.id"
            type="button"
            class="topic-row"
            :class="{ 'topic-row--active': selectedTopicIds.includes(topic.id) }"
            @click="toggleTopic(topic.id)"
          >
            <span>{{ topic.title }}</span>
            <strong>{{ topic.count }}</strong>
          </button>
        </div>

        <div class="pool-filters__section">
          <h2>Статус</h2>
          <div class="status-chips">
            <TbChip :active="statusFilter === 'ready'" @click="statusFilter = 'ready'">
              Готова
            </TbChip>
            <TbChip :active="statusFilter === 'pending'" @click="statusFilter = 'pending'">
              В работе
            </TbChip>
            <TbChip :active="statusFilter === 'draft'" @click="statusFilter = 'draft'">
              Черновик
            </TbChip>
          </div>
        </div>

        <div v-if="incidentTypeOptions.length" class="pool-filters__section">
          <h2>Тип происшествия</h2>
          <div class="status-chips">
            <TbChip
              v-for="opt in incidentTypeOptions"
              :key="opt.code"
              :active="selectedIncidentTypes.includes(opt.code)"
              @click="toggleIncidentType(opt.code)"
            >
              {{ opt.label }}
            </TbChip>
          </div>
        </div>

        <label class="pool-filters__toggle">
          <input v-model="unusedOnly" type="checkbox" />
          Только не в вариантах
        </label>
      </aside>

      <div class="pool-list">
        <div class="pool-list__meta">
          <div class="pool-list__chips">
            <strong>{{ ruCount(filteredTickets.length, 'карточка', 'карточки', 'карточек') }}</strong>
            <button
              v-for="chip in activeFilters"
              :key="chip.id"
              type="button"
              class="filter-chip"
              @click="chip.clear()"
            >
              {{ chip.label }}
              <span aria-hidden="true">×</span>
            </button>
            <button
              v-if="activeFilters.length"
              type="button"
              class="pool-list__reset"
              @click="resetFilters"
            >
              Сбросить
            </button>
          </div>
          <span class="pool-list__sort">Недавно изменённые</span>
        </div>

        <p v-if="error && !createOpen" class="curriculum-error">{{ error }}</p>
        <p v-if="loading" class="curriculum-empty">Загрузка…</p>
        <p v-else-if="!filteredTickets.length" class="curriculum-empty">
          Карточки не найдены.
          <button type="button" class="pool-page__inline-link" @click="openCreate">
            Создать
          </button>
        </p>
        <template v-else>
          <div class="curriculum-table-head pool-table-head">
            <span>Карточка</span>
            <span>Тема</span>
            <span>Тип</span>
            <span>Слотов</span>
            <span>В вариантах</span>
            <span>Статус</span>
            <span>Изменена</span>
          </div>
          <div class="pool-table-list">
            <button
              v-for="ticket in visibleTickets"
              :key="ticket.id"
              type="button"
              class="ticket-row"
              :class="{ 'ticket-row--selected': selectedId === ticket.id }"
              @click="openTicket(ticket)"
            >
              <div class="ticket-row__main">
                <span class="ticket-row__icon" aria-hidden="true">!</span>
                <div>
                  <strong>{{ ticket.title }}</strong>
                  <p>{{ ticketSubtitle(ticket) }}</p>
                </div>
              </div>
              <div class="ticket-row__cell">
                <span class="ticket-row__tag">{{ topicName(ticket.topic_id) }}</span>
              </div>
              <div class="ticket-row__cell">{{ ticketTypeLabel(ticket) }}</div>
              <div class="ticket-row__cell">{{ ticketSlotsLabel(ticket) }}</div>
              <div class="ticket-row__cell">{{ ticketUsageLabel(ticket) }}</div>
              <div class="ticket-row__cell">
                <TbBadge :tone="ticketStatusTone(ticket)">
                  {{ ticketStatusLabel(ticket) }}
                </TbBadge>
              </div>
              <div class="ticket-row__cell ticket-row__cell--date">
                {{ formatChanged(ticket.created_at) }}
              </div>
            </button>
          </div>
          <button
            v-if="remainingCount > 0"
            type="button"
            class="pool-list__more"
            @click="visibleCount += PAGE_SIZE"
          >
            Показать ещё {{ remainingCount > PAGE_SIZE ? PAGE_SIZE : remainingCount }}
          </button>
        </template>
      </div>
    </div>

    <TbModal v-model:open="createOpen" title="Новая карточка">
      <div class="curriculum-form">
        <TbField label="Тема">
          <TbSelect v-model="form.topic_id" :options="topicOptions" placeholder="Выберите тему" />
        </TbField>
        <TbField label="Название">
          <TbInput v-model="form.title" />
        </TbField>
        <TbField label="Текст">
          <TbTextarea v-model="form.body" :rows="5" :maxlength="4000" show-counter />
        </TbField>
        <p v-if="error" class="curriculum-error">{{ error }}</p>
      </div>
      <template #footer>
        <TbButton variant="secondary" @click="createOpen = false">Отмена</TbButton>
        <TbButton :busy="busy" @click="create">Создать</TbButton>
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

.pool-layout {
  display: grid;
  grid-template-columns: 280px minmax(0, 1fr);
  gap: 24px;
  align-items: start;
  min-height: 0;
}

.pool-filters {
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 20px;
  background: var(--color-surface);
  border-radius: var(--radius-md);
}

.pool-filters__section {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.pool-filters__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.pool-filters h2 {
  margin: 0;
  font: 700 14px/1.3 var(--font-sans);
}

.pool-filters__count {
  color: var(--color-text-muted);
  font: var(--font-cap);
}

.topic-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  width: 100%;
  padding: 10px 12px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  text-align: left;
  cursor: pointer;
  color: inherit;
  font: 500 13px/1.3 var(--font-sans);
}

.topic-row:hover,
.topic-row--active {
  background: var(--color-secondary);
}

.topic-row span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.topic-row strong {
  color: var(--color-text-muted);
  font: var(--font-cap);
  flex-shrink: 0;
}

.status-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.pool-filters__toggle {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font: var(--font-mute);
  cursor: pointer;
}

.pool-list {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 20px 24px 24px;
  background: var(--color-surface);
  border-radius: var(--radius-md);
}

.pool-list__meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}

.pool-list__chips {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.pool-list__chips strong {
  font: var(--font-mute);
}

.filter-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 28px;
  padding: 0 10px;
  border: 1px solid var(--color-border);
  border-radius: 999px;
  background: var(--color-surface);
  color: var(--color-text);
  font: 500 12px/1.3 var(--font-sans);
  cursor: pointer;
}

.pool-list__reset,
.pool-list__more,
.pool-page__inline-link {
  border: 0;
  background: transparent;
  color: var(--color-primary);
  font: 600 13px/1.3 var(--font-sans);
  cursor: pointer;
  padding: 0;
}

.pool-list__sort {
  color: var(--color-text-muted);
  font: var(--font-cap);
}

.pool-table-head {
  grid-template-columns: minmax(220px, 2.2fr) 140px 80px 80px 120px 100px 80px;
  gap: 12px;
  padding: 0 14px 8px;
}

.pool-table-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.ticket-row {
  display: grid;
  grid-template-columns: minmax(220px, 2.2fr) 140px 80px 80px 120px 100px 80px;
  gap: 12px;
  align-items: center;
  width: 100%;
  padding: 12px 14px;
  border: 1.5px solid transparent;
  border-radius: var(--radius-sm);
  background: transparent;
  text-align: left;
  cursor: pointer;
  color: inherit;
  font: inherit;
  box-sizing: border-box;
}

.ticket-row:hover {
  background: var(--color-primary-soft);
}

.ticket-row--selected {
  background: var(--color-secondary);
  border-color: var(--color-primary);
}

.ticket-row__main {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  min-width: 0;
}

.ticket-row__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 8px;
  background: var(--color-danger-soft);
  color: var(--color-danger);
  font: 700 14px/1 var(--font-sans);
  flex-shrink: 0;
}

.ticket-row__main strong {
  display: block;
  font: var(--font-block);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ticket-row__main p {
  margin: 2px 0 0;
  color: var(--color-text-muted);
  font: var(--font-cap);
}

.ticket-row__cell {
  color: var(--color-text-muted);
  font: 500 13px/1.3 var(--font-sans);
  min-width: 0;
}

.ticket-row__tag {
  display: inline-flex;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  padding: 4px 8px;
  border-radius: 999px;
  border: 1px solid var(--color-border);
  font: 500 12px/1.3 var(--font-sans);
}

.ticket-row__cell--date {
  color: var(--color-text-subtle);
}

.pool-list__more {
  align-self: flex-start;
  margin-top: 4px;
  padding: 8px 0;
}

@media (max-width: 1200px) {
  .pool-layout {
    grid-template-columns: 1fr;
  }

  .pool-table-head,
  .ticket-row {
    grid-template-columns: minmax(0, 1fr) 100px 90px;
  }

  .pool-table-head span:nth-child(n + 4),
  .ticket-row__cell:nth-child(n + 4) {
    display: none;
  }
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
