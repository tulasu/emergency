<script setup lang="ts">
import type { Attempt, Ticket, Variant } from '~/types/curriculum';
import type { UserProfile } from '~/types/groups';
import { apiErrorMessage } from '~/utils/api-error';
import { attemptStatusLabel } from '~/utils/curriculum-labels';

definePageMeta({
  middleware: ['auth', 'staff'],
});

const route = useRoute();
const moduleId = computed(() => String(route.params.id));
const lessonId = computed(() => String(route.params.lessonId));
const variantId = computed(() => String(route.query.variant || ''));

const attemptsApi = useAttempts();
const variantsApi = useVariants();
const ticketsApi = useTickets();
const usersApi = useUsers();

const variants = ref<Variant[]>([]);
const selectedVariantId = ref('');
const attempts = ref<Attempt[]>([]);
const tickets = ref<Ticket[]>([]);
const usersById = ref<Record<string, UserProfile>>({});
const loading = ref(true);
const error = ref('');
const toast = ref('');
const grantBusy = ref('');
const grantUserId = ref('');
const filter = ref('all');

const selectedVariant = computed(
  () => variants.value.find((v) => v.id === selectedVariantId.value) || null,
);

const filtered = computed(() => {
  if (filter.value === 'live') {
    return attempts.value.filter((a) => a.status === 'in_progress' || a.status === 'available');
  }
  if (filter.value === 'done') {
    return attempts.value.filter((a) =>
      ['submitted', 'timed_out', 'scored', 'finished'].includes(a.status),
    );
  }
  return attempts.value;
});

const stats = computed(() => {
  const all = attempts.value;
  const done = all.filter((a) =>
    ['submitted', 'timed_out', 'scored', 'finished'].includes(a.status),
  );
  const scores = done
    .map((a) => a.score)
    .filter((s): s is number => typeof s === 'number');
  const avg = scores.length
    ? Math.round(scores.reduce((a, b) => a + b, 0) / scores.length)
    : null;
  return {
    total: all.length,
    live: all.filter((a) => a.status === 'in_progress').length,
    done: done.length,
    avg,
  };
});

function userLabel(userId: string): string {
  const u = usersById.value[userId];
  return u?.full_name || u?.login || userId.slice(0, 8);
}

function answeredCount(at: Attempt): number {
  const answers = at.answers || {};
  return Object.keys(answers).length;
}

async function loadAttempts(): Promise<void> {
  if (!selectedVariantId.value) {
    attempts.value = [];
    tickets.value = [];
    return;
  }
  const [list, ticketList] = await Promise.all([
    attemptsApi.listByVariant(selectedVariantId.value),
    ticketsApi.listByVariant(selectedVariantId.value),
  ]);
  // latest attempt per user for grid
  const byUser = new Map<string, Attempt>();
  for (const at of list) {
    const prev = byUser.get(at.user_id);
    if (!prev || at.attempt_no > prev.attempt_no) {
      byUser.set(at.user_id, at);
    }
  }
  attempts.value = [...byUser.values()].sort((a, b) =>
    userLabel(a.user_id).localeCompare(userLabel(b.user_id), 'ru'),
  );
  tickets.value = ticketList;
}

async function load(): Promise<void> {
  loading.value = true;
  error.value = '';
  try {
    const [vars, users] = await Promise.all([
      variantsApi.list(lessonId.value),
      usersApi.list(),
    ]);
    variants.value = vars;
    usersById.value = Object.fromEntries(users.map((u) => [u.id, u]));
    selectedVariantId.value =
      variantId.value ||
      vars.find((v) => v.is_primary)?.id ||
      vars[0]?.id ||
      '';
    await loadAttempts();
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    loading.value = false;
  }
}

async function grantExtra(userId: string): Promise<void> {
  if (!selectedVariantId.value) {
    return;
  }
  grantBusy.value = userId;
  error.value = '';
  toast.value = '';
  try {
    await attemptsApi.grant(userId, { variant_id: selectedVariantId.value });
    toast.value = 'Новая попытка выдана';
    await loadAttempts();
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    grantBusy.value = '';
  }
}

async function grantManual(): Promise<void> {
  if (!grantUserId.value.trim()) {
    return;
  }
  await grantExtra(grantUserId.value.trim());
  grantUserId.value = '';
}

onMounted(() => {
  void load();
});

watch(selectedVariantId, () => {
  void loadAttempts();
});
</script>

<template>
  <section class="page-sheet page-sheet--wide">
    <div class="sheet">
      <header class="sheet__header">
        <div class="sheet__header-row">
          <div>
            <nav class="curriculum-breadcrumbs" aria-label="Навигация">
              <NuxtLink :to="`/modules/${moduleId}`" class="curriculum-breadcrumbs__link">
                Модуль
              </NuxtLink>
              <span class="curriculum-breadcrumbs__sep">/</span>
              <NuxtLink
                :to="`/modules/${moduleId}/lessons/${lessonId}`"
                class="curriculum-breadcrumbs__link"
              >
                Занятие
              </NuxtLink>
              <span class="curriculum-breadcrumbs__sep">/</span>
              <span>Разбор</span>
            </nav>
            <h1 class="page-title">Разбор занятия</h1>
            <p class="page-sub">
              Сетка группы · {{ stats.total }} попыток · в работе {{ stats.live }} · сдано
              {{ stats.done }}
              <template v-if="stats.avg != null"> · средняя {{ stats.avg }}%</template>
            </p>
          </div>
          <div class="toolbar">
            <TbSelect
              v-model="selectedVariantId"
              :options="variants.map((v) => ({ value: v.id, label: v.title }))"
            />
            <TbSegmented
              v-model="filter"
              :items="[
                { id: 'all', label: 'Все' },
                { id: 'live', label: 'В работе' },
                { id: 'done', label: 'Сдали' },
              ]"
            />
          </div>
        </div>
      </header>
      <div class="sheet__rule"></div>
      <div class="sheet__body">
        <p v-if="error" class="curriculum-error">{{ error }}</p>
        <p v-if="toast" class="toast">{{ toast }}</p>
        <p v-if="loading" class="curriculum-empty">Загрузка…</p>
        <template v-else>
          <div class="grant-row">
            <TbField label="Выдать попытку по user id">
              <TbInput v-model="grantUserId" placeholder="uuid ученика" />
            </TbField>
            <TbButton
              :disabled="!grantUserId || !selectedVariantId"
              :busy="Boolean(grantBusy)"
              @click="grantManual"
            >
              Выдать
            </TbButton>
          </div>
          <p v-if="!filtered.length" class="curriculum-empty">
            Попыток по варианту «{{ selectedVariant?.title || '—' }}» нет
          </p>
          <div v-else class="grid">
            <article v-for="at in filtered" :key="at.id" class="card">
              <div class="card__head">
                <strong>{{ userLabel(at.user_id) }}</strong>
                <TbBadge
                  :tone="
                    at.status === 'in_progress'
                      ? 'info'
                      : at.score != null && at.score >= 70
                        ? 'success'
                        : 'neutral'
                  "
                >
                  {{ attemptStatusLabel(at.status) }}
                </TbBadge>
              </div>
              <ul class="card__meta">
                <li>Попытка № {{ at.attempt_no }}</li>
                <li>
                  Карты {{ answeredCount(at) }} / {{ tickets.length || '—' }}
                </li>
                <li>Оценка {{ at.score != null ? `${at.score}%` : '—' }}</li>
              </ul>
              <div class="card__actions">
                <TbButton
                  variant="secondary"
                  @click="navigateTo(`/attempts/${at.id}/review`)"
                >
                  Разбор
                </TbButton>
                <TbButton
                  variant="ghost"
                  :busy="grantBusy === at.user_id"
                  @click="grantExtra(at.user_id)"
                >
                  Ещё попытка
                </TbButton>
              </div>
            </article>
          </div>
        </template>
      </div>
    </div>
  </section>
</template>

<style scoped>
.toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
}

.grant-row {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: flex-end;
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 12px;
}

.card {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-surface);
}

.card__head {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  align-items: flex-start;
}

.card__meta {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 4px;
  color: var(--color-text-muted);
  font: 400 13px/1.4 var(--font-sans);
}

.card__actions {
  display: flex;
  gap: 8px;
  margin-top: auto;
}

.toast {
  margin: 0;
  color: var(--color-success);
  font: var(--font-mute);
}
</style>
