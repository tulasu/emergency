<script setup lang="ts">
import type { UserProfile } from '~/types/groups';
import { apiErrorMessage } from '~/utils/api-error';

definePageMeta({
  middleware: ['auth', 'staff'],
});

const usersApi = useUsers();
const groupsApi = useGroups();
const route = useRoute();
const userId = computed(() => String(route.params.id));

const user = ref<UserProfile | null>(null);
const groups = ref<string[]>([]);
const loading = ref(true);
const busy = ref(false);
const error = ref('');
const toast = ref('');
const role = ref('student');

async function load(): Promise<void> {
  loading.value = true;
  error.value = '';
  try {
    user.value = await usersApi.get(userId.value);
    role.value = user.value.role;
    await groupsApi.load();
    const detailed = await Promise.all(
      groupsApi.groups.value.map((g) => groupsApi.get(g.id)),
    );
    groups.value = detailed
      .filter((g) => (g.members || []).some((m) => m.user_id === userId.value))
      .map((g) => g.name);
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    loading.value = false;
  }
}

async function saveRole(): Promise<void> {
  busy.value = true;
  try {
    await usersApi.setRole(userId.value, role.value);
    toast.value = 'Роль обновлена';
    await load();
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    busy.value = false;
  }
}

async function toggleBlock(): Promise<void> {
  if (!user.value) {
    return;
  }
  busy.value = true;
  try {
    await usersApi.setBlocked(userId.value, !user.value.blocked);
    toast.value = user.value.blocked ? 'Разблокирован' : 'Заблокирован';
    await load();
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
            <h1 class="page-title">{{ user?.full_name || 'Пользователь' }}</h1>
            <p class="page-sub">
              {{ user?.login || '…' }} · {{ user?.role || '…' }}
              <template v-if="user?.blocked"> · заблокирован</template>
            </p>
          </div>
          <TbButton variant="danger" :busy="busy" @click="toggleBlock">
            {{ user?.blocked ? 'Разблокировать' : 'Заблокировать' }}
          </TbButton>
        </div>
      </header>
      <div class="sheet__rule"></div>
      <div class="sheet__body">
        <p v-if="error" class="curriculum-error">{{ error }}</p>
        <p v-if="toast" class="toast">{{ toast }}</p>
        <p v-if="loading" class="curriculum-empty">Загрузка…</p>
        <template v-else-if="user">
          <div class="stats">
            <div class="stat">
              <span>Группы</span>
              <strong>{{ groups.length ? groups.join(', ') : '—' }}</strong>
            </div>
            <div class="stat">
              <span>Статус</span>
              <strong>{{ user.blocked ? 'Заблокирован' : 'Активен' }}</strong>
            </div>
          </div>
          <div class="role-row">
            <TbField label="Роль">
              <TbSelect
                v-model="role"
                :options="[
                  { value: 'student', label: 'Обучающийся' },
                  { value: 'teacher', label: 'Преподаватель' },
                  { value: 'admin', label: 'Администратор' },
                ]"
              />
            </TbField>
            <TbButton :busy="busy" @click="saveRole">Сохранить роль</TbButton>
          </div>
          <p class="page-sub">
            История попыток открывается из разбора занятия по варианту.
          </p>
        </template>
      </div>
    </div>
  </section>
</template>

<style scoped>
.stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
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

.role-row {
  display: flex;
  gap: 12px;
  align-items: flex-end;
  flex-wrap: wrap;
}

.toast {
  margin: 0;
  color: var(--color-success);
  font: var(--font-mute);
}
</style>
