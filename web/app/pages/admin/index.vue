<script setup lang="ts">
import type { Group, UserProfile } from '~/types/groups';
import { apiErrorMessage } from '~/utils/api-error';

definePageMeta({
  middleware: ['auth', 'admin'],
});

const usersApi = useUsers();
const groupsApi = useGroups();
const { $api } = useNuxtApp();

const users = ref<UserProfile[]>([]);
const groups = ref<Group[]>([]);
const health = ref<{ status?: string; version?: string } | null>(null);
const loading = ref(true);
const error = ref('');

async function load(): Promise<void> {
  loading.value = true;
  error.value = '';
  try {
    const [userList, groupList, healthRes] = await Promise.all([
      usersApi.list(),
      groupsApi.load().then(() => groupsApi.groups.value),
      $api<{ status?: string; version?: string }>('/health').catch(() => ({ status: 'ok' })),
    ]);
    users.value = userList;
    groups.value = groupList;
    health.value = healthRes;
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    loading.value = false;
  }
}

const stats = computed(() => ({
  users: users.value.length,
  students: users.value.filter((u) => u.role === 'student').length,
  teachers: users.value.filter((u) => u.role === 'teacher').length,
  blocked: users.value.filter((u) => u.blocked).length,
  groups: groups.value.length,
  activeToday: users.value.filter((u) => !u.blocked).length,
}));

onMounted(() => {
  void load();
});
</script>

<template>
  <section class="page-sheet">
    <div class="sheet">
      <header class="sheet__header">
        <h1 class="page-title">Панель администратора</h1>
        <p class="page-sub">
          {{ health?.status === 'ok' || health ? 'Системы доступны' : 'Проверка…' }}
          <template v-if="health?.version"> · версия {{ health.version }}</template>
        </p>
      </header>
      <div class="sheet__rule"></div>
      <div class="sheet__body">
        <p v-if="error" class="curriculum-error">{{ error }}</p>
        <p v-if="loading" class="curriculum-empty">Загрузка…</p>
        <template v-else>
          <div class="stats">
            <div class="stat"><span>Пользователи</span><strong>{{ stats.users }}</strong></div>
            <div class="stat"><span>Обучающиеся</span><strong>{{ stats.students }}</strong></div>
            <div class="stat"><span>Преподаватели</span><strong>{{ stats.teachers }}</strong></div>
            <div class="stat"><span>Группы</span><strong>{{ stats.groups }}</strong></div>
            <div class="stat"><span>Заблокированы</span><strong>{{ stats.blocked }}</strong></div>
          </div>
          <div class="links">
            <NuxtLink class="link-card" to="/admin/users">Пользователи</NuxtLink>
            <NuxtLink class="link-card" to="/admin/groups">Группы и права</NuxtLink>
            <NuxtLink class="link-card" to="/admin/audit">Журнал аудита</NuxtLink>
            <NuxtLink class="link-card" to="/admin/system">Состояние системы</NuxtLink>
          </div>
        </template>
      </div>
    </div>
  </section>
</template>

<style scoped>
.stats {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 12px;
}

.stat {
  padding: 16px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.stat span {
  color: var(--color-text-muted);
  font: 500 12px/1.3 var(--font-sans);
}

.stat strong {
  font: 700 24px/1.1 var(--font-sans);
}

.links {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.link-card {
  padding: 20px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  text-decoration: none;
  color: inherit;
  font: 700 16px/1.3 var(--font-sans);
}

.link-card:hover {
  border-color: color-mix(in srgb, var(--color-primary) 35%, transparent);
}

@media (max-width: 900px) {
  .stats,
  .links {
    grid-template-columns: 1fr 1fr;
  }
}
</style>
