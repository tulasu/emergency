import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { AuthStore } from './auth.store';

export const authGuard: CanActivateFn = async () => {
  const store = inject(AuthStore);
  const router = inject(Router);
  if (!store.hydrated()) {
    await store.hydrate();
  }
  if (store.isAuthenticated()) {
    return true;
  }
  return router.parseUrl('/login');
};

export const guestGuard: CanActivateFn = async () => {
  const store = inject(AuthStore);
  const router = inject(Router);
  if (!store.hydrated()) {
    await store.hydrate();
  }
  if (store.isAuthenticated()) {
    return router.parseUrl('/');
  }
  return true;
};

export const staffGuard: CanActivateFn = async () => {
  const store = inject(AuthStore);
  const router = inject(Router);
  if (!store.hydrated()) {
    await store.hydrate();
  }
  const role = store.user()?.role;
  if (role === 'admin' || role === 'teacher') {
    return true;
  }
  return router.parseUrl('/');
};
