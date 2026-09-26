<script setup lang="ts">
definePageMeta({
  middleware: ['auth', 'staff'],
});

const drafts = useImportDraftStore();
const error = ref('');
const fileInput = ref<HTMLInputElement | null>(null);

async function onFile(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = '';
  if (!file) {
    return;
  }
  const parsed = parseUsersFile(await file.text(), file.name);
  if (!parsed.ok) {
    error.value = parsed.error;
    return;
  }
  error.value = '';
  drafts.set({ fileName: file.name, kind: parsed.kind, users: parsed.users });
  await navigateTo('/users/new/import');
}
</script>

<template>
  <section class="page">
    <div class="sheet">
      <header class="sheet__header">
        <h1 class="page-title">Новые пользователи</h1>
        <p class="page-sub">Выберите, как создать новые учётные записи</p>
      </header>
      <div class="sheet__rule"></div>
      <div class="choices">
        <NuxtLink class="choice" to="/users/new/manual">
          <span class="choice__icon">
            <TbIcon name="user-plus" />
          </span>
          <h2>Вручную</h2>
          <p>Создать одного или несколько пользователей.</p>
        </NuxtLink>
        <div class="choice">
          <button class="choice__link" type="button" @click="fileInput?.click()">
            <span class="choice__icon">
              <TbIcon name="file" />
            </span>
            <h2>CSV или JSON</h2>
            <p>Загрузить пользователей в систему из файла.</p>
          </button>
          <p class="choice__samples">
            <a href="/samples/users.csv" download>Образец CSV</a>
            <span aria-hidden="true">·</span>
            <a href="/samples/users.json" download>Образец JSON</a>
          </p>
          <p v-if="error" class="choice__error">{{ error }}</p>
        </div>
      </div>
      <input
        ref="fileInput"
        class="file-input"
        type="file"
        accept=".csv,.json,text/csv,application/json"
        @change="onFile"
      />
    </div>
  </section>
</template>

<style scoped>
.page {
  width: 100%;
  min-height: 100%;
  display: flex;
  justify-content: center;
  align-items: center;
}

.sheet {
  max-width: 880px;
}

.sheet__header {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 32px 32px 24px;
}

.sheet__rule {
  height: 1px;
  background: var(--color-border);
}

.choices {
  display: grid;
  grid-template-columns: 1fr 1fr;
}

.choice,
.choice__link {
  display: flex;
  flex-direction: column;
  gap: 16px;
  text-decoration: none;
  color: inherit;
}

.choice {
  padding: 32px;
  box-sizing: border-box;
}

.choice__link {
  width: 100%;
  border: 0;
  background: transparent;
  padding: 0;
  text-align: left;
  cursor: pointer;
  font: inherit;
}

.choice__icon {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-sm);
  background: var(--color-secondary);
  color: var(--color-primary);
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.choice h2 {
  font-size: 24px;
  font-weight: 700;
  line-height: 1.2;
}

.choice p {
  color: var(--color-text-muted);
  font: var(--font-mute);
}

.choice__samples {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  color: var(--color-text-subtle);
}

.choice__samples a {
  color: var(--color-primary);
  text-decoration: none;
}

.choice__samples a:hover {
  text-decoration: underline;
}

.choice__error {
  margin: 0;
  color: var(--color-danger);
  font: var(--font-mute);
}

.file-input {
  display: none;
}

@media (max-width: 720px) {
  .choices {
    grid-template-columns: 1fr;
  }
}
</style>
