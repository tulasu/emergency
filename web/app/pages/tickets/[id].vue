<script setup lang="ts">
import type {
  EmergencyService,
  IncidentType,
  ReferenceAnswer,
  TagGroup,
  Topic,
} from '~/types/curriculum';
import { apiErrorMessage } from '~/utils/api-error';

definePageMeta({
  middleware: ['auth', 'staff'],
});

const route = useRoute();
const ticketId = computed(() => String(route.params.id));
const ticketsApi = useTickets();
const catalogApi = useCatalog();
const { $api } = useNuxtApp();

const title = ref('');
const body = ref('');
const topicId = ref('');
const topics = ref<Topic[]>([]);
const types = ref<IncidentType[]>([]);
const services = ref<EmergencyService[]>([]);
const tagGroups = ref<TagGroup[]>([]);
const typeCode = ref('');
const tagCodes = ref<string[]>([]);
const serviceCodes = ref<string[]>([]);
const applicantLastName = ref('');
const applicantFirstName = ref('');
const callerNumber = ref('');
const dictatedNumber = ref('');
const paraphraseNotes = ref('');
const previewOpen = ref(false);
const loading = ref(true);
const busy = ref(false);
const refBusy = ref(false);
const error = ref('');
const toast = ref('');
const hasReference = ref(false);

const topicOptions = computed(() =>
  topics.value.map((t) => ({ value: t.id, label: t.title })),
);
const typeOptions = computed(() =>
  types.value.map((t) => ({ value: t.code, label: t.title })),
);

const previewFacts = computed(() => [
  { label: 'Тип', value: types.value.find((t) => t.code === typeCode.value)?.title || '—' },
  { label: 'ФИО', value: [applicantLastName.value, applicantFirstName.value].filter(Boolean).join(' ') || '—' },
  { label: 'Телефон', value: callerNumber.value || '—' },
  { label: 'Признаки', value: tagCodes.value.length ? `${tagCodes.value.length} шт.` : '—' },
  { label: 'Службы', value: serviceCodes.value.length ? `${serviceCodes.value.length} шт.` : '—' },
]);

async function loadTags(): Promise<void> {
  if (!typeCode.value) {
    tagGroups.value = [];
    return;
  }
  tagGroups.value = await catalogApi.tags(typeCode.value);
}

function toggleTag(code: string, mode: string): void {
  const set = new Set(tagCodes.value);
  if (mode === 'single') {
    const group = tagGroups.value.find((g) => g.tags.some((t) => t.code === code));
    for (const t of group?.tags || []) {
      set.delete(t.code);
    }
    if (!tagCodes.value.includes(code)) {
      set.add(code);
    }
  } else if (set.has(code)) {
    set.delete(code);
  } else {
    set.add(code);
  }
  tagCodes.value = [...set];
}

function toggleService(code: string): void {
  const set = new Set(serviceCodes.value);
  if (set.has(code)) {
    set.delete(code);
  } else {
    set.add(code);
  }
  serviceCodes.value = [...set];
}

async function load(): Promise<void> {
  loading.value = true;
  error.value = '';
  try {
    const [ticket, topicList, typeList, serviceList, ref] = await Promise.all([
      ticketsApi.get(ticketId.value),
      $api<Topic[]>('/topics'),
      catalogApi.incidentTypes(),
      catalogApi.services(),
      catalogApi.getReference(ticketId.value),
    ]);
    title.value = ticket.title;
    body.value = ticket.body;
    topicId.value = ticket.topic_id;
    topics.value = topicList;
    types.value = typeList;
    services.value = serviceList;
    if (ref) {
      hasReference.value = true;
      typeCode.value = ref.incident_type_code;
      tagCodes.value = [...ref.tag_codes];
      serviceCodes.value = [...ref.service_codes];
      applicantLastName.value = ref.applicant_last_name || '';
      applicantFirstName.value = ref.applicant_first_name || '';
      callerNumber.value = ref.caller_number || '';
      dictatedNumber.value = ref.dictated_number || '';
      await loadTags();
    }
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    loading.value = false;
  }
}

async function saveMeta(): Promise<void> {
  busy.value = true;
  error.value = '';
  toast.value = '';
  try {
    await ticketsApi.update(ticketId.value, {
      topic_id: topicId.value,
      title: title.value.trim(),
      body: body.value.trim(),
    });
    toast.value = 'Карточка сохранена';
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    busy.value = false;
  }
}

async function saveReference(): Promise<void> {
  if (!typeCode.value) {
    error.value = 'Укажите тип происшествия для эталона';
    return;
  }
  refBusy.value = true;
  error.value = '';
  toast.value = '';
  try {
    const bodyRef: Omit<ReferenceAnswer, 'ticket_id'> = {
      incident_type_code: typeCode.value,
      tag_codes: tagCodes.value,
      service_codes: serviceCodes.value,
      applicant_last_name: applicantLastName.value,
      applicant_first_name: applicantFirstName.value,
      caller_number: callerNumber.value,
      dictated_number: dictatedNumber.value,
    };
    await catalogApi.setReference(ticketId.value, bodyRef);
    hasReference.value = true;
    if (paraphraseNotes.value.trim()) {
      // paraphrases domain not yet on API — keep in ticket body appendix for now
      toast.value = 'Эталон сохранён. Перефразы пока храните в тексте карточки.';
    } else {
      toast.value = 'Эталонный ответ сохранён';
    }
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    refBusy.value = false;
  }
}

async function onTypeChange(): Promise<void> {
  tagCodes.value = [];
  await loadTags();
}

onMounted(() => {
  void load();
});
</script>

<template>
  <section class="page-sheet page-sheet--wide">
    <div class="sheet">
      <header class="sheet__header">
        <div class="sheet__header-row">
          <div>
            <h1 class="page-title">Составление карточки</h1>
            <p class="page-sub">
              Текст сценария, эталон и предпоказ
              <template v-if="hasReference"> · эталон задан</template>
            </p>
          </div>
          <div class="actions">
            <TbButton variant="ghost" @click="previewOpen = true">Предпоказ</TbButton>
            <TbButton variant="secondary" @click="navigateTo('/tickets')">К списку</TbButton>
          </div>
        </div>
      </header>
      <div class="sheet__rule"></div>
      <div class="sheet__body">
        <p v-if="loading" class="curriculum-empty">Загрузка…</p>
        <div v-else class="editor-layout">
          <div class="curriculum-form">
            <h2 class="sheet__section-title">Исходный текст</h2>
            <TbField label="Тема">
              <TbSelect v-model="topicId" :options="topicOptions" />
            </TbField>
            <TbField label="Название">
              <TbInput v-model="title" />
            </TbField>
            <TbField label="Текст билета">
              <TbTextarea v-model="body" :rows="10" :maxlength="4000" show-counter />
            </TbField>
            <TbButton :busy="busy" @click="saveMeta">Сохранить карточку</TbButton>
          </div>
          <div class="curriculum-form">
            <h2 class="sheet__section-title">Эталонный ответ</h2>
            <TbField label="Тип происшествия">
              <TbSelect
                v-model="typeCode"
                :options="[{ value: '', label: 'Не выбран' }, ...typeOptions]"
                @update:model-value="onTypeChange"
              />
            </TbField>
            <div v-for="group in tagGroups" :key="group.code" class="tag-group">
              <p class="tag-group__title">{{ group.title }}</p>
              <div class="chip-row">
                <button
                  v-for="tag in group.tags"
                  :key="tag.code"
                  type="button"
                  class="chip"
                  :class="{ 'chip--on': tagCodes.includes(tag.code) }"
                  @click="toggleTag(tag.code, group.selection_mode)"
                >
                  {{ tag.title }}
                </button>
              </div>
            </div>
            <p class="tag-group__title">Состав служб</p>
            <div class="chip-row">
              <button
                v-for="svc in services"
                :key="svc.code"
                type="button"
                class="chip"
                :class="{ 'chip--on': serviceCodes.includes(svc.code) }"
                @click="toggleService(svc.code)"
              >
                {{ svc.title }}
              </button>
            </div>
            <div class="name-row">
              <TbField label="Фамилия">
                <TbInput v-model="applicantLastName" />
              </TbField>
              <TbField label="Имя">
                <TbInput v-model="applicantFirstName" />
              </TbField>
            </div>
            <div class="name-row">
              <TbField label="Номер звонящего">
                <TbInput v-model="callerNumber" />
              </TbField>
              <TbField label="Набранный номер">
                <TbInput v-model="dictatedNumber" />
              </TbField>
            </div>
            <TbField label="Перефразы / маркеры (заметки для сценария)">
              <TbTextarea
                v-model="paraphraseNotes"
                :rows="3"
                :maxlength="2000"
                placeholder="Формулировки перефразов и маркеров — до появления отдельного API"
              />
            </TbField>
            <TbButton :busy="refBusy" @click="saveReference">Сохранить эталон</TbButton>
          </div>
        </div>
        <p v-if="error" class="curriculum-error">{{ error }}</p>
        <p v-if="toast" class="toast">{{ toast }}</p>
      </div>
    </div>

    <TbModal v-model:open="previewOpen" title="Предпоказ карточки">
      <div class="preview">
        <h3>{{ title || 'Без названия' }}</h3>
        <p class="preview__body">{{ body }}</p>
        <ul>
          <li v-for="fact in previewFacts" :key="fact.label">
            <strong>{{ fact.label }}:</strong> {{ fact.value }}
          </li>
        </ul>
      </div>
    </TbModal>
  </section>
</template>

<style scoped>
.actions {
  display: flex;
  gap: 8px;
}

.editor-layout {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 24px;
  align-items: start;
}

.tag-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.tag-group__title {
  margin: 0;
  font: 700 13px/1.3 var(--font-sans);
}

.chip-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.chip {
  padding: 6px 12px;
  border-radius: 999px;
  border: 1px solid var(--color-border);
  background: var(--color-bg);
  font: 500 12px/1.3 var(--font-sans);
  cursor: pointer;
  color: inherit;
}

.chip--on {
  background: var(--color-primary);
  border-color: var(--color-primary);
  color: #fff;
}

.name-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.preview h3 {
  margin: 0 0 8px;
}

.preview__body {
  white-space: pre-wrap;
  color: var(--color-text-muted);
}

.preview ul {
  margin: 12px 0 0;
  padding-left: 18px;
}

.toast {
  margin: 0;
  color: var(--color-success);
  font: var(--font-mute);
}

@media (max-width: 1000px) {
  .editor-layout {
    grid-template-columns: 1fr;
  }
}
</style>
