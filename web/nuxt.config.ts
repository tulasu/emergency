export default defineNuxtConfig({
  compatibilityDate: '2025-09-26',
  ssr: false,
  modules: ['@pinia/nuxt'],
  css: [
    '~/assets/css/reset.css',
    '~/assets/css/tokens.css',
    '~/assets/css/typography.css',
    '~/assets/css/sheet.css',
    '~/assets/css/curriculum.css',
  ],
  components: [
    { path: '~/components/ui', pathPrefix: false },
    { path: '~/components/layout', pathPrefix: false },
    { path: '~/components/users', pathPrefix: false },
    { path: '~/components/curriculum', pathPrefix: false },
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
  },
  typescript: {
    strict: true,
  },
});
