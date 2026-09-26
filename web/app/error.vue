<script setup lang="ts">
const error = useError();

onMounted(() => {
  if (error.value?.statusCode === 404) {
    void clearError({ redirect: '/' });
  }
});
</script>

<template>
  <div class="error" v-if="error && error.statusCode !== 404">
    <h1 class="page-title">{{ error.statusCode }}</h1>
    <p class="page-sub">{{ error.message }}</p>
  </div>
</template>

<style scoped>
.error {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  gap: 8px;
  padding: 24px;
}
</style>
