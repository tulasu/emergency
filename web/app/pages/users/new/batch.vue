<script setup lang="ts">
definePageMeta({
  middleware: ['auth', 'staff'],
});

await navigateTo('/users/new/import', { replace: true });
</script>

<template>
  <div></div>
</template>
