import type { RouteLocationRaw } from 'vue-router';

export default defineNuxtPlugin((nuxtApp) => {
  const nav = useNavHistoryStore();
  const router = useRouter();

  window.addEventListener('popstate', () => {
    nav.markPop();
  });

  const originalReplace = router.replace.bind(router);
  router.replace = ((to: RouteLocationRaw) => {
    nav.markReplace();
    return originalReplace(to);
  }) as typeof router.replace;

  nuxtApp.hook('page:finish', () => {
    const route = useRoute();
    nav.onEnd(route.fullPath, route.meta.skipHistory === true);
  });
});
