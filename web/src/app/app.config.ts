import { provideHttpClient, withInterceptors } from '@angular/common/http';
import {
  ApplicationConfig,
  inject,
  provideAppInitializer,
  provideBrowserGlobalErrorListeners,
  provideZonelessChangeDetection,
} from '@angular/core';
import { provideRouter } from '@angular/router';
import { APP_SETTINGS } from './core/config/app-settings';
import { authInterceptor } from './core/auth/auth.interceptor';
import { AuthStore } from './core/auth/auth.store';
import { routes } from './app.routes';

export const appConfig: ApplicationConfig = {
  providers: [
    provideBrowserGlobalErrorListeners(),
    provideZonelessChangeDetection(),
    provideRouter(routes),
    provideHttpClient(withInterceptors([authInterceptor])),
    provideAppInitializer(() => {
      document.title = inject(APP_SETTINGS).companyName;
    }),
    provideAppInitializer(() => inject(AuthStore).hydrate()),
  ],
};
