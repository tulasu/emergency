export default defineNuxtRouteMiddleware(async () => {
  const auth = useAuthStore();
  if (!auth.hydrated) {
    await auth.hydrate();
  }
  const role = auth.user?.role;
  if (role === 'admin' || role === 'teacher') {
    return;
  }
  return navigateTo('/');
});
