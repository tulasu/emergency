<script setup lang="ts">
const auth = useAuthStore();
const nav = useNavHistoryStore();
const route = useRoute();

const createOpen = ref(false);

const isStaff = computed(
  () => auth.user?.role === 'admin' || auth.user?.role === 'teacher',
);

const ticketsActive = computed(() => route.path.startsWith('/tickets'));
const modulesActive = computed(
  () => route.path.startsWith('/modules') || route.path.startsWith('/lessons'),
);
const usersActive = computed(() => route.path.startsWith('/users'));
const homeActive = computed(() => route.path === '/' || route.path.startsWith('/learn'));

const shortName = computed(() => {
  const full = auth.user?.full_name?.trim();
  if (!full) {
    return auth.user?.login ?? '';
  }
  const parts = full.split(/\s+/).filter(Boolean);
  const first = parts[0];
  const second = parts[1];
  if (!first || !second) {
    return first ?? full;
  }
  const initial = second.charAt(0);
  return initial ? `${first} ${initial}.` : first;
});

function toggleCreate(event: Event): void {
  event.stopPropagation();
  createOpen.value = !createOpen.value;
}

function closeMenus(): void {
  createOpen.value = false;
}

function onKeydown(event: KeyboardEvent): void {
  if (event.key === 'Escape') {
    closeMenus();
  }
}

onMounted(() => {
  document.addEventListener('click', closeMenus);
  document.addEventListener('keydown', onKeydown);
});

onUnmounted(() => {
  document.removeEventListener('click', closeMenus);
  document.removeEventListener('keydown', onKeydown);
});

async function logout(): Promise<void> {
  closeMenus();
  await auth.logout();
  await navigateTo('/login');
}
</script>

<template>
  <aside class="sidebar">
    <nav class="sidebar__nav">
      <div
        class="sidebar__back"
        :class="{ 'sidebar__back--open': nav.canGoBack }"
        :inert="nav.canGoBack ? undefined : true"
      >
        <div class="sidebar__back-inner">
          <button class="nav" type="button" @click="nav.back()">
            <TbIcon name="arrow-left" />
            <span>Назад</span>
          </button>
          <span class="sidebar__divider"></span>
        </div>
      </div>

      <template v-if="isStaff">
        <div class="sidebar__create" @click.stop>
          <button class="nav nav--create" type="button" title="Создать" @click="toggleCreate">
            <TbIcon name="plus" />
            <span>Создать</span>
          </button>
          <div
            class="sidebar__create-menu"
            :class="{ 'sidebar__create-menu--open': createOpen }"
            :inert="createOpen ? undefined : true"
          >
            <NuxtLink class="nav" to="/modules/new" @click="closeMenus">Модуль</NuxtLink>
            <NuxtLink class="nav" to="/tickets" @click="closeMenus">Карточка</NuxtLink>
          </div>
        </div>
        <NuxtLink class="nav" to="/tickets" title="Карточки" :class="{ 'nav--active': ticketsActive }">
          <TbIcon name="layers" />
          <span>Карточки</span>
        </NuxtLink>
        <NuxtLink class="nav" to="/modules" title="Модули" :class="{ 'nav--active': modulesActive }">
          <TbIcon name="book" />
          <span>Модули</span>
        </NuxtLink>
        <NuxtLink class="nav" to="/users/new" title="Группы" :class="{ 'nav--active': usersActive }">
          <TbIcon name="users" />
          <span>Группы</span>
        </NuxtLink>
        <button class="nav" type="button" title="Аналитика" disabled>
          <TbIcon name="chart" />
          <span>Аналитика</span>
        </button>
        <button class="nav" type="button" title="Справочник" disabled>
          <TbIcon name="file" />
          <span>Справочник</span>
        </button>
      </template>

      <template v-else>
        <NuxtLink class="nav" to="/" title="Главная" :class="{ 'nav--active': homeActive }">
          <TbIcon name="home" />
          <span>Главная</span>
        </NuxtLink>
        <button class="nav" type="button" title="Справочная база" disabled>
          <TbIcon name="book" />
          <span>Справочная база</span>
        </button>
        <button class="nav" type="button" title="Мои результаты" disabled>
          <TbIcon name="chart" />
          <span>Мои результаты</span>
        </button>
      </template>
    </nav>

    <div class="sidebar__account">
      <button class="nav" type="button" title="Настройки" disabled>
        <TbIcon name="settings" />
        <span>Настройки</span>
      </button>
      <button class="nav nav--danger" type="button" title="Выйти" @click="logout">
        <TbIcon name="log-out" />
        <span>Выйти</span>
      </button>
      <div class="sidebar__avatar" :title="shortName || undefined">
        <span class="sidebar__avatar-icon">
          <TbIcon name="user" />
        </span>
        <span v-if="shortName" class="sidebar__avatar-name">{{ shortName }}</span>
      </div>
    </div>
  </aside>
</template>

<style scoped>
.sidebar {
  width: 220px;
  height: calc(100vh - 48px);
  background: var(--color-surface);
  border-radius: var(--radius-md);
  padding: 24px 16px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  flex-shrink: 0;
}

.sidebar__nav {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.sidebar__create {
  position: relative;
}

.sidebar__create-menu {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  right: 0;
  z-index: 2;
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 8px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-surface);
  box-shadow: var(--shadow-card);
  opacity: 0;
  pointer-events: none;
  transform: translateY(-4px);
  transition:
    opacity 160ms ease,
    transform 160ms ease;
}

.sidebar__create-menu--open {
  opacity: 1;
  pointer-events: auto;
  transform: translateY(0);
}

.nav--create {
  border: 1px solid var(--color-primary);
  color: var(--color-primary);
}

.sidebar__account {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.sidebar__back {
  display: grid;
  grid-template-rows: 0fr;
  margin-bottom: -6px;
  pointer-events: none;
  transition:
    grid-template-rows 180ms ease,
    margin-bottom 180ms ease;
}

.sidebar__back--open {
  grid-template-rows: 1fr;
  margin-bottom: 0;
  pointer-events: auto;
}

.sidebar__back-inner {
  overflow: hidden;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.sidebar__back-inner > * {
  opacity: 0;
  transform: translateY(8px);
  transition:
    opacity 180ms ease,
    transform 180ms ease;
}

.sidebar__back--open .sidebar__back-inner > * {
  opacity: 1;
  transform: translateY(0);
}

.sidebar__divider {
  width: 100%;
  height: 1px;
  background: var(--color-border-strong);
}

.nav {
  width: 100%;
  min-height: 40px;
  padding: 8px 10px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text);
  display: inline-flex;
  align-items: center;
  gap: 10px;
  text-decoration: none;
  font: var(--font-body);
}

.nav:disabled {
  color: var(--color-text-subtle);
  cursor: default;
}

.nav--active {
  background: var(--color-primary);
  color: var(--color-primary-text);
}

.nav--danger {
  color: var(--color-danger);
}

.sidebar__avatar {
  width: 100%;
  min-height: 40px;
  padding: 6px 8px;
  display: inline-flex;
  align-items: center;
  gap: 10px;
  color: var(--color-text);
}

.sidebar__avatar-icon {
  width: 28px;
  height: 28px;
  border-radius: 14px;
  background: var(--color-primary);
  color: var(--color-primary-text);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.sidebar__avatar-name {
  font: var(--font-body);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (max-width: 900px) {
  .sidebar {
    width: 64px;
    padding: 16px 8px;
  }

  .nav,
  .sidebar__avatar {
    justify-content: center;
    padding-left: 8px;
    padding-right: 8px;
    gap: 0;
  }

  .nav > span,
  .sidebar__avatar-name {
    position: absolute;
    width: 1px;
    height: 1px;
    padding: 0;
    margin: -1px;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    white-space: nowrap;
    border: 0;
  }

  .sidebar__create-menu {
    left: calc(100% + 8px);
    right: auto;
    width: 160px;
  }
}
</style>
