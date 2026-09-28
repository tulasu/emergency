<script setup lang="ts">
definePageMeta({
  middleware: ['auth', 'admin'],
});
</script>

<template>
  <section class="page-sheet">
    <div class="sheet">
      <header class="sheet__header">
        <nav class="curriculum-breadcrumbs" aria-label="Навигация">
          <NuxtLink to="/admin" class="curriculum-breadcrumbs__link">Админ</NuxtLink>
          <span class="curriculum-breadcrumbs__sep">/</span>
          <span>Группы и права</span>
        </nav>
        <h1 class="page-title">Группы и права</h1>
        <p class="page-sub">Состав групп, роли и уровни доступа</p>
      </header>
      <div class="sheet__rule"></div>
      <div class="sheet__body">
        <p class="page-sub">
          Управление составом — на странице групп. Смена роли пользователя — в карточке
          пользователя. Матрица прав на уровне ролей: admin / teacher / student.
        </p>
        <div class="matrix">
          <div class="matrix__row matrix__head">
            <span>Право</span>
            <span>Student</span>
            <span>Teacher</span>
            <span>Admin</span>
          </div>
          <div
            v-for="row in [
              ['Проходить занятия', 'да', 'нет', 'нет'],
              ['Создавать карточки', 'нет', 'да', 'да'],
              ['Запускать занятия', 'нет', 'да', 'да'],
              ['Управлять группами', 'нет', 'да', 'да'],
              ['Менять роли / блокировать', 'нет', 'нет', 'да'],
              ['Админ-панель', 'нет', 'нет', 'да'],
            ]"
            :key="row[0]"
            class="matrix__row"
          >
            <span>{{ row[0] }}</span>
            <span>{{ row[1] }}</span>
            <span>{{ row[2] }}</span>
            <span>{{ row[3] }}</span>
          </div>
        </div>
        <TbButton @click="navigateTo('/groups')">К списку групп</TbButton>
      </div>
    </div>
  </section>
</template>

<style scoped>
.matrix {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.matrix__row {
  display: grid;
  grid-template-columns: 2fr 1fr 1fr 1fr;
  gap: 12px;
  padding: 12px 16px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  font: 500 14px/1.3 var(--font-sans);
}

.matrix__head {
  color: var(--color-text-muted);
  font: 500 12px/1.3 var(--font-sans);
  background: transparent;
  border: 0;
}
</style>
