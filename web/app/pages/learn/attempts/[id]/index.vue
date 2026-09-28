<script setup lang="ts">
import type {
  Attempt,
  AttemptAnswer,
  EmergencyService,
  IncidentType,
  SaveAnswerBody,
  TagGroup,
  Ticket,
} from '~/types/curriculum';
import { apiErrorMessage } from '~/utils/api-error';

definePageMeta({
  middleware: 'auth',
});

const route = useRoute();
const attemptId = computed(() => String(route.params.id));
const attemptsApi = useAttempts();
const ticketsApi = useTickets();
const catalogApi = useCatalog();

const attempt = ref<Attempt | null>(null);
const tickets = ref<Ticket[]>([]);
const types = ref<IncidentType[]>([]);
const services = ref<EmergencyService[]>([]);
const tagGroups = ref<TagGroup[]>([]);
const activeTicketId = ref('');
const form = ref<SaveAnswerBody>(emptyAnswer());
const typeCode = ref('');
const loading = ref(true);
const saving = ref(false);
const submitting = ref(false);
const error = ref('');
const toast = ref('');
const confirmSubmit = ref(false);
let saveTimer: ReturnType<typeof setTimeout> | null = null;

const activeTicket = computed(
  () => tickets.value.find((t) => t.id === activeTicketId.value) || null,
);

const answeredCount = computed(() => {
  const answers = attempt.value?.answers || {};
  return tickets.value.filter((t) => {
    const a = answers[t.id];
    return a && (a.incident_type_code || a.tag_codes?.length || a.service_codes?.length);
  }).length;
});

const remainingLabel = computed(() => {
  const deadline = attempt.value?.deadline_at;
  if (!deadline) {
    return 'без лимита';
  }
  const ms = new Date(deadline).getTime() - Date.now();
  if (ms <= 0) {
    return 'время вышло';
  }
  const totalSec = Math.floor(ms / 1000);
  const m = Math.floor(totalSec / 60);
  const s = totalSec % 60;
  return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`;
});

const readonly = computed(() =>
  ['submitted', 'timed_out', 'scored', 'finished'].includes(attempt.value?.status || ''),
);

const typeOptions = computed(() =>
  types.value.map((t) => ({ value: t.code, label: t.title })),
);

function emptyAnswer(): SaveAnswerBody {
  return {
    incident_type_code: null,
    tag_codes: [],
    service_codes: [],
    applicant_last_name: '',
    applicant_first_name: '',
    caller_number: '',
    dictated_number: '',
    notes: '',
  };
}

function fromAnswer(a?: AttemptAnswer | null): SaveAnswerBody {
  if (!a) {
    return emptyAnswer();
  }
  return {
    incident_type_code: a.incident_type_code ?? null,
    tag_codes: [...(a.tag_codes || [])],
    service_codes: [...(a.service_codes || [])],
    applicant_last_name: a.applicant_last_name || '',
    applicant_first_name: a.applicant_first_name || '',
    caller_number: a.caller_number || '',
    dictated_number: a.dictated_number || '',
    notes: a.notes || '',
  };
}

function ticketDone(ticketId: string): boolean {
  const a = attempt.value?.answers?.[ticketId];
  return Boolean(a?.incident_type_code);
}

async function loadTags(typeCode: string | null | undefined): Promise<void> {
  if (!typeCode) {
    tagGroups.value = [];
    return;
  }
  try {
    tagGroups.value = await catalogApi.tags(typeCode);
  } catch {
    tagGroups.value = [];
  }
}

function selectTicket(ticketId: string): void {
  activeTicketId.value = ticketId;
  form.value = fromAnswer(attempt.value?.answers?.[ticketId]);
  typeCode.value = form.value.incident_type_code || '';
  void loadTags(typeCode.value);
}

async function onTypeChange(): Promise<void> {
  form.value.incident_type_code = typeCode.value || null;
  form.value.tag_codes = [];
  await loadTags(typeCode.value);
  scheduleSave();
}

function toggleTag(code: string, mode: string): void {
  if (readonly.value) {
    return;
  }
  const set = new Set(form.value.tag_codes);
  if (mode === 'single') {
    if (set.has(code)) {
      set.delete(code);
    } else {
      // clear siblings of same group
      const group = tagGroups.value.find((g) => g.tags.some((t) => t.code === code));
      for (const t of group?.tags || []) {
        set.delete(t.code);
      }
      set.add(code);
    }
  } else if (set.has(code)) {
    set.delete(code);
  } else {
    set.add(code);
  }
  form.value.tag_codes = [...set];
  scheduleSave();
}

function toggleService(code: string): void {
  if (readonly.value) {
    return;
  }
  const set = new Set(form.value.service_codes);
  if (set.has(code)) {
    set.delete(code);
  } else {
    set.add(code);
  }
  form.value.service_codes = [...set];
  scheduleSave();
}

async function applyRecommend(): Promise<void> {
  if (!form.value.incident_type_code) {
    return;
  }
  try {
    form.value.service_codes = await catalogApi.recommend(
      form.value.incident_type_code,
      form.value.tag_codes,
    );
    scheduleSave();
  } catch (err) {
    error.value = apiErrorMessage(err);
  }
}

function scheduleSave(): void {
  if (readonly.value) {
    return;
  }
  if (saveTimer) {
    clearTimeout(saveTimer);
  }
  saveTimer = setTimeout(() => {
    void save();
  }, 600);
}

async function save(): Promise<void> {
  if (!attempt.value || !activeTicketId.value || readonly.value) {
    return;
  }
  saving.value = true;
  error.value = '';
  try {
    attempt.value = await attemptsApi.saveAnswer(
      attempt.value.id,
      activeTicketId.value,
      form.value,
    );
    toast.value = 'Черновик сохранён';
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    saving.value = false;
  }
}

async function doSubmit(): Promise<void> {
  if (!attempt.value) {
    return;
  }
  submitting.value = true;
  error.value = '';
  try {
    await save();
    attempt.value = await attemptsApi.submit(attempt.value.id);
    confirmSubmit.value = false;
    await navigateTo(`/learn/attempts/${attempt.value.id}/result`);
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    submitting.value = false;
  }
}

async function load(): Promise<void> {
  loading.value = true;
  error.value = '';
  try {
    let at = await attemptsApi.get(attemptId.value);
    if (at.status === 'available') {
      at = await attemptsApi.start(at.id);
    }
    attempt.value = at;
    const [ticketList, typeList, serviceList] = await Promise.all([
      ticketsApi.listByVariant(at.variant_id),
      catalogApi.incidentTypes(),
      catalogApi.services(),
    ]);
    tickets.value = ticketList;
    types.value = typeList;
    services.value = serviceList;
    const first = ticketList[0];
    if (first) {
      selectTicket(first.id);
    }
    if (readonly.value) {
      await navigateTo(`/learn/attempts/${at.id}/result`);
    }
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  void load();
});

onUnmounted(() => {
  if (saveTimer) {
    clearTimeout(saveTimer);
  }
});
</script>

<template>
  <section class="page-sheet page-sheet--wide">
    <div class="sheet">
      <header class="sheet__header">
        <div class="sheet__header-row">
          <div>
            <nav class="curriculum-breadcrumbs" aria-label="Навигация">
              <NuxtLink to="/" class="curriculum-breadcrumbs__link">Главная</NuxtLink>
              <span class="curriculum-breadcrumbs__sep">/</span>
              <span>Попытка {{ attempt?.attempt_no ?? '…' }}</span>
            </nav>
            <h1 class="page-title">Окно занятия</h1>
            <p class="page-sub">
              Задания {{ answeredCount }} / {{ tickets.length }} · до конца
              {{ remainingLabel }}
              <span v-if="saving"> · сохранение…</span>
            </p>
          </div>
          <div class="header-actions">
            <TbButton
              variant="secondary"
              :disabled="readonly"
              @click="confirmSubmit = true"
            >
              Завершить попытку
            </TbButton>
          </div>
        </div>
      </header>
      <div class="sheet__rule"></div>
      <div class="sheet__body">
        <p v-if="error" class="curriculum-error">{{ error }}</p>
        <p v-if="toast" class="toast">{{ toast }}</p>
        <p v-if="loading" class="curriculum-empty">Загрузка…</p>
        <div v-else class="attempt-layout">
          <aside class="task-list">
            <h2 class="sheet__section-title">Задания</h2>
            <button
              v-for="(ticket, index) in tickets"
              :key="ticket.id"
              type="button"
              class="task-item"
              :class="{
                'task-item--active': ticket.id === activeTicketId,
                'task-item--done': ticketDone(ticket.id),
              }"
              @click="selectTicket(ticket.id)"
            >
              <span class="task-item__num">{{ String(index + 1).padStart(2, '0') }}</span>
              <span class="task-item__title">{{ ticket.title }}</span>
              <TbBadge v-if="ticketDone(ticket.id)" tone="success">готово</TbBadge>
            </button>
          </aside>
          <div v-if="activeTicket" class="task-workspace">
            <div class="briefing">
              <h2>{{ activeTicket.title }}</h2>
              <p class="briefing__body">{{ activeTicket.body }}</p>
            </div>
            <div class="answer-form">
              <TbField label="Тип происшествия">
                <TbSelect
                  v-model="typeCode"
                  :options="[{ value: '', label: 'Не выбран' }, ...typeOptions]"
                  :disabled="readonly"
                  @update:model-value="onTypeChange"
                />
              </TbField>
              <div v-if="tagGroups.length" class="tag-groups">
                <div v-for="group in tagGroups" :key="group.code" class="tag-group">
                  <p class="tag-group__title">{{ group.title }}</p>
                  <div class="chip-row">
                    <button
                      v-for="tag in group.tags"
                      :key="tag.code"
                      type="button"
                      class="chip"
                      :class="{ 'chip--on': form.tag_codes.includes(tag.code) }"
                      :disabled="readonly"
                      @click="toggleTag(tag.code, group.selection_mode)"
                    >
                      {{ tag.title }}
                    </button>
                  </div>
                </div>
              </div>
              <div class="services-block">
                <div class="services-block__head">
                  <p class="tag-group__title">Службы</p>
                  <TbButton
                    variant="ghost"
                    :disabled="readonly || !form.incident_type_code"
                    @click="applyRecommend"
                  >
                    Рекомендовать
                  </TbButton>
                </div>
                <div class="chip-row">
                  <button
                    v-for="svc in services"
                    :key="svc.code"
                    type="button"
                    class="chip"
                    :class="{ 'chip--on': form.service_codes.includes(svc.code) }"
                    :disabled="readonly"
                    @click="toggleService(svc.code)"
                  >
                    {{ svc.title }}
                  </button>
                </div>
              </div>
              <div class="name-row">
                <TbField label="Фамилия заявителя">
                  <TbInput
                    v-model="form.applicant_last_name"
                    :disabled="readonly"
                    @update:model-value="scheduleSave"
                  />
                </TbField>
                <TbField label="Имя заявителя">
                  <TbInput
                    v-model="form.applicant_first_name"
                    :disabled="readonly"
                    @update:model-value="scheduleSave"
                  />
                </TbField>
              </div>
              <div class="name-row">
                <TbField label="Номер звонящего">
                  <TbInput
                    v-model="form.caller_number"
                    :disabled="readonly"
                    @update:model-value="scheduleSave"
                  />
                </TbField>
                <TbField label="Набранный номер">
                  <TbInput
                    v-model="form.dictated_number"
                    :disabled="readonly"
                    @update:model-value="scheduleSave"
                  />
                </TbField>
              </div>
              <TbField label="Заметки">
                <TbTextarea
                  v-model="form.notes"
                  :rows="3"
                  :maxlength="1000"
                  :disabled="readonly"
                  @update:model-value="scheduleSave"
                />
              </TbField>
            </div>
          </div>
        </div>
      </div>
    </div>

    <TbModal v-model:open="confirmSubmit" title="Завершить попытку?">
      <p>Ответы будут проверены по эталону. После сдачи изменить их нельзя.</p>
      <template #footer>
        <TbButton variant="ghost" @click="confirmSubmit = false">Отмена</TbButton>
        <TbButton :busy="submitting" @click="doSubmit">Сдать</TbButton>
      </template>
    </TbModal>
  </section>
</template>

<style scoped>
.header-actions {
  display: flex;
  gap: 8px;
}

.attempt-layout {
  display: grid;
  grid-template-columns: 280px minmax(0, 1fr);
  gap: 20px;
  align-items: start;
}

.task-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 16px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}

.task-item {
  display: grid;
  grid-template-columns: 32px 1fr auto;
  gap: 10px;
  align-items: center;
  width: 100%;
  padding: 10px 12px;
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  background: transparent;
  text-align: left;
  cursor: pointer;
  color: inherit;
  font: inherit;
}

.task-item--active {
  background: var(--color-secondary);
  border-color: color-mix(in srgb, var(--color-primary) 25%, transparent);
}

.task-item--done .task-item__num {
  color: var(--color-success);
}

.task-item__num {
  font: 700 12px/1.2 var(--font-sans);
  color: var(--color-text-muted);
}

.task-item__title {
  font: 500 13px/1.3 var(--font-sans);
}

.task-workspace {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.briefing {
  padding: 20px 24px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}

.briefing h2 {
  margin: 0 0 8px;
  font: 700 18px/1.3 var(--font-sans);
}

.briefing__body {
  margin: 0;
  white-space: pre-wrap;
  color: var(--color-text-muted);
  font: 400 14px/1.5 var(--font-sans);
}

.answer-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 20px 24px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}

.tag-groups,
.services-block {
  display: flex;
  flex-direction: column;
  gap: 12px;
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

.chip:disabled {
  opacity: 0.7;
  cursor: default;
}

.services-block__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.name-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.toast {
  margin: 0;
  color: var(--color-success);
  font: var(--font-mute);
}

@media (max-width: 900px) {
  .attempt-layout {
    grid-template-columns: 1fr;
  }

  .name-row {
    grid-template-columns: 1fr;
  }
}
</style>
