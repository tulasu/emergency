<script setup lang="ts">
import type { Module } from '~/types/curriculum';
import { apiErrorMessage } from '~/utils/api-error';

definePageMeta({
  middleware: ['auth', 'staff'],
});

const modulesApi = useModules();
const query = ref('');
const scope = ref('all');
const items = ref<Module[]>([]);
const selectedId = ref('');
const loading = ref(true);
const error = ref('');
const sortNewestFirst = ref(true);

const sortedItems = computed(() => {
  const list = [...items.value];
  list.sort((a, b) => {
    const da = new Date(a.created_at).getTime();
    const db = new Date(b.created_at).getTime();
    return sortNewestFirst.value ? db - da : da - db;
  });
  return list;
});

const scopes = computed(() => [
  { id: 'all', label: 'Все', count: scopeCounts.value.all },
  { id: 'active', label: 'Идут', count: scopeCounts.value.active },
  { id: 'draft', label: 'Черновики', count: scopeCounts.value.draft },
  { id: 'archived', label: 'Архив', count: scopeCounts.value.archived },
]);

const scopeCounts = ref({ all: 0, active: 0, draft: 0, archived: 0 });

let timer: ReturnType<typeof setTimeout> | null = null;

async function loadCounts(): Promise<void> {
  try {
    const all = await modulesApi.list(query.value.trim(), 'all');
    scopeCounts.value = {
      all: all.length,
      active: all.filter((m) => m.status === 'active').length,
      draft: all.filter((m) => m.status === 'draft').length,
      archived: all.filter((m) => m.status === 'archived').length,
    };
  } catch {
    // keep previous counts
  }
}

async function load(): Promise<void> {
  loading.value = true;
  error.value = '';
  try {
    await loadCounts();
    items.value = await modulesApi.list(
      query.value.trim(),
      scope.value as 'all' | 'active' | 'draft' | 'archived',
    );
    if (!selectedId.value && items.value[0]) {
      selectedId.value = items.value[0].id;
    }
  } catch (err) {
    error.value = apiErrorMessage(err);
    items.value = [];
  } finally {
    loading.value = false;
  }
}

function onSearch(): void {
  if (timer) {
    clearTimeout(timer);
  }
  timer = setTimeout(() => {
    void load();
  }, 250);
}

function openModule(mod: Module): void {
  selectedId.value = mod.id;
  void navigateTo(`/modules/${mod.id}`);
}

watch(scope, () => {
  void load();
});

onMounted(() => {
  void load();
});
</script>

<template>
  <section class="modules-page">
    <header class="modules-page__header">
      <div class="modules-page__titles">
        <h1 class="page-title">Модули</h1>
        <p class="page-sub">
          Каждый модуль — самостоятельный набор занятий. Назначайте его группам и отдельным
          ученикам.
        </p>
      </div>
      <div class="modules-page__actions">
        <div class="search-wrap">
          <TbSearch v-model="query" placeholder="Поиск" @update:model-value="onSearch" />
        </div>
        <TbButton @click="navigateTo('/modules/new')">
          <TbIcon name="plus" />
          Создать модуль
        </TbButton>
      </div>
    </header>

    <div class="modules-card">
      <div class="modules-card__filters">
        <TbSegmented v-model="scope" :items="scopes" />
        <button
          type="button"
          class="sort-hint"
          @click="sortNewestFirst = !sortNewestFirst"
        >
          <TbIcon name="pencil" />
          {{ sortNewestFirst ? 'Сначала недавно изменённые' : 'Сначала старые' }}
        </button>
      </div>

      <div class="curriculum-table-head curriculum-table-head--modules">
        <span>Модуль</span>
        <span>Занятий</span>
        <span>Назначен</span>
        <span>Открыто</span>
        <span>Успешность</span>
        <span>Статус</span>
      </div>

      <p v-if="error" class="curriculum-error">{{ error }}</p>
      <p v-else-if="loading" class="curriculum-empty">Загрузка…</p>
      <p v-else-if="!sortedItems.length" class="curriculum-empty">Модули не найдены</p>
      <div v-else class="modules-card__list">
        <ModuleTableRow
          v-for="mod in sortedItems"
          :key="mod.id"
          :module="mod"
          :selected="selectedId === mod.id"
          @click="openModule(mod)"
        />
      </div>
    </div>
  </section>
</template>

<style scoped>
.modules-page {
  display: flex;
  flex-direction: column;
  gap: 24px;
  width: 100%;
  min-height: 100%;
}

.modules-page__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
}

.modules-page__titles {
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-width: 640px;
}

.modules-page__actions {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
}

.search-wrap {
  width: 280px;
}

.modules-card {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 24px;
  background: var(--color-surface);
  border-radius: var(--radius-md);
  min-height: 0;
}

.modules-card__filters {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px 16px;
}

.sort-hint {
  margin-left: auto;
  color: var(--color-text-muted);
  font: 500 13px/1.3 var(--font-sans);
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
  border: 0;
  background: transparent;
  padding: 0;
  cursor: pointer;
}

.sort-hint :deep(.tb-icon) {
  width: 14px;
  height: 14px;
}

.modules-card__list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.curriculum-table-head--modules {
  grid-template-columns: minmax(240px, 1fr) 90px 300px 150px 110px 138px;
  padding: 0 16px;
}

@media (max-width: 1360px) {
  .modules-page__header {
    flex-direction: column;
  }

  .modules-page__actions {
    width: 100%;
    flex-wrap: wrap;
  }

  .search-wrap {
    flex: 1;
    min-width: 0;
    width: auto;
  }

  .sort-hint {
    margin-left: 0;
  }
}

@media (max-width: 640px) {
  .modules-page__actions {
    flex-direction: column;
    align-items: stretch;
  }

  .search-wrap {
    width: 100%;
  }

  .modules-page__actions :deep(.tb-btn) {
    width: 100%;
  }

  .modules-card {
    padding: 16px;
  }
}
</style>
