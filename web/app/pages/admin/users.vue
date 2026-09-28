<script setup lang="ts">
import type { UserProfile } from '~/types/groups';
import { apiErrorMessage } from '~/utils/api-error';

definePageMeta({
  middleware: ['auth', 'admin'],
});

const usersApi = useUsers();
const users = ref<UserProfile[]>([]);
const filter = ref('all');
const loading = ref(true);
const error = ref('');
const q = ref('');

const filtered = computed(() => {
  let list = users.value;
  if (filter.value !== 'all') {
    if (filter.value === 'blocked') {
      list = list.filter((u) => u.blocked);
    } else {
      list = list.filter((u) => u.role === filter.value);
    }
  }
  const query = q.value.trim().toLowerCase();
  if (query) {
    list = list.filter(
      (u) =>
        u.full_name.toLowerCase().includes(query) ||
        u.login.toLowerCase().includes(query),
    );
  }
  return list;
});

async function load(): Promise<void> {
  loading.value = true;
  try {
    users.value = await usersApi.list();
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
              <NuxtLink to="/admin" class="curriculum-breadcrumbs__link">Админ</NuxtLink>
              <span class="curriculum-breadcrumbs__sep">/</span>
              <span>Пользователи</span>
            </nav>
            <h1 class="page-title">Пользователи</h1>
            <p class="page-sub">{{ users.length }} профилей</p>
          </div>
          <TbSearch v-model="q" placeholder="Поиск" />
        </div>
      </header>
      <div class="sheet__toolbar">
        <TbSegmented
          v-model="filter"
          :items="[
            { id: 'all', label: 'Все', count: users.length },
            {
              id: 'student',
              label: 'Обучающиеся',
              count: users.filter((u) => u.role === 'student').length,
            },
            {
              id: 'teacher',
              label: 'Преподаватели',
              count: users.filter((u) => u.role === 'teacher').length,
            },
            {
              id: 'admin',
              label: 'Админы',
              count: users.filter((u) => u.role === 'admin').length,
            },
            {
              id: 'blocked',
              label: 'Блок',
              count: users.filter((u) => u.blocked).length,
            },
          ]"
        />
      </div>
      <div class="sheet__rule"></div>
      <div class="sheet__body">
        <p v-if="error" class="curriculum-error">{{ error }}</p>
        <p v-if="loading" class="curriculum-empty">Загрузка…</p>
        <div v-else class="table">
          <div class="table__head">
            <span>ФИО</span>
            <span>Логин</span>
            <span>Роль</span>
            <span>Статус</span>
          </div>
          <button
            v-for="user in filtered"
            :key="user.id"
            type="button"
            class="table__row"
            @click="navigateTo(`/users/${user.id}`)"
          >
            <span>{{ user.full_name || '—' }}</span>
            <span>{{ user.login }}</span>
            <span>{{ user.role }}</span>
            <span>{{ user.blocked ? 'Заблокирован' : 'Активен' }}</span>
          </button>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.table {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.table__head,
.table__row {
  display: grid;
  grid-template-columns: 1.4fr 1fr 1fr 1fr;
  gap: 12px;
  padding: 12px 16px;
}

.table__head {
  color: var(--color-text-muted);
  font: 500 12px/1.3 var(--font-sans);
}

.table__row {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  text-align: left;
  cursor: pointer;
  color: inherit;
  font: 500 14px/1.3 var(--font-sans);
}
</style>
