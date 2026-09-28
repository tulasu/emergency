<script setup lang="ts">
import type { Attempt, Ticket } from '~/types/curriculum';
import type { UserProfile } from '~/types/groups';
import { apiErrorMessage } from '~/utils/api-error';
import { attemptStatusLabel } from '~/utils/curriculum-labels';

definePageMeta({
  middleware: ['auth', 'staff'],
});

const route = useRoute();
const attemptId = computed(() => String(route.params.id));
const attemptsApi = useAttempts();
const ticketsApi = useTickets();
const usersApi = useUsers();

const attempt = ref<Attempt | null>(null);
const tickets = ref<Ticket[]>([]);
const user = ref<UserProfile | null>(null);
const loading = ref(true);
const error = ref('');
const toast = ref('');
const grantBusy = ref(false);
const remark = ref('');

const ticketById = computed(() =>
  Object.fromEntries(tickets.value.map((t) => [t.id, t])),
);

const fieldLabel: Record<string, string> = {
  incident_type_code: 'Тип происшествия',
  tag_codes: 'Признаки',
  service_codes: 'Службы',
  applicant_last_name: 'Фамилия',
  applicant_first_name: 'Имя',
  caller_number: 'Номер звонящего',
  dictated_number: 'Набранный номер',
};

function formatVal(v: unknown): string {
  if (Array.isArray(v)) {
    return v.join(', ') || '—';
  }
  if (v == null || v === '') {
    return '—';
  }
  return String(v);
}

async function load(): Promise<void> {
  loading.value = true;
  error.value = '';
  try {
    const at = await attemptsApi.get(attemptId.value);
    attempt.value = at;
    const [ticketList, profile] = await Promise.all([
      ticketsApi.listByVariant(at.variant_id),
      usersApi.get(at.user_id).catch(() => null),
    ]);
    tickets.value = ticketList;
    user.value = profile;
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    loading.value = false;
  }
}

async function grantExtra(): Promise<void> {
  if (!attempt.value) {
    return;
  }
  grantBusy.value = true;
  error.value = '';
  toast.value = '';
  try {
    await attemptsApi.grant(attempt.value.user_id, {
      variant_id: attempt.value.variant_id,
    });
    toast.value = 'Новая попытка выдана';
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    grantBusy.value = false;
  }
}

function saveRemark(): void {
  toast.value = remark.value.trim()
    ? 'Замечание сохранено локально (сервер замечаний пока не подключён)'
    : '';
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
            <h1 class="page-title">Разбор попытки</h1>
            <p class="page-sub">
              {{ user?.full_name || attempt?.user_id?.slice(0, 8) || '…' }} · попытка
              № {{ attempt?.attempt_no ?? '—' }} ·
              {{ attemptStatusLabel(attempt?.status) }}
            </p>
          </div>
          <div class="score" v-if="attempt">
            <span>Оценка</span>
            <strong>{{ attempt.report?.overall_score ?? attempt.score ?? '—' }}%</strong>
          </div>
        </div>
      </header>
      <div class="sheet__rule"></div>
      <div class="sheet__body">
        <p v-if="error" class="curriculum-error">{{ error }}</p>
        <p v-if="toast" class="toast">{{ toast }}</p>
        <p v-if="loading" class="curriculum-empty">Загрузка…</p>
        <template v-else-if="attempt">
          <div class="actions">
            <TbButton :busy="grantBusy" @click="grantExtra">Дать ещё попытку</TbButton>
          </div>
          <div class="remark">
            <TbField label="Замечание обучающемуся">
              <TbTextarea v-model="remark" :rows="3" :maxlength="1000" />
            </TbField>
            <TbButton variant="secondary" @click="saveRemark">Сохранить замечание</TbButton>
          </div>
          <div
            v-for="item in attempt.report?.items || []"
            :key="item.ticket_id"
            class="report-card"
          >
            <div class="report-card__head">
              <h2>{{ ticketById[item.ticket_id]?.title || item.ticket_id.slice(0, 8) }}</h2>
              <TbBadge :tone="item.score >= 70 ? 'success' : 'warning'">
                {{ item.score }}%
              </TbBadge>
            </div>
            <p v-if="!item.errors?.length" class="ok">Ошибок нет</p>
            <ul v-else class="errors">
              <li v-for="(err, idx) in item.errors" :key="idx">
                <strong>{{ fieldLabel[err.field] || err.field }}</strong>
                <span>ожидалось: {{ formatVal(err.expected) }}</span>
                <span>получено: {{ formatVal(err.actual) }}</span>
              </li>
            </ul>
          </div>
        </template>
      </div>
    </div>
  </section>
</template>

<style scoped>
.score {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
}

.score strong {
  font: 700 28px/1.1 var(--font-sans);
}

.actions,
.remark {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.report-card {
  padding: 16px 20px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.report-card__head {
  display: flex;
  justify-content: space-between;
  gap: 8px;
}

.report-card__head h2 {
  margin: 0;
  font: 700 16px/1.3 var(--font-sans);
}

.ok {
  margin: 0;
  color: var(--color-success);
}

.errors {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.errors li {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 8px 10px;
  background: var(--color-bg);
  border-radius: var(--radius-sm);
  font: 400 13px/1.4 var(--font-sans);
}

.errors span {
  color: var(--color-text-muted);
}

.toast {
  margin: 0;
  color: var(--color-success);
  font: var(--font-mute);
}
</style>
