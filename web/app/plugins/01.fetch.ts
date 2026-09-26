import type { $Fetch, FetchOptions } from 'ofetch';

export default defineNuxtPlugin(() => {
  const auth = useAuthStore();

  const api = $fetch.create({
    onRequest({ options }: { options: FetchOptions }) {
      if (!auth.token) {
        return;
      }
      const headers = new Headers(options.headers as HeadersInit | undefined);
      headers.set('Authorization', `Bearer ${auth.token}`);
      options.headers = headers;
    },
    onResponseError({ request, response }) {
      const url = typeof request === 'string' ? request : request.toString();
      if (response.status === 401 && !url.includes('/auth/login') && auth.token) {
        auth.clear();
        void navigateTo('/login');
      }
    },
  });

  return {
    provide: {
      api: api as $Fetch,
    },
  };
});
