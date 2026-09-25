import { Routes } from '@angular/router';
import { staffGuard } from '../../core/auth/auth.guard';

export const usersRoutes: Routes = [
  { path: 'new', loadComponent: () => import('./new-users-page/new-users-page').then((m) => m.NewUsersPage) },
  {
    path: 'new/manual',
    loadComponent: () => import('./create-manual-page/create-manual-page').then((m) => m.CreateManualPage),
  },
  {
    path: 'new/import',
    loadComponent: () => import('./import-users-page/import-users-page').then((m) => m.ImportUsersPage),
  },
  { path: 'new/batch', redirectTo: 'new/import', pathMatch: 'full' as const },
  {
    path: 'credentials',
    loadComponent: () => import('./credentials-page/credentials-page').then((m) => m.CredentialsPage),
  },
].map((route) => (route.redirectTo ? route : { ...route, canActivate: [staffGuard] }));
