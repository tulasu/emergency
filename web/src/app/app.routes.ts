import { Routes } from '@angular/router';
import { authGuard, guestGuard } from './core/auth/auth.guard';
import { AppShell } from './core/layout/app-shell/app-shell';
import { AuthLayout } from './core/layout/auth-layout/auth-layout';
import { skipHistory } from './core/nav/nav-history';

export const routes: Routes = [
  {
    path: 'login',
    component: AuthLayout,
    canActivate: [guestGuard],
    data: skipHistory,
    children: [
      {
        path: '',
        loadComponent: () => import('./features/auth/login-page/login-page').then((m) => m.LoginPage),
      },
    ],
  },
  {
    path: '',
    component: AppShell,
    canActivate: [authGuard],
    children: [
      {
        path: '',
        loadComponent: () => import('./features/home-page/home-page').then((m) => m.HomePage),
      },
      {
        path: 'users',
        loadChildren: () => import('./features/users/users.routes').then((m) => m.usersRoutes),
      },
    ],
  },
  { path: '**', redirectTo: '' },
];
