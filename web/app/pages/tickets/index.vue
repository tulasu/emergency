<script setup lang="ts">
import type { Ticket, Topic } from '~/types/curriculum';
import { apiErrorMessage } from '~/utils/api-error';
import { audioStatusLabel } from '~/utils/curriculum-labels';

definePageMeta({
  middleware: ['auth', 'staff'],
});

const ticketsApi = useTickets();
const { $api } = useNuxtApp();

const query = ref('');
const tickets = ref<Ticket[]>([]);
const topics = ref<Topic[]>([]);
const loading = ref(true);
const error = ref('');
const createOpen = ref(false);
const busy = ref(false);
const form = reactive({
  title: '',
  body: '',
  topic_id: '',
});

let timer: ReturnType<typeof setTimeout> | null = null;

const topicOptions = computed(() =>
  topics.value.map((t) => ({ value: t.id, label: t.title })),
);

const topicName = computed(() => {
  const map = new Map(topics.value.map((t) => [t.id, t.title]));
  return (id: string) => map.get(id) || 'Без темы';
});

const folders = computed(() => {
  const map = new Map<string, Ticket[]>();
  for (const ticket of tickets.value) {
    const key = topicName.value(ticket.topic_id);
    const list = map.get(key) || [];
    list.push(ticket);
    map.set(key, list);
  }
  return Array.from(map.entries()).map(([title, items], index) => ({
    title: `Тема ${index + 1} · ${title}`,
    items,
  }));
});

function ticketTags(ticket: Ticket): string[] {
  const topic = topicName.value(ticket.topic_id);
  const audio = audioStatusLabel(ticket.audio_status);
  if (audio === 'Без аудио') {
    return [topic];
  }
  return [topic, audio];
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

onMounted(() => {
  void load();
});
</script>

<template>
  <section class="pool-page">
    <header class="pool-page__header">
      <h1 class="page-title">Карточки</h1>
      <div class="pool-page__actions">
        <div class="search-wrap">
          <TbSearch v-model="query" placeholder="Поиск" @update:model-value="onSearch" />
        </div>
      </div>
    </header>

    <div class="pool-page__body">
      <p v-if="error && !createOpen" class="curriculum-error">{{ error }}</p>
      <p v-if="loading" class="curriculum-empty">Загрузка…</p>
      <p v-else-if="!tickets.length" class="curriculum-empty">
        Карточки не найдены.
        <button type="button" class="pool-page__inline-link" @click="openCreate">Создать</button>
      </p>
      <div v-else class="curriculum-folders">
        <section v-for="folder in folders" :key="folder.title" class="curriculum-folder">
          <h2 class="curriculum-folder__title">{{ folder.title }}</h2>
          <div class="curriculum-grid">
            <TbFlashCard
              v-for="ticket in folder.items"
              :key="ticket.id"
              :title="ticket.title"
              :tags="ticketTags(ticket)"
              action-label="Открыть"
              @action="navigateTo(`/tickets/${ticket.id}`)"
            />
          </div>
        </section>
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

.pool-page__body {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 28px;
  min-height: 0;
}

.pool-page__inline-link {
  margin-left: 8px;
  border: 0;
  background: transparent;
  color: var(--color-primary);
  font: inherit;
  cursor: pointer;
  text-decoration: underline;
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
