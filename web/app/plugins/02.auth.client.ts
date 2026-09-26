export default defineNuxtPlugin(async () => {
  const auth = useAuthStore();
  if (!auth.hydrated) {
    await auth.hydrate();
  }
});
