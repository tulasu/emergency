export default defineNuxtRouteMiddleware(async () => {
  const auth = useAuthStore();
  if (!auth.hydrated) {
    await auth.hydrate();
  }
  if (auth.isAuthenticated) {
    return navigateTo('/');
  }
});
