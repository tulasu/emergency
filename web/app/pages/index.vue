<script setup lang="ts">
definePageMeta({
  middleware: 'auth',
});

const auth = useAuthStore();
const config = useRuntimeConfig();
const companyName = config.public.companyName;
</script>

<template>
  <section class="home">
    <h1 class="page-title">{{ companyName }}</h1>
    <p class="page-sub">Вы вошли как {{ auth.user?.login }}</p>
  </section>
</template>

<style scoped>
.home {
  padding: 24px 8px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
</style>
