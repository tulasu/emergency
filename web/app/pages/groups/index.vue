<script setup lang="ts">
import type { Group, UserProfile } from '~/types/groups';
import { apiErrorMessage } from '~/utils/api-error';

definePageMeta({
  middleware: ['auth', 'staff'],
});

const groupsApi = useGroups();
const usersApi = useUsers();

const groups = ref<Group[]>([]);
const usersById = ref<Record<string, UserProfile>>({});
const loading = ref(true);
const error = ref('');
const toast = ref('');
const newName = ref('');
const busy = ref(false);

async function load(): Promise<void> {
  loading.value = true;
  error.value = '';
  try {
    const [list, users] = await Promise.all([groupsApi.load().then(() => groupsApi.groups.value), usersApi.list()]);
    // reload with members via get
    const detailed = await Promise.all(list.map((g) => groupsApi.get(g.id)));
    groups.value = detailed;
    usersById.value = Object.fromEntries(users.map((u) => [u.id, u]));
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    loading.value = false;
  }
}

function memberName(userId: string): string {
  return usersById.value[userId]?.full_name || usersById.value[userId]?.login || userId.slice(0, 8);
}

async function createGroup(): Promise<void> {
  if (!newName.value.trim()) {
    return;
  }
  busy.value = true;
  try {
    const g = await groupsApi.create(newName.value.trim());
    newName.value = '';
    toast.value = 'Группа создана';
    await navigateTo(`/groups/${g.id}`);
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
            <h1 class="page-title">Рабочие группы</h1>
            <p class="page-sub">{{ groups.length }} групп · состав и доступы</p>
          </div>
          <div class="create">
            <TbInput v-model="newName" placeholder="Название группы" />
            <TbButton :busy="busy" @click="createGroup">Создать</TbButton>
            <TbButton variant="secondary" @click="navigateTo('/users/new')">
              Добавить пользователей
            </TbButton>
          </div>
        </div>
      </header>
      <div class="sheet__rule"></div>
      <div class="sheet__body">
        <p v-if="error" class="curriculum-error">{{ error }}</p>
        <p v-if="toast" class="toast">{{ toast }}</p>
        <p v-if="loading" class="curriculum-empty">Загрузка…</p>
        <div v-else class="list">
          <button
            v-for="group in groups"
            :key="group.id"
            type="button"
            class="row"
            @click="navigateTo(`/groups/${group.id}`)"
          >
            <div>
              <strong>{{ group.name }}</strong>
              <p>
                {{ group.members?.length || 0 }} участников ·
                {{
                  (group.members || [])
                    .filter((m) => m.role === 'teacher')
                    .map((m) => memberName(m.user_id))
                    .join(', ') || 'без преподавателя'
                }}
              </p>
            </div>
            <span class="link">Открыть</span>
          </button>
          <p v-if="!groups.length" class="curriculum-empty">Групп пока нет</p>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.create {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.row {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: center;
  width: 100%;
  padding: 16px 20px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  text-align: left;
  cursor: pointer;
  color: inherit;
  font: inherit;
}

.row strong {
  font: 700 16px/1.3 var(--font-sans);
}

.row p {
  margin: 4px 0 0;
  color: var(--color-text-muted);
  font: 400 13px/1.4 var(--font-sans);
}

.link {
  color: var(--color-primary);
  font: 500 13px/1.3 var(--font-sans);
}

.toast {
  margin: 0;
  color: var(--color-success);
  font: var(--font-mute);
}
</style>
