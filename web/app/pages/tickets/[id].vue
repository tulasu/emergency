<script setup lang="ts">
import type { Topic } from '~/types/curriculum';
import { apiErrorMessage } from '~/utils/api-error';

definePageMeta({
  middleware: ['auth', 'staff'],
});

const route = useRoute();
const ticketId = computed(() => String(route.params.id));
const ticketsApi = useTickets();
const { $api } = useNuxtApp();

const title = ref('');
const body = ref('');
const topicId = ref('');
const topics = ref<Topic[]>([]);
const loading = ref(true);
const busy = ref(false);
const error = ref('');
const toast = ref('');

const topicOptions = computed(() =>
  topics.value.map((t) => ({ value: t.id, label: t.title })),
);

async function load(): Promise<void> {
  loading.value = true;
  error.value = '';
  try {
    const [ticket, topicList] = await Promise.all([
      ticketsApi.get(ticketId.value),
      $api<Topic[]>('/topics'),
    ]);
    title.value = ticket.title;
    body.value = ticket.body;
    topicId.value = ticket.topic_id;
    topics.value = topicList;
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    loading.value = false;
  }
}

async function save(): Promise<void> {
  busy.value = true;
  error.value = '';
  toast.value = '';
  try {
    await ticketsApi.update(ticketId.value, {
      topic_id: topicId.value,
      title: title.value.trim(),
      body: body.value.trim(),
    });
    toast.value = 'Сохранено';
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
  <section class="page-sheet">
    <div class="sheet">
      <header class="sheet__header">
        <div class="sheet__header-row">
          <div>
            <h1 class="page-title">Карточка</h1>
            <p class="page-sub">Редактирование сценария</p>
          </div>
          <TbButton variant="secondary" @click="navigateTo('/tickets')">К списку</TbButton>
        </div>
      </header>
      <div class="sheet__rule"></div>
      <div class="sheet__body">
        <p v-if="loading" class="curriculum-empty">Загрузка…</p>
        <div v-else class="curriculum-form">
          <TbField label="Тема">
            <TbSelect v-model="topicId" :options="topicOptions" />
          </TbField>
          <TbField label="Название">
            <TbInput v-model="title" />
          </TbField>
          <TbField label="Текст">
            <TbTextarea v-model="body" :rows="8" :maxlength="4000" show-counter />
          </TbField>
          <p v-if="error" class="curriculum-error">{{ error }}</p>
          <p v-if="toast" class="toast">{{ toast }}</p>
          <TbButton :busy="busy" @click="save">Сохранить</TbButton>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.toast {
  margin: 0;
  color: var(--color-success);
  font: var(--font-mute);
}
</style>
