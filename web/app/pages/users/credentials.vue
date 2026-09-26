<script setup lang="ts">
definePageMeta({
  middleware: ['auth', 'staff'],
});

const results = useProvisionResultStore();
const created = computed(() => results.result?.created ?? []);
const failed = computed(() => results.result?.failed ?? []);

onMounted(async () => {
  if (!results.result) {
    await navigateTo('/users/new', { replace: true });
  }
});

const subtitle = computed(() => {
  const count = ruCount(created.value.length, 'пользователь', 'пользователя', 'пользователей');
  const group = results.groupName;
  const groupPart = group ? ` · группа ${group}` : '';
  return `${count}${groupPart}. Пароли больше не будут отображаться. Скопируйте или скачайте их сейчас.`;
});

function failMessage(code: string): string {
  return errorCodeMessage(code);
}

async function copyRow(login: string, password: string): Promise<void> {
  await navigator.clipboard.writeText(`${login}\t${password}`);
}

async function copyAll(): Promise<void> {
  const text = created.value
    .map((user) => `${user.full_name}\t${user.login}\t${user.password}`)
    .join('\n');
  await navigator.clipboard.writeText(text);
}

function download(kind: 'csv' | 'txt'): void {
  const rows = created.value;
  const body =
    kind === 'csv'
      ? exportAccessesCsv(rows)
      : rows.map((user) => `${user.full_name}\t${user.login}\t${user.password}`).join('\n');
  const blob = new Blob([body], { type: 'text/plain;charset=utf-8' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = kind === 'csv' ? 'accesses.csv' : 'accesses.txt';
  a.click();
  URL.revokeObjectURL(url);
}
</script>

<template>
  <section class="page">
    <header class="page__header">
      <div class="page__titles">
        <h1 class="page-title">Учётные записи созданы</h1>
        <p class="page-sub">{{ subtitle }}</p>
      </div>
      <div class="page__actions">
        <TbButton variant="secondary" type="button" @click="copyAll">Скопировать всё</TbButton>
        <TbButton variant="ghost" type="button" @click="download('csv')">Скачать CSV</TbButton>
        <TbButton variant="ghost" type="button" @click="download('txt')">Скачать TXT</TbButton>
      </div>
    </header>

    <div class="sheet table-wrap">
      <div class="table">
        <div class="table__head">
          <span>ФИО</span>
          <span>Логин</span>
          <span>Пароль</span>
          <span></span>
        </div>
        <div v-for="user in created" :key="user.id" class="table__row">
          <span>{{ user.full_name }}</span>
          <span>{{ user.login }}</span>
          <span>{{ user.password }}</span>
          <button
            type="button"
            class="copy"
            aria-label="Скопировать"
            @click="copyRow(user.login, user.password)"
          >
            <TbIcon name="copy" />
          </button>
        </div>
      </div>
    </div>

    <div v-if="failed.length" class="failed">
      <h2>Не созданы</h2>
      <p v-for="row in failed" :key="row.index">
        {{ row.login || `Строка ${row.index + 1}` }} — {{ failMessage(row.error) }}
      </p>
    </div>
  </section>
</template>

<style scoped>
.page {
  width: 100%;
  min-height: 100%;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.page__header {
  display: flex;
  justify-content: space-between;
  gap: 24px;
  flex-wrap: wrap;
  align-items: flex-start;
}

.page__titles {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: min(100%, 240px);
  flex: 1 1 320px;
}

.page__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.table-wrap {
  overflow: hidden;
  flex: 1;
  border-color: var(--color-border);
}

.table {
  overflow-x: auto;
}

.table__head,
.table__row {
  display: grid;
  grid-template-columns: minmax(200px, 1fr) 220px 200px 40px;
  gap: 16px;
  align-items: center;
  padding: 12px 20px;
  min-width: 560px;
}

.table__head {
  background: var(--color-bg);
  color: var(--color-text-muted);
}

.table__row {
  border-bottom: 1px solid var(--color-border);
}

.copy {
  border: 0;
  background: transparent;
  color: var(--color-text-muted);
}

.failed {
  color: var(--color-danger);
}

.failed h2 {
  font-size: 16px;
  margin-bottom: 8px;
}

@media (max-width: 640px) {
  .table__head,
  .table__row {
    grid-template-columns: 1fr 1fr 1fr 36px;
    gap: 8px;
    padding: 12px;
    min-width: 0;
  }
}
</style>
