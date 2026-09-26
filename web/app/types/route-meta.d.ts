import type { $Fetch } from 'ofetch';

export {};

declare module 'vue-router' {
  interface RouteMeta {
    skipHistory?: boolean;
  }
}

declare module '#app' {
  interface PageMeta {
    skipHistory?: boolean;
  }

  interface NuxtApp {
    $api: $Fetch;
  }
}

declare module 'vue' {
  interface ComponentCustomProperties {
    $api: $Fetch;
  }
}
