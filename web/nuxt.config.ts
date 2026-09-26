export default defineNuxtConfig({
  compatibilityDate: '2025-09-26',
  ssr: false,
  modules: ['@pinia/nuxt'],
  css: [
    '~/assets/css/reset.css',
    '~/assets/css/tokens.css',
    '~/assets/css/typography.css',
    '~/assets/css/sheet.css',
  ],
  components: [
    { path: '~/components/ui', pathPrefix: false },
    { path: '~/components/layout', pathPrefix: false },
    { path: '~/components/users', pathPrefix: false },
  ],
  runtimeConfig: {
    public: {
      companyName: 'Токенoeжки',
    },
  },
  app: {
    head: {
      htmlAttrs: { lang: 'ru' },
      title: 'Токенoeжки',
      link: [
        { rel: 'icon', type: 'image/x-icon', href: '/favicon.ico' },
        { rel: 'preconnect', href: 'https://fonts.googleapis.com' },
        { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: '' },
        {
          rel: 'stylesheet',
          href: 'https://fonts.googleapis.com/css2?family=Manrope:wght@400;500;600;700&display=swap',
        },
      ],
    },
  },
  routeRules: {
    '/users/new/batch': { redirect: '/users/new/import' },
    '/auth/**': { proxy: 'http://127.0.0.1:8080/auth/**' },
    '/groups/**': { proxy: 'http://127.0.0.1:8080/groups/**' },
    '/health/**': { proxy: 'http://127.0.0.1:8080/health/**' },
  },
  vite: {
    server: {
      proxy: {
        '/auth': { target: 'http://127.0.0.1:8080' },
        '/health': { target: 'http://127.0.0.1:8080' },
        '/groups': { target: 'http://127.0.0.1:8080' },
      },
    },
  },
  nitro: {
    devProxy: {
      '/auth': { target: 'http://127.0.0.1:8080' },
      '/health': { target: 'http://127.0.0.1:8080' },
      '/groups': { target: 'http://127.0.0.1:8080' },
    },
  },
  typescript: {
    strict: true,
  },
})
