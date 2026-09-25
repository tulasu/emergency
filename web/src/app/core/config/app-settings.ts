import { InjectionToken } from '@angular/core';
import { environment } from 'src/environments/environment';

export interface AppSettings {
  readonly companyName: string;
}

export const APP_SETTINGS = new InjectionToken<AppSettings>('APP_SETTINGS', {
  providedIn: 'root',
  factory: () => ({
    companyName: environment.companyName,
  }),
});
