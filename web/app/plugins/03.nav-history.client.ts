export default defineNuxtPlugin((nuxtApp) => {
  const nav = useNavHistoryStore();
  const router = useRouter();

  window.addEventListener('popstate', () => {
    nav.markPop();
  });

  const originalReplace = router.replace.bind(router);
  router.replace = ((to, extras) => {
    nav.markReplace();
    return originalReplace(to, extras);
  }) as typeof router.replace;

  nuxtApp.hook('page:finish', () => {
    const route = useRoute();
    nav.onEnd(route.fullPath, route.meta.skipHistory === true);
  });
});
