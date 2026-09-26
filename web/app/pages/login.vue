<script setup lang="ts">
definePageMeta({
  layout: 'auth',
  middleware: 'guest',
  skipHistory: true,
});

const auth = useAuthStore();
const config = useRuntimeConfig();
const companyName = config.public.companyName;
const year = new Date().getFullYear();

const login = ref('');
const password = ref('');
const error = ref('');
const pending = ref(false);
const disabled = computed(() => pending.value);

function isValid(): boolean {
  return login.value.length >= 3 && login.value.length <= 64 && password.value.length >= 8;
}

async function submit(): Promise<void> {
  if (!isValid() || pending.value) {
    return;
  }
  pending.value = true;
  try {
    await auth.login(login.value, password.value);
    await navigateTo('/', { replace: true });
  } catch (err) {
    error.value = apiErrorMessage(err);
  } finally {
    pending.value = false;
  }
}
</script>

<template>
  <div class="login">
    <TbCard as="form" class="login__card" @submit.prevent="submit">
      <header class="login__header">
        <h1 class="page-title page-title--login">Вход</h1>
        <p class="page-sub">Введите логин и пароль</p>
      </header>
      <div class="login__main">
        <div class="login__fields">
          <TbField label="Логин">
            <TbInput
              v-model="login"
              autocomplete="username"
              placeholder="Введите логин"
              :disabled="disabled"
            />
          </TbField>
          <TbField label="Пароль">
            <TbInput
              v-model="password"
              type="password"
              autocomplete="current-password"
              placeholder="Введите пароль"
              :disabled="disabled"
            />
          </TbField>
        </div>
        <div class="login__alert" :class="{ 'login__alert--on': error }" :aria-hidden="error ? undefined : true">
          <div class="login__alert-inner">
            <p class="login__error" aria-live="polite">{{ error }}</p>
          </div>
        </div>
        <TbButton width="block" type="submit" :busy="pending">Войти</TbButton>
      </div>
    </TbCard>
    <p class="login__copy">© {{ companyName }}, {{ year }}</p>
  </div>
</template>

<style scoped>
.login {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 24px;
}

.login__card {
  width: min(420px, 100%);
  margin: auto 0;
  padding: 40px;
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 28px;
}

.login__header {
  text-align: center;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.login__main {
  display: flex;
  flex-direction: column;
}

.login__fields {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 16px;
  margin-bottom: 28px;
  transition: margin-bottom 220ms ease;
}

.login__main:has(.login__alert--on) .login__fields {
  margin-bottom: 0;
}

.login__alert {
  display: grid;
  grid-template-rows: 0fr;
  transition: grid-template-rows 220ms ease;
}

.login__alert--on {
  grid-template-rows: 1fr;
}

.login__alert-inner {
  overflow: hidden;
  min-height: 0;
}

.login__error {
  margin: 0;
  padding: 12px 0;
  text-align: center;
  color: var(--color-danger);
  font: var(--font-cap);
  opacity: 0;
  transform: translateY(-6px);
  transition:
    opacity 220ms ease,
    transform 220ms ease;
}

.login__alert--on .login__error {
  opacity: 1;
  transform: translateY(0);
}

.login__copy {
  color: var(--color-text-subtle);
  font: var(--font-cap);
}
</style>
