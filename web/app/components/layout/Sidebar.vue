<script setup lang="ts">
const auth = useAuthStore();
const nav = useNavHistoryStore();
const route = useRoute();

const profileOpen = ref(false);
const usersActive = computed(() => route.path.startsWith('/users'));

function toggleProfile(event: Event): void {
  event.stopPropagation();
  profileOpen.value = !profileOpen.value;
}

function closeProfile(): void {
  profileOpen.value = false;
}

function onKeydown(event: KeyboardEvent): void {
  if (event.key === 'Escape') {
    closeProfile();
  }
}

onMounted(() => {
  document.addEventListener('click', closeProfile);
  document.addEventListener('keydown', onKeydown);
});

onUnmounted(() => {
  document.removeEventListener('click', closeProfile);
  document.removeEventListener('keydown', onKeydown);
});

async function logout(): Promise<void> {
  profileOpen.value = false;
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
          <button class="nav" type="button" aria-label="Назад" @click="nav.back()">
            <TbIcon name="arrow-left" />
          </button>
          <span class="sidebar__divider"></span>
        </div>
      </div>
      <button class="nav" type="button" disabled aria-label="Создать">
        <TbIcon name="plus" />
      </button>
      <button class="nav" type="button" disabled aria-label="Карточки">
        <TbIcon name="layers" />
      </button>
      <button class="nav" type="button" disabled aria-label="Уроки">
        <TbIcon name="folder" />
      </button>
      <NuxtLink
        class="nav"
        to="/users/new"
        :class="{ 'nav--active': usersActive }"
        aria-label="Группы"
      >
        <TbIcon name="users" />
      </NuxtLink>
    </nav>
    <div class="sidebar__account" @click.stop>
      <div
        class="sidebar__menu"
        :class="{ 'sidebar__menu--open': profileOpen }"
        :inert="profileOpen ? undefined : true"
      >
        <button class="nav" type="button" disabled aria-label="Настройки">
          <TbIcon name="settings" />
        </button>
        <button class="nav nav--danger" type="button" aria-label="Выйти" @click="logout">
          <TbIcon name="log-out" />
        </button>
      </div>
      <button
        class="sidebar__avatar"
        type="button"
        :aria-expanded="profileOpen"
        aria-label="Профиль"
        @click="toggleProfile"
      >
        <TbIcon name="user" />
      </button>
    </div>
  </aside>
</template>

<style scoped>
.sidebar {
  width: 72px;
  height: calc(100vh - 48px);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: 24px 16px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  align-items: center;
}

.sidebar__nav {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
}

.sidebar__account {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.sidebar__menu {
  position: absolute;
  bottom: 48px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  opacity: 0;
  transform: translateY(8px);
  pointer-events: none;
  transition:
    opacity 180ms ease,
    transform 180ms ease;
}

.sidebar__menu--open {
  opacity: 1;
  transform: translateY(0);
  pointer-events: auto;
}

.sidebar__back {
  display: grid;
  grid-template-rows: 0fr;
  margin-bottom: -16px;
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
  align-items: center;
  gap: 16px;
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
  width: 24px;
  height: 1px;
  background: var(--color-border-strong);
}

.nav {
  width: 40px;
  height: 40px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text);
  display: inline-flex;
  align-items: center;
  justify-content: center;
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
  width: 40px;
  height: 40px;
  border: 0;
  border-radius: 20px;
  background: var(--color-primary);
  color: var(--color-primary-text);
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
</style>
