<script setup lang="ts">
const props = defineProps<{
  items: string[];
}>();

function splitItem(item: string): { title: string; sub?: string } {
  const sep = item.indexOf(' · ');
  if (sep === -1) {
    return { title: item };
  }
  return { title: item.slice(0, sep), sub: item.slice(sep + 3) };
}

const rows = computed(() => props.items.map(splitItem));
</script>

<template>
  <aside class="side-panel">
    <h2 class="side-panel__title">Требует внимания</h2>
    <p v-if="!items.length" class="side-panel__empty">Пока всё в порядке</p>
    <ul v-else class="side-panel__list">
      <li v-for="(row, index) in rows" :key="`${row.title}-${index}`" class="side-panel__attention">
        <TbIcon name="info" />
        <div class="side-panel__attention-text">
          <p class="side-panel__attention-title">{{ row.title }}</p>
          <p v-if="row.sub" class="side-panel__attention-sub">{{ row.sub }}</p>
        </div>
      </li>
    </ul>
  </aside>
</template>
