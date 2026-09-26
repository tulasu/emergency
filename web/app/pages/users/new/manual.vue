<script setup lang="ts">
import type { ProvisionResult } from '~/types/auth';
import { emptyRow, filledUsers, type UserRow } from '~/utils/users-form';

definePageMeta({
  middleware: ['auth', 'staff'],
});

const { $api } = useNuxtApp();
const results = useProvisionResultStore();
const { groupOptions, load, groupName } = useGroups();

const groupId = ref('');
const users = ref<UserRow[]>([emptyRow()]);
const pending = ref(false);
const error = ref('');

onMounted(async () => {
  await load();
});

const countLabel = computed(() =>
  ruCount(users.value.length, 'учётная запись', 'учётные записи', 'учётных записей'),
);

function addRow(): void {
  users.value.push(emptyRow());
}

function removeRow(index: number): void {
  users.value.splice(index, 1);
  if (users.value.length === 0) {
    addRow();
  }
}

async function submit(): Promise<void> {
  const filled = filledUsers(users.value);
  if (!groupId.value || filled.length === 0 || pending.value) {
    error.value = 'Выберите группу и заполните хотя бы одну учётную запись';
    return;
  }
  pending.value = true;
  error.value = '';
  try {
    const result = await $api<ProvisionResult>('/auth/users/batch', {
      method: 'POST',
      body: { group_id: groupId.value, users: filled },
    });
    results.set(result, groupName(groupId.value));
    await navigateTo('/users/credentials');
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    pending.value = false;
  }
}
</script>

<template>
  <section class="page">
    <form class="sheet" @submit.prevent="submit">
      <header class="sheet__header">
        <div class="sheet__titles">
          <h1 class="page-title">Новые учётные записи</h1>
          <p class="page-sub">Учётные записи будут добавлены в выбранную группу.</p>
        </div>
        <div class="sheet__actions">
          <TbField class="sheet__group" label="Группа">
            <TbSelect v-model="groupId" placeholder="Выберите группу" :options="groupOptions" />
          </TbField>
        </div>
      </header>
      <UsersTable :users="users" @add="addRow" @remove="removeRow" />
      <footer class="foot">
        <p>{{ countLabel }}</p>
        <p v-if="error" class="foot__error">{{ error }}</p>
        <TbButton type="submit" :disabled="pending">Создать учётные записи</TbButton>
      </footer>
    </form>
  </section>
</template>

<style scoped src="~/assets/css/provision-sheet.css"></style>
