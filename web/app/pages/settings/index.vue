<script setup lang="ts">
import { apiErrorMessage } from '~/utils/api-error';

definePageMeta({
  middleware: 'auth',
});

const usersApi = useUsers();
const current = ref('');
const next = ref('');
const confirm = ref('');
const busy = ref(false);
const error = ref('');
const toast = ref('');

async function save(): Promise<void> {
  error.value = '';
  toast.value = '';
  if (next.value.length < 8) {
    error.value = 'Новый пароль должен быть не короче 8 символов';
    return;
  }
  if (next.value !== confirm.value) {
    error.value = 'Пароли не совпадают';
    return;
  }
  busy.value = true;
  try {
    await usersApi.changePassword(current.value, next.value);
    toast.value = 'Пароль изменён';
    current.value = '';
    next.value = '';
    confirm.value = '';
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    busy.value = false;
  }
}
</script>

<template>
  <section class="page-sheet">
    <div class="sheet">
      <header class="sheet__header">
        <h1 class="page-title">Настройки</h1>
        <p class="page-sub">Смена пароля</p>
      </header>
      <div class="sheet__rule"></div>
      <div class="sheet__body">
        <div class="curriculum-form" style="max-width: 420px">
          <TbField label="Текущий пароль">
            <TbInput v-model="current" type="password" />
          </TbField>
          <TbField label="Новый пароль">
            <TbInput v-model="next" type="password" />
          </TbField>
          <TbField label="Повтор нового пароля">
            <TbInput v-model="confirm" type="password" />
          </TbField>
          <p v-if="error" class="curriculum-error">{{ error }}</p>
          <p v-if="toast" class="toast">{{ toast }}</p>
          <TbButton :busy="busy" @click="save">Сменить пароль</TbButton>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.toast {
  margin: 0;
  color: var(--color-success);
  font: var(--font-mute);
}
</style>
