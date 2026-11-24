import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { AuthStore } from '../store/auth.store';

export const authGuard: CanActivateFn = async () => {
  const authStore = inject(AuthStore);
  const router = inject(Router);

  if (authStore.isAuthenticated()) {
    return true;
  }

  if (!authStore.isInitialized()) {
    await authStore.restoreSession();
  }

  return authStore.isAuthenticated() ? true : router.createUrlTree(['/login']);
};

export const guestGuard: CanActivateFn = async () => {
  const authStore = inject(AuthStore);
  const router = inject(Router);

  if (!authStore.isInitialized()) {
    await authStore.restoreSession();
  }

  return authStore.isAuthenticated() ? router.createUrlTree(['/dashboard']) : true;
};
