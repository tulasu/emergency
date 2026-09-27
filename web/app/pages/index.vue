<script setup lang="ts">
import type { AssignedLesson, AssignedModule } from '~/types/curriculum';
import { apiErrorMessage } from '~/utils/api-error';
import { attemptStatusLabel, isOpenAttemptStatus, lessonCountLabel } from '~/utils/curriculum-labels';
import type { TbBadgeTone, TbIconName } from '~/types/ui';

definePageMeta({
  middleware: 'auth',
});

const auth = useAuthStore();
const config = useRuntimeConfig();
const companyName = config.public.companyName;
const attemptsApi = useAttempts();

const isStaff = computed(
  () => auth.user?.role === 'admin' || auth.user?.role === 'teacher',
);

const modules = ref<AssignedModule[]>([]);
const filter = ref('all');
const loading = ref(false);
const error = ref('');

const greetingName = computed(() => {
  const full = auth.user?.full_name?.trim();
  if (!full) {
    return auth.user?.login || '';
  }
  return full.split(/\s+/)[0] || full;
});

const activeLesson = computed(() => {
  for (const mod of modules.value) {
    for (const lesson of mod.lessons) {
      if (isOpenAttemptStatus(lesson.status)) {
        return { module: mod, lesson };
      }
    }
  }
  return null;
});

const filteredModules = computed(() => {
  if (filter.value === 'progress') {
    return modules.value.filter((m) =>
      m.lessons.some((l) => isOpenAttemptStatus(l.status)),
    );
  }
  if (filter.value === 'done') {
    return modules.value.filter(
      (m) =>
        m.lessons.length > 0 &&
        m.lessons.every((l) =>
          ['submitted', 'scored', 'finished'].includes(l.status || ''),
        ),
    );
  }
  return modules.value;
});

const filterItems = computed(() => [
  { id: 'all', label: 'Все', count: modules.value.length },
  {
    id: 'progress',
    label: 'В процессе',
    count: modules.value.filter((m) =>
      m.lessons.some((l) => isOpenAttemptStatus(l.status)),
    ).length,
  },
  {
    id: 'done',
    label: 'Сданы',
    count: modules.value.filter(
      (m) =>
        m.lessons.length > 0 &&
        m.lessons.every((l) =>
          ['submitted', 'scored', 'finished'].includes(l.status || ''),
        ),
    ).length,
  },
]);

function progress(mod: AssignedModule): { done: number; total: number } {
  const total = mod.lessons.length;
  const done = mod.lessons.filter((l) =>
    ['submitted', 'scored', 'finished'].includes(l.status || ''),
  ).length;
  return { done, total };
}

function avgScore(mod: AssignedModule): number | null {
  const scores = mod.lessons
    .map((l) => l.score)
    .filter((s): s is number => typeof s === 'number');
  if (!scores.length) {
    return null;
  }
  return Math.round(scores.reduce((a, b) => a + b, 0) / scores.length);
}

function openLabel(mod: AssignedModule): string {
  const open = mod.lessons.find((l) => isOpenAttemptStatus(l.status));
  if (open) {
    return `${lessonCountLabel(mod.lessons.length)} · открыто занятие «${open.lesson.title}»`;
  }
  if (mod.lessons.some((l) => l.status)) {
    return lessonCountLabel(mod.lessons.length);
  }
  return 'занятия ещё не открыты';
}

function moduleState(mod: AssignedModule): {
  tone: 'done' | 'active' | 'locked';
  icon: TbIconName;
  status: string;
  badgeTone: TbBadgeTone;
  action: string;
  actionVariant: 'primary' | 'secondary';
  deadline: string;
} {
  const p = progress(mod);
  const open = mod.lessons.find((l) => isOpenAttemptStatus(l.status));
  const deadlineLesson =
    open ||
    mod.lessons.find((l) => l.deadline_at) ||
    mod.lessons[mod.lessons.length - 1];

  if (p.total > 0 && p.done === p.total) {
    return {
      tone: 'done',
      icon: 'check',
      status: 'Завершён',
      badgeTone: 'success',
      action: 'Разбор ошибок',
      actionVariant: 'secondary',
      deadline: 'сдан',
    };
  }
  if (open) {
    return {
      tone: 'active',
      icon: 'play',
      status: attemptStatusLabel(open.status),
      badgeTone: 'info',
      action: 'Открыть модуль',
      actionVariant: 'primary',
      deadline: formatDeadline(deadlineLesson?.deadline_at) || '—',
    };
  }
  if (!mod.lessons.some((l) => l.status)) {
    return {
      tone: 'locked',
      icon: 'lock',
      status: 'Не начат',
      badgeTone: 'neutral',
      action: 'Ждёт открытия',
      actionVariant: 'secondary',
      deadline: '—',
    };
  }
  return {
    tone: 'locked',
    icon: 'folder',
    status: 'Не начат',
    badgeTone: 'neutral',
    action: 'Открыть модуль',
    actionVariant: 'secondary',
    deadline: formatDeadline(deadlineLesson?.deadline_at) || '—',
  };
}

function formatDeadline(value?: string): string {
  if (!value) {
    return '';
  }
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) {
    return '';
  }
  const dd = String(d.getDate()).padStart(2, '0');
  const mm = String(d.getMonth() + 1).padStart(2, '0');
  const hh = String(d.getHours()).padStart(2, '0');
  const mi = String(d.getMinutes()).padStart(2, '0');
  return `до ${dd}.${mm}, ${hh}:${mi}`;
}

function formatShort(value?: string): string {
  if (!value) {
    return '';
  }
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) {
    return '';
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
    return 'дедлайн прошёл';
  }
  if (diff === 0) {
    return 'до дедлайна сегодня';
  }
  if (diff === 1) {
    return 'до дедлайна 1 день';
  }
  return `до дедлайна ${diff} дн.`;
}

function heroMeta(item: { module: AssignedModule; lesson: AssignedLesson }): string {
  const parts = [
    item.module.module.title,
    item.lesson.status === 'in_progress' ? 'в работе' : 'попытка',
    `вариант ${item.lesson.variant.title}`,
  ];
  if (item.lesson.variant.is_primary) {
    parts.push('основной');
  }
  return parts.join(' · ');
}

function heroNote(item: { module: AssignedModule; lesson: AssignedLesson }): string {
  const parts: string[] = [];
  if (item.lesson.available_from) {
    parts.push(`Доступно с ${formatShort(item.lesson.available_from)}`);
  }
  if (item.lesson.deadline_at) {
    parts.push(daysLeft(item.lesson.deadline_at));
  }
  return parts.join(' · ') || 'Открыто преподавателем';
}

async function loadStudentModules(): Promise<void> {
  if (isStaff.value) {
    return;
  }
  loading.value = true;
  error.value = '';
  try {
    const list = await attemptsApi.listMyModules();
    modules.value = await Promise.all(
      list.map(async (mod) => ({
        ...mod,
        lessons: await Promise.all(
          mod.lessons.map(async (lesson) => {
            try {
              const mine = await attemptsApi.listMine(lesson.variant.id);
              const arr = Array.isArray(mine) ? mine : mine ? [mine] : [];
              const open =
                arr.find((a) => isOpenAttemptStatus(a.status)) || arr[0];
              if (!open) {
                return lesson;
              }
              return {
                ...lesson,
                attempt_id: open.id,
                status: open.status,
                available_from: open.available_from,
                deadline_at: open.deadline_at,
                score: open.score,
              };
            } catch {
              return lesson;
            }
          }),
        ),
      })),
    );
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    loading.value = false;
  }
}

async function startFromHero(item: {
  module: AssignedModule;
  lesson: AssignedLesson;
}): Promise<void> {
  await navigateTo(`/learn/modules/${item.module.module.id}`);
}

onMounted(() => {
  if (isStaff.value) {
    void navigateTo('/modules');
    return;
  }
  void loadStudentModules();
});
</script>

<template>
  <section v-if="isStaff" class="home">
    <h1 class="page-title">{{ companyName }}</h1>
    <p class="page-sub">Перенаправление в модули…</p>
  </section>

  <section v-else class="student-home">
    <header class="student-home__header">
      <h1 class="page-title">Добрый день, {{ greetingName }}</h1>
      <p class="page-sub">Назначено {{ modules.length }} модулей</p>
    </header>

    <p v-if="error" class="curriculum-error">{{ error }}</p>
    <p v-if="loading" class="curriculum-empty">Загрузка…</p>

    <div v-if="activeLesson" class="curriculum-hero">
      <div class="curriculum-hero__head">
        <span
          v-if="activeLesson.lesson.deadline_at"
          class="curriculum-hero__deadline"
        >
          <TbIcon name="calendar" />
          Дедлайн {{ formatShort(activeLesson.lesson.deadline_at) }}
        </span>
        <span v-else></span>
        <p class="curriculum-hero__eyebrow">Можно приступить</p>
      </div>
      <div class="curriculum-hero__body">
        <h2 class="curriculum-hero__title">
          {{ activeLesson.lesson.lesson.title }}
        </h2>
        <p class="curriculum-hero__meta">{{ heroMeta(activeLesson) }}</p>
      </div>
      <div class="curriculum-hero__foot">
        <p class="curriculum-hero__note">
          {{ heroNote(activeLesson) }}
        </p>
        <div class="curriculum-hero__actions">
          <TbButton class="curriculum-hero__cta" @click="startFromHero(activeLesson)">
            Приступить
          </TbButton>
        </div>
      </div>
    </div>

    <div class="curriculum-surface modules-panel">
      <div class="modules-head">
        <h2 class="sheet__section-title">Мои модули</h2>
        <TbSegmented v-model="filter" :items="filterItems" />
      </div>

      <p v-if="!loading && !filteredModules.length" class="curriculum-empty">
        Пока нет назначенных модулей
      </p>
      <div v-else class="module-list">
        <button
          v-for="item in filteredModules"
          :key="item.module.id"
          type="button"
          class="module-row"
          :class="`module-row--${moduleState(item).tone}`"
          @click="navigateTo(`/learn/modules/${item.module.id}`)"
        >
          <span class="module-row__icon" aria-hidden="true">
            <TbIcon :name="moduleState(item).icon" />
          </span>
          <div class="module-row__text">
            <h3>{{ item.module.title }}</h3>
            <p>{{ openLabel(item) }}</p>
          </div>
          <div class="module-row__progress">
            <div class="bar">
              <span
                :class="{ 'bar__fill--done': moduleState(item).tone === 'done' }"
                :style="{
                  width: `${
                    progress(item).total
                      ? Math.round((progress(item).done / progress(item).total) * 100)
                      : 0
                  }%`,
                }"
              ></span>
            </div>
            <span>{{ progress(item).done }}/{{ progress(item).total }}</span>
          </div>
          <div class="module-row__score">
            <span>Моя успешность</span>
            <strong>{{ avgScore(item) != null ? `${avgScore(item)}%` : '—' }}</strong>
          </div>
          <span class="module-row__deadline">{{ moduleState(item).deadline }}</span>
          <TbBadge :tone="moduleState(item).badgeTone">
            {{ moduleState(item).status }}
          </TbBadge>
          <TbButton
            :variant="moduleState(item).actionVariant"
            @click.stop="navigateTo(`/learn/modules/${item.module.id}`)"
          >
            {{ moduleState(item).action }}
          </TbButton>
        </button>
      </div>
    </div>
  </section>
</template>

<style scoped>
.home {
  padding: 24px 8px;
}

.student-home {
  display: flex;
  flex-direction: column;
  gap: 24px;
  width: 100%;
  min-height: 100%;
}

.student-home__header {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.modules-panel {
  gap: 8px;
  padding: 24px;
}

.modules-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
  padding: 0 0 12px;
}

.module-list {
  display: flex;
  flex-direction: column;
}

.module-row {
  display: grid;
  grid-template-columns: 40px minmax(180px, 1.4fr) 200px 110px 140px 140px auto;
  gap: 16px;
  align-items: center;
  width: 100%;
  padding: 14px 0;
  border: 0;
  border-bottom: 1px solid var(--color-border);
  background: transparent;
  text-align: left;
  cursor: pointer;
  color: inherit;
  font: inherit;
}

.module-row:last-child {
  border-bottom: 0;
}

.module-row__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: var(--radius-md);
  background: var(--color-bg);
  color: var(--color-text-muted);
}

.module-row--done .module-row__icon {
  background: var(--color-success-soft);
  color: var(--color-success);
}

.module-row--active .module-row__icon {
  background: var(--color-secondary);
  color: var(--color-primary);
}

.module-row__text h3 {
  margin: 0 0 3px;
  font: 700 15px/1.3 var(--font-sans);
}

.module-row__text p {
  margin: 0;
  color: var(--color-text-muted);
  font: var(--font-mute);
}

.module-row__progress {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--color-text);
  font: 500 12px/1.3 var(--font-sans);
}

.bar {
  flex: 1;
  height: 6px;
  border-radius: 999px;
  background: var(--color-bg);
  overflow: hidden;
  min-width: 80px;
}

.bar > span {
  display: block;
  height: 100%;
  background: var(--color-primary);
}

.bar__fill--done {
  background: var(--color-success);
}

.module-row__score {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.module-row__score span {
  color: var(--color-text-muted);
  font: 500 11px/1.3 var(--font-sans);
}

.module-row__score strong {
  font: 700 15px/1.3 var(--font-sans);
}

.module-row__deadline {
  color: var(--color-text-muted);
  font: 500 12px/1.3 var(--font-sans);
}

.module-row :deep(.tb-btn) {
  height: 36px;
  padding: 0 16px;
  font: 500 13px/1.3 var(--font-sans);
}

@media (max-width: 1100px) {
  .module-row {
    grid-template-columns: 40px 1fr auto;
    grid-template-areas:
      'icon text action'
      'icon progress action'
      'icon meta action';
  }

  .module-row__icon {
    grid-area: icon;
  }

  .module-row__text {
    grid-area: text;
  }

  .module-row__progress {
    grid-area: progress;
  }

  .module-row__score,
  .module-row__deadline,
  .module-row :deep(.tb-badge) {
    display: none;
  }

  .module-row :deep(.tb-btn) {
    grid-area: action;
  }
}
</style>
