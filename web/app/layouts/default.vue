<template>
  <div class="shell">
    <Sidebar />
    <main class="shell__main">
      <slot />
    </main>
  </div>
</template>

<style scoped>
.shell {
  min-height: 100vh;
  display: flex;
  gap: var(--space-lg);
  padding: 24px;
  background: var(--color-bg);
}

.shell__main {
  flex: 1;
  min-width: 0;
  min-height: calc(100vh - 48px);
  display: flex;
  flex-direction: column;
}

.shell__main > :deep(*) {
  flex: 1;
  width: 100%;
  min-height: 0;
}
</style>
