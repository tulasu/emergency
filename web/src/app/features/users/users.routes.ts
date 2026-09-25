import { Routes } from '@angular/router';
import { staffGuard } from '../../core/auth/auth.guard';

export const usersRoutes: Routes = [
  { path: 'new', loadComponent: () => import('./new-users-page/new-users-page').then((m) => m.NewUsersPage) },
  {
    path: 'new/manual',
    loadComponent: () => import('./create-manual-page/create-manual-page').then((m) => m.CreateManualPage),
  },
  {
    path: 'new/batch',
    loadComponent: () => import('./create-batch-page/create-batch-page').then((m) => m.CreateBatchPage),
  },
  {
    path: 'credentials',
    loadComponent: () => import('./credentials-page/credentials-page').then((m) => m.CredentialsPage),
  },
].map((route) => ({ ...route, canActivate: [staffGuard] }));
