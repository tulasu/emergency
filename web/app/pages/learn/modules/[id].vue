<script setup lang="ts">
import type { AssignedLesson, AssignedModule } from '~/types/curriculum';
import { apiErrorMessage } from '~/utils/api-error';
import { attemptStatusLabel, isOpenAttemptStatus } from '~/utils/curriculum-labels';
import type { TbBadgeTone, TbIconName } from '~/types/ui';

definePageMeta({
  middleware: 'auth',
});

const route = useRoute();
const moduleId = computed(() => String(route.params.id));
const attemptsApi = useAttempts();

const assigned = ref<AssignedModule | null>(null);
const selected = ref<AssignedLesson | null>(null);
const loading = ref(true);
const busyId = ref('');
const error = ref('');
const toast = ref('');

const progress = computed(() => {
  const lessons = assigned.value?.lessons || [];
  const done = lessons.filter((l) =>
    ['submitted', 'scored', 'finished'].includes(l.status || ''),
  ).length;
  return { done, total: lessons.length };
});

const avgScore = computed(() => {
  const scores = (assigned.value?.lessons || [])
    .map((l) => l.score)
    .filter((s): s is number => typeof s === 'number');
  if (!scores.length) {
    return null;
  }
  return Math.round(scores.reduce((a, b) => a + b, 0) / scores.length);
});

function formatDateTime(value?: string): string {
  if (!value) {
    return '—';
  }
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) {
    return '—';
  }
  const dd = String(d.getDate()).padStart(2, '0');
  const mm = String(d.getMonth() + 1).padStart(2, '0');
  const yyyy = d.getFullYear();
  const hh = String(d.getHours()).padStart(2, '0');
  const mi = String(d.getMinutes()).padStart(2, '0');
  return `${dd}.${mm}.${yyyy}, ${hh}:${mi}`;
}

function formatShort(value?: string): string {
  if (!value) {
    return '—';
  }
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) {
    return '—';
  }
  const dd = String(d.getDate()).padStart(2, '0');
  const mm = String(d.getMonth() + 1).padStart(2, '0');
  const hh = String(d.getHours()).padStart(2, '0');
  const mi = String(d.getMinutes()).padStart(2, '0');
  return `${dd}.${mm}, ${hh}:${mi}`;
}

function daysLeft(value?: string): string {
  if (!value) {
    return '';
  }
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) {
    return '';
  }
  const diff = Math.ceil((d.getTime() - Date.now()) / (1000 * 60 * 60 * 24));
  if (diff < 0) {
    return 'прошёл';
  }
  if (diff === 0) {
    return 'сегодня';
  }
  if (diff === 1) {
    return 'остался 1 день';
  }
  return `осталось ${diff} дн.`;
}

function durationLabel(seconds?: number): string {
  if (!seconds) {
    return '—';
  }
  const mins = Math.round(seconds / 60);
  return `${mins} мин`;
}

function lessonVisual(item: AssignedLesson): {
  icon: TbIconName;
  iconTone: string;
  badgeTone: TbBadgeTone;
  status: string;
  action: string;
  actionVariant: 'primary' | 'secondary' | 'ghost';
  showAction: boolean;
} {
  if (['submitted', 'scored', 'finished'].includes(item.status || '')) {
    return {
      icon: 'check',
      iconTone: 'success',
      badgeTone: 'success',
      status: 'Завершено',
      action: 'Результат',
      actionVariant: 'secondary',
      showAction: true,
    };
  }
  if (isOpenAttemptStatus(item.status)) {
    return {
      icon: 'play',
      iconTone: 'accent',
      badgeTone: 'info',
      status: attemptStatusLabel(item.status),
      action: item.status === 'in_progress' ? 'Продолжить' : 'Приступить',
      actionVariant: 'primary',
      showAction: true,
    };
  }
  if (item.available_from) {
    return {
      icon: 'clock',
      iconTone: 'warning',
      badgeTone: 'warning',
      status: 'Запланировано',
      action: '',
      actionVariant: 'ghost',
      showAction: false,
    };
  }
  return {
    icon: 'lock',
    iconTone: 'neutral',
    badgeTone: 'neutral',
    status: 'Не открыто',
    action: '',
    actionVariant: 'ghost',
    showAction: false,
  };
}

function lessonMeta(item: AssignedLesson): string {
  if (!item.status) {
    return 'Преподаватель ещё не открыл занятие';
  }
  const parts = [`Вариант ${item.variant.title}`];
  if (item.variant.is_primary) {
    parts.unshift('Основной вариант');
  }
  if (item.lesson.duration_seconds) {
    parts.push(`норматив ${durationLabel(item.lesson.duration_seconds)}`);
  }
  if (item.score != null) {
    parts.push(`результат ${Math.round(item.score)}%`);
  }
  return parts.join(' · ');
}

function timeline(item: AssignedLesson): Array<{ label: string; meta: string; active?: boolean }> {
  return [
    {
      label: 'Модуль назначен',
      meta: assigned.value ? 'вам доступен' : '—',
    },
    {
      label: item.status ? 'Занятие открыто' : 'Ожидает открытия',
      meta: item.available_from ? formatDateTime(item.available_from) : '—',
      active: Boolean(item.status),
    },
    {
      label: attemptStatusLabel(item.status),
      meta: item.deadline_at ? `до ${formatDateTime(item.deadline_at)}` : '—',
      active: isOpenAttemptStatus(item.status),
    },
    {
      label: 'Дедлайн',
      meta: item.deadline_at
        ? `${formatDateTime(item.deadline_at)}${daysLeft(item.deadline_at) ? ` · ${daysLeft(item.deadline_at)}` : ''}`
        : 'не задан',
    },
  ];
}

async function enrichLesson(item: AssignedLesson): Promise<AssignedLesson> {
  try {
    const mine = await attemptsApi.listMine(item.variant.id);
    const arr = Array.isArray(mine) ? mine : mine ? [mine] : [];
    const open =
      arr.find((a) => isOpenAttemptStatus(a.status)) ||
      arr[0];
    if (!open) {
      return item;
    }
    return {
      ...item,
      attempt_id: open.id,
      status: open.status,
      available_from: open.available_from,
      deadline_at: open.deadline_at,
      score: open.score,
    };
  } catch {
    return item;
  }
}

async function load(): Promise<void> {
  loading.value = true;
  error.value = '';
  try {
    const list = await attemptsApi.listMyModules();
    const found = list.find((item) => item.module.id === moduleId.value) ?? null;
    if (!found) {
      assigned.value = null;
      error.value = 'Модуль не найден или не назначен';
      selected.value = null;
      return;
    }
    found.lessons = await Promise.all(found.lessons.map((l) => enrichLesson(l)));
    assigned.value = found;
    selected.value =
      found.lessons.find((l) => isOpenAttemptStatus(l.status)) ||
      found.lessons[0] ||
      null;
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    loading.value = false;
  }
}

async function startLesson(item: AssignedLesson): Promise<void> {
  busyId.value = item.variant.id;
  error.value = '';
  toast.value = '';
  try {
    let id = item.attempt_id;
    if (!id) {
      const mine = await attemptsApi.listMine(item.variant.id);
      const open = mine.find((a) => isOpenAttemptStatus(a.status));
      id = open?.id;
    }
    if (!id) {
      error.value = 'Попытка ещё не выдана. Дождитесь открытия занятия преподавателем.';
      return;
    }
    const attempt = await attemptsApi.start(id);
    toast.value = `Попытка № ${attempt.attempt_no} начата`;
    await load();
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    busyId.value = '';
  }
}

onMounted(() => {
  void load();
});
</script>

<template>
  <section class="page-sheet page-sheet--wide">
    <div class="sheet">
      <header class="sheet__header module-header">
        <div class="module-header__row">
          <div>
            <nav class="curriculum-breadcrumbs" aria-label="Навигация">
              <NuxtLink to="/" class="curriculum-breadcrumbs__link">Главная</NuxtLink>
              <span class="curriculum-breadcrumbs__sep">/</span>
              <span>{{ assigned?.module.title || 'Модуль' }}</span>
            </nav>
            <h1 class="page-title">{{ assigned?.module.title || 'Модуль' }}</h1>
            <p class="page-sub">
              порог допуска {{ assigned?.module.success_threshold ?? '—' }}%
            </p>
          </div>
          <div class="module-header__stats">
            <div class="curriculum-stat">
              <span class="curriculum-stat__label">Пройдено занятий</span>
              <strong class="curriculum-stat__value">
                {{ progress.done }} / {{ progress.total }}
              </strong>
            </div>
            <div class="curriculum-stat">
              <span class="curriculum-stat__label">Моя успешность</span>
              <strong class="curriculum-stat__value">
                {{ avgScore != null ? `${avgScore}%` : '—' }}
              </strong>
            </div>
          </div>
        </div>
      </header>
      <div class="sheet__body">
        <p v-if="error" class="curriculum-error">{{ error }}</p>
        <p v-if="toast" class="toast">{{ toast }}</p>
        <p v-if="loading" class="curriculum-empty">Загрузка…</p>
        <div v-else-if="assigned" class="curriculum-layout student-module-layout">
          <div class="lessons-panel">
            <div class="lessons-panel__head">
              <h2 class="sheet__section-title">Занятия модуля</h2>
              <p class="page-sub">
                Занятия открывает преподаватель — по одному или несколько сразу
              </p>
            </div>
            <p v-if="!assigned.lessons.length" class="curriculum-empty">Занятий нет</p>
            <button
              v-for="item in assigned.lessons"
              :key="item.lesson.id"
              type="button"
              class="lesson-row"
              :class="{ 'lesson-row--active': selected?.lesson.id === item.lesson.id }"
              @click="selected = item"
            >
              <span
                class="lesson-row__icon"
                :class="`lesson-row__icon--${lessonVisual(item).iconTone}`"
                aria-hidden="true"
              >
                <TbIcon :name="lessonVisual(item).icon" />
              </span>
              <div class="lesson-row__text">
                <strong>{{ item.lesson.title }}</strong>
                <p>{{ lessonMeta(item) }}</p>
              </div>
              <div class="lesson-row__deadline">
                <span>Дедлайн</span>
                <strong>{{ formatShort(item.deadline_at) }}</strong>
              </div>
              <TbBadge :tone="lessonVisual(item).badgeTone">
                {{ lessonVisual(item).status }}
              </TbBadge>
              <TbButton
                v-if="lessonVisual(item).showAction"
                :variant="lessonVisual(item).actionVariant"
                :busy="busyId === item.variant.id"
                @click.stop="startLesson(item)"
              >
                {{ lessonVisual(item).action }}
              </TbButton>
              <span v-else class="lesson-row__wait">
                <TbIcon name="lock" />
                {{
                  item.available_from
                    ? `Откроется ${formatShort(item.available_from)}`
                    : 'Ждёт открытия'
                }}
              </span>
            </button>
          </div>
          <aside v-if="selected" class="side-panel attempt-panel">
            <div class="attempt-panel__head">
              <div class="attempt-panel__kicker">
                <p class="side-panel__title">Текущая попытка</p>
                <TbBadge :tone="lessonVisual(selected).badgeTone">
                  {{ lessonVisual(selected).status }}
                </TbBadge>
              </div>
              <h3>{{ selected.lesson.title }}</h3>
              <p class="page-sub">Вариант {{ selected.variant.title }}</p>
            </div>
            <ul class="side-panel__list attempt-fields">
              <li class="side-panel__row">
                <span class="attempt-field">
                  <TbIcon name="check" />
                  Основной вариант
                </span>
                <strong>{{ selected.variant.is_primary ? 'да' : 'нет' }}</strong>
              </li>
              <li class="side-panel__row">
                <span class="attempt-field">
                  <TbIcon name="calendar" />
                  Можно приступить с
                </span>
                <strong>{{ formatDateTime(selected.available_from) }}</strong>
              </li>
              <li class="side-panel__row">
                <span class="attempt-field">
                  <TbIcon name="clock" />
                  Дедлайн
                </span>
                <strong :class="{ 'text-warning': selected.deadline_at }">
                  {{
                    selected.deadline_at
                      ? `${formatDateTime(selected.deadline_at)}${
                          daysLeft(selected.deadline_at)
                            ? ` (${daysLeft(selected.deadline_at)})`
                            : ''
                        }`
                      : '—'
                  }}
                </strong>
              </li>
              <li class="side-panel__row">
                <span class="attempt-field">
                  <TbIcon name="info" />
                  Норматив
                </span>
                <strong>{{ durationLabel(selected.lesson.duration_seconds) }}</strong>
              </li>
            </ul>
            <div class="timeline">
              <p class="timeline__title">Как появилась попытка</p>
              <ol class="timeline__list">
                <li
                  v-for="(event, index) in timeline(selected)"
                  :key="`${event.label}-${index}`"
                  class="timeline__item"
                  :class="{ 'timeline__item--active': event.active }"
                >
                  <span class="timeline__dot"></span>
                  <div>
                    <strong>{{ event.label }}</strong>
                    <p>{{ event.meta }}</p>
                  </div>
                </li>
              </ol>
            </div>
            <div class="attempt-panel__actions">
              <TbButton
                width="block"
                :busy="busyId === selected.variant.id"
                :disabled="
                  !selected.status ||
                  ['submitted', 'scored', 'finished'].includes(selected.status)
                "
                @click="startLesson(selected)"
              >
                {{
                  selected.status === 'in_progress'
                    ? 'Продолжить попытку'
                    : selected.status
                      ? 'Приступить к попытке'
                      : 'Ещё не открыто'
                }}
              </TbButton>
              <p class="attempt-panel__note">
                Таймер норматива запустится после нажатия. Задания выдаются из выбранного
                варианта.
              </p>
            </div>
          </aside>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.module-header__row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
  flex-wrap: wrap;
}

.module-header__stats {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.student-module-layout {
  grid-template-columns: minmax(0, 1fr) 440px;
}

.lessons-panel {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 24px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}

.lessons-panel__head {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding-bottom: 12px;
}

.lesson-row {
  display: grid;
  grid-template-columns: 40px minmax(160px, 1.4fr) 120px 150px auto;
  gap: 16px;
  align-items: center;
  width: 100%;
  padding: 16px;
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  background: transparent;
  text-align: left;
  cursor: pointer;
  color: inherit;
  font: inherit;
}

.lesson-row--active {
  background: var(--color-secondary);
  border-color: color-mix(in srgb, var(--color-primary) 25%, transparent);
}

.lesson-row__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: var(--radius-md);
  background: var(--color-bg);
  color: var(--color-text-muted);
}

.lesson-row__icon--success {
  background: var(--color-success-soft);
  color: var(--color-success);
}

.lesson-row__icon--accent {
  background: var(--color-primary);
  color: #fff;
}

.lesson-row__icon--warning {
  background: var(--color-warning-soft);
  color: var(--color-warning);
}

.lesson-row__icon--neutral {
  background: var(--color-bg);
  color: var(--color-text-subtle);
}

.lesson-row__text strong {
  display: block;
  font: 700 15px/1.3 var(--font-sans);
}

.lesson-row__text p {
  margin: 4px 0 0;
  color: var(--color-text-muted);
  font: 400 12px/1.4 var(--font-sans);
}

.lesson-row__deadline {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.lesson-row__deadline span {
  color: var(--color-text-muted);
  font: 500 11px/1.3 var(--font-sans);
}

.lesson-row__deadline strong {
  font: 500 14px/1.3 var(--font-sans);
}

.lesson-row__wait {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--color-text-muted);
  font: 500 12px/1.3 var(--font-sans);
}

.lesson-row :deep(.tb-btn) {
  height: 36px;
  padding: 0 16px;
  font: 500 13px/1.3 var(--font-sans);
}

.attempt-panel {
  min-height: 100%;
  gap: 20px;
}

.attempt-panel__head h3 {
  margin: 8px 0 4px;
  font: 700 20px/1.3 var(--font-sans);
}

.attempt-panel__kicker {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.attempt-field {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: var(--color-text-muted);
}

.attempt-fields .side-panel__row {
  align-items: flex-start;
  padding: 10px 0;
  border-bottom: 1px solid var(--color-border);
}

.attempt-fields .side-panel__row:last-child {
  border-bottom: 0;
}

.text-warning {
  color: var(--color-warning);
}

.timeline__title {
  margin: 0 0 12px;
  font: 700 14px/1.3 var(--font-sans);
}

.timeline__list {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
}

.timeline__item {
  display: grid;
  grid-template-columns: 14px 1fr;
  gap: 12px;
  position: relative;
  padding-bottom: 16px;
}

.timeline__item:not(:last-child)::before {
  content: '';
  position: absolute;
  left: 6px;
  top: 14px;
  bottom: 0;
  width: 1px;
  background: var(--color-border);
}

.timeline__dot {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  border: 2px solid var(--color-border);
  background: var(--color-surface);
  margin-top: 2px;
}

.timeline__item--active .timeline__dot {
  border-color: var(--color-primary);
  background: var(--color-primary);
}

.timeline__item strong {
  display: block;
  font: 500 13px/1.3 var(--font-sans);
}

.timeline__item p {
  margin: 2px 0 0;
  color: var(--color-text-muted);
  font: 400 12px/1.3 var(--font-sans);
}

.attempt-panel__actions {
  margin-top: auto;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.attempt-panel__note {
  margin: 0;
  color: var(--color-text-subtle);
  font: 400 12px/1.4 var(--font-sans);
}

.toast {
  margin: 0;
  color: var(--color-success);
  font: var(--font-mute);
}

@media (max-width: 1100px) {
  .student-module-layout {
    grid-template-columns: 1fr;
  }

  .lesson-row {
    grid-template-columns: 40px 1fr auto;
  }

  .lesson-row__deadline {
    display: none;
  }
}
</style>
