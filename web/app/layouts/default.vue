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

.shell__main > :deep(.page-sheet) {
  display: flex;
  justify-content: stretch;
  align-items: stretch;
  padding: 0;
}

.shell__main > :deep(.page-sheet > .sheet) {
  max-width: none;
}

@media (max-width: 900px) {
  .shell {
    gap: var(--space-md);
    padding: 16px;
  }

  .shell__main {
    min-height: calc(100vh - 32px);
  }
}

@media (max-width: 640px) {
  .shell {
    gap: 12px;
    padding: 12px;
  }
}
</style>
