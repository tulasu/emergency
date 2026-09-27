<script setup lang="ts">
defineProps<{
  title: string;
  subtitle?: string;
  breadcrumbs?: Array<{ label: string; to?: string }>;
}>();
</script>

<template>
  <header class="sheet__header curriculum-page-header">
    <div class="sheet__header-row">
      <div class="curriculum-page-header__text">
        <nav v-if="breadcrumbs?.length" class="curriculum-breadcrumbs" aria-label="Навигация">
          <template v-for="(crumb, index) in breadcrumbs" :key="`${crumb.label}-${index}`">
            <NuxtLink v-if="crumb.to" :to="crumb.to" class="curriculum-breadcrumbs__link">
              {{ crumb.label }}
            </NuxtLink>
            <span v-else class="curriculum-breadcrumbs__current">{{ crumb.label }}</span>
            <span
              v-if="index < breadcrumbs.length - 1"
              class="curriculum-breadcrumbs__sep"
            >/</span>
          </template>
        </nav>
        <h1 class="page-title">{{ title }}</h1>
        <p v-if="subtitle" class="page-sub">{{ subtitle }}</p>
        <slot name="meta" />
      </div>
      <div v-if="$slots.actions" class="curriculum-actions">
        <slot name="actions" />
      </div>
    </div>
  </header>
</template>
