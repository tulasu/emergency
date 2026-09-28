<script setup lang="ts">
import type { Group, UserProfile } from '~/types/groups';
import { apiErrorMessage } from '~/utils/api-error';

definePageMeta({
  middleware: ['auth', 'staff'],
});

const route = useRoute();
const groupId = computed(() => String(route.params.id));
const groupsApi = useGroups();
const usersApi = useUsers();

const group = ref<Group | null>(null);
const usersById = ref<Record<string, UserProfile>>({});
const name = ref('');
const addUserId = ref('');
const addRole = ref('student');
const loading = ref(true);
const busy = ref(false);
const error = ref('');
const toast = ref('');

const roleOptions = [
  { value: 'student', label: 'Обучающийся' },
  { value: 'teacher', label: 'Преподаватель' },
];

function memberName(userId: string): string {
  return usersById.value[userId]?.full_name || usersById.value[userId]?.login || userId.slice(0, 8);
}

async function load(): Promise<void> {
  loading.value = true;
  error.value = '';
  try {
    const [g, users] = await Promise.all([groupsApi.get(groupId.value), usersApi.list()]);
    group.value = g;
    name.value = g.name;
    usersById.value = Object.fromEntries(users.map((u) => [u.id, u]));
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    loading.value = false;
  }
}

async function rename(): Promise<void> {
  busy.value = true;
  try {
    group.value = await groupsApi.rename(groupId.value, name.value.trim());
    toast.value = 'Название сохранено';
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    busy.value = false;
  }
}

async function addMember(): Promise<void> {
  if (!addUserId.value.trim()) {
    return;
  }
  busy.value = true;
  try {
    await groupsApi.addMember(groupId.value, addUserId.value.trim(), addRole.value);
    addUserId.value = '';
    toast.value = 'Участник добавлен';
    await load();
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    busy.value = false;
  }
}

async function removeMember(userId: string): Promise<void> {
  busy.value = true;
  try {
    await groupsApi.removeMember(groupId.value, userId);
    toast.value = 'Участник удалён';
    await load();
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    busy.value = false;
  }
}

async function removeGroup(): Promise<void> {
  if (!confirm('Удалить группу?')) {
    return;
  }
  busy.value = true;
  try {
    await groupsApi.remove(groupId.value);
    await navigateTo('/groups');
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
            <nav class="curriculum-breadcrumbs" aria-label="Навигация">
              <NuxtLink to="/groups" class="curriculum-breadcrumbs__link">Группы</NuxtLink>
              <span class="curriculum-breadcrumbs__sep">/</span>
              <span>{{ group?.name || '…' }}</span>
            </nav>
            <h1 class="page-title">{{ group?.name || 'Группа' }}</h1>
            <p class="page-sub">{{ group?.members?.length || 0 }} участников</p>
          </div>
          <TbButton variant="danger" :busy="busy" @click="removeGroup">Удалить</TbButton>
        </div>
      </header>
      <div class="sheet__rule"></div>
      <div class="sheet__body">
        <p v-if="error" class="curriculum-error">{{ error }}</p>
        <p v-if="toast" class="toast">{{ toast }}</p>
        <p v-if="loading" class="curriculum-empty">Загрузка…</p>
        <template v-else>
          <div class="rename">
            <TbField label="Название">
              <TbInput v-model="name" />
            </TbField>
            <TbButton :busy="busy" @click="rename">Сохранить</TbButton>
          </div>
          <div class="add">
            <TbField label="User id">
              <TbInput v-model="addUserId" placeholder="uuid" />
            </TbField>
            <TbField label="Роль">
              <TbSelect v-model="addRole" :options="roleOptions" />
            </TbField>
            <TbButton :busy="busy" @click="addMember">Добавить</TbButton>
          </div>
          <div class="members">
            <button
              v-for="m in group?.members || []"
              :key="m.user_id"
              type="button"
              class="member"
              @click="navigateTo(`/users/${m.user_id}`)"
            >
              <div>
                <strong>{{ memberName(m.user_id) }}</strong>
                <p>{{ m.role }} · {{ usersById[m.user_id]?.login || m.user_id.slice(0, 8) }}</p>
              </div>
              <TbButton variant="ghost" @click.stop="removeMember(m.user_id)">Убрать</TbButton>
            </button>
          </div>
        </template>
      </div>
    </div>
  </section>
</template>

<style scoped>
.rename,
.add {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: flex-end;
}

.members {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.member {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: center;
  width: 100%;
  padding: 12px 16px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  text-align: left;
  cursor: pointer;
  color: inherit;
  font: inherit;
}

.member p {
  margin: 4px 0 0;
  color: var(--color-text-muted);
  font: 400 12px/1.3 var(--font-sans);
}

.toast {
  margin: 0;
  color: var(--color-success);
  font: var(--font-mute);
}
</style>
