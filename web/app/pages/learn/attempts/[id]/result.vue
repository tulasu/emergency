<script setup lang="ts">
import type { Attempt, Ticket } from '~/types/curriculum';
import { apiErrorMessage } from '~/utils/api-error';

definePageMeta({
  middleware: 'auth',
});

const route = useRoute();
const attemptId = computed(() => String(route.params.id));
const attemptsApi = useAttempts();
const ticketsApi = useTickets();

const attempt = ref<Attempt | null>(null);
const tickets = ref<Ticket[]>([]);
const loading = ref(true);
const error = ref('');

const ticketById = computed(() => {
  const map: Record<string, Ticket> = {};
  for (const t of tickets.value) {
    map[t.id] = t;
  }
  return map;
});

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
    tickets.value = await ticketsApi.listByVariant(at.variant_id);
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    loading.value = false;
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
            <nav class="curriculum-breadcrumbs" aria-label="Навигация">
              <NuxtLink to="/" class="curriculum-breadcrumbs__link">Главная</NuxtLink>
              <span class="curriculum-breadcrumbs__sep">/</span>
              <NuxtLink to="/results" class="curriculum-breadcrumbs__link">Мои результаты</NuxtLink>
              <span class="curriculum-breadcrumbs__sep">/</span>
              <span>Разбор</span>
            </nav>
            <h1 class="page-title">Результат попытки</h1>
            <p class="page-sub">
              Попытка № {{ attempt?.attempt_no ?? '—' }} · статус
              {{ attempt?.status || '—' }}
            </p>
          </div>
          <div class="score-card" v-if="attempt">
            <span>Итог</span>
            <strong>{{ attempt.report?.overall_score ?? attempt.score ?? '—' }}%</strong>
          </div>
        </div>
      </header>
      <div class="sheet__rule"></div>
      <div class="sheet__body">
        <p v-if="error" class="curriculum-error">{{ error }}</p>
        <p v-if="loading" class="curriculum-empty">Загрузка…</p>
        <template v-else-if="attempt">
          <div
            v-for="item in attempt.report?.items || []"
            :key="item.ticket_id"
            class="report-card"
          >
            <div class="report-card__head">
              <h2>{{ ticketById[item.ticket_id]?.title || item.ticket_id.slice(0, 8) }}</h2>
              <TbBadge :tone="item.score >= 70 ? 'success' : item.score >= 40 ? 'warning' : 'danger'">
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
          <p v-if="!(attempt.report?.items || []).length" class="curriculum-empty">
            Отчёт пока пуст
          </p>
        </template>
      </div>
    </div>
  </section>
</template>

<style scoped>
.score-card {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
  padding: 12px 16px;
  background: var(--color-secondary);
  border-radius: var(--radius-md);
}

.score-card span {
  font: 500 12px/1.3 var(--font-sans);
  color: var(--color-text-muted);
}

.score-card strong {
  font: 700 28px/1.1 var(--font-sans);
}

.report-card {
  padding: 16px 20px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.report-card__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.report-card__head h2 {
  margin: 0;
  font: 700 16px/1.3 var(--font-sans);
}

.ok {
  margin: 0;
  color: var(--color-success);
  font: var(--font-mute);
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
  padding: 10px 12px;
  background: var(--color-bg);
  border-radius: var(--radius-sm);
  font: 400 13px/1.4 var(--font-sans);
}

.errors strong {
  font: 700 13px/1.3 var(--font-sans);
}

.errors span {
  color: var(--color-text-muted);
}
</style>
