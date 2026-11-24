import { HttpErrorResponse } from '@angular/common/http';
import { computed, inject } from '@angular/core';
import { Router } from '@angular/router';
import { patchState, signalStore, withComputed, withMethods, withState } from '@ngrx/signals';
import { firstValueFrom } from 'rxjs';
import {
  ApiErrorResponse,
  AuthResponse,
  LoginRequest,
  RegisterRequest,
  User,
} from '../models/auth.model';
import { AuthApiService } from '../services/auth-api.service';

interface AuthState {
  user: User | null;
  accessToken: string | null;
  isLoading: boolean;
  isInitialized: boolean;
  error: string | null;
}

const initialState: AuthState = {
  user: null,
  accessToken: null,
  isLoading: false,
  isInitialized: false,
  error: null,
};

export const AuthStore = signalStore(
  { providedIn: 'root' },
  withState(initialState),
  withComputed((store) => ({
    isAuthenticated: computed(() => Boolean(store.accessToken() && store.user())),
    currentLevel: computed(() => store.user()?.cefr_level ?? 'A1'),
    displayName: computed(() => store.user()?.full_name ?? 'Student'),
  })),
  withMethods((store) => {
    const authApi = inject(AuthApiService);
    const router = inject(Router);

    const extractErrorMessage = (err: unknown, fallback: string): string => {
      if (err instanceof HttpErrorResponse) {
        const apiErr = err.error as ApiErrorResponse | undefined;
        if (apiErr?.error) {
          return apiErr.error;
        }
      }
      return fallback;
    };

    const setSession = (response: AuthResponse): void => {
      patchState(store, {
        user: response.user,
        accessToken: response.access_token,
        isInitialized: true,
        error: null,
      });
    };

    const clearSession = (): void => {
      patchState(store, {
        user: null,
        accessToken: null,
        isInitialized: true,
        error: null,
      });
    };

    return {
      clearError(): void {
        patchState(store, { error: null });
      },

      clearSession,

      async login(credentials: LoginRequest): Promise<boolean> {
        patchState(store, { isLoading: true, error: null });
        try {
          const response = await firstValueFrom(authApi.login(credentials));
          setSession(response);
          return true;
        } catch (err) {
          patchState(store, {
            error: extractErrorMessage(err, 'Invalid email or password'),
          });
          return false;
        } finally {
          patchState(store, { isLoading: false });
        }
      },

      async register(payload: RegisterRequest): Promise<boolean> {
        patchState(store, { isLoading: true, error: null });
        try {
          await firstValueFrom(authApi.register(payload));
          return true;
        } catch (err) {
          patchState(store, {
            error: extractErrorMessage(err, 'Could not complete registration'),
          });
          return false;
        } finally {
          patchState(store, { isLoading: false });
        }
      },

      /**
       * Try retrieve access_token using HttpOnly cookie (lms_refresh_token)
       * when user reload page or when JWT expire is 15 minutes
       */
      async restoreSession(): Promise<boolean> {
        try {
          const response = await firstValueFrom(authApi.refresh());
          setSession(response);
          return true;
        } catch {
          clearSession();
          return false;
        }
      },

      async refreshProfile(): Promise<void> {
        try {
          const user = await firstValueFrom(authApi.me());
          patchState(store, { user });
        } catch {
          // interceptor handle 401 if token expired
        }
      },

      async logout(): Promise<void> {
        patchState(store, { isLoading: true });
        try {
          await firstValueFrom(authApi.logout());
        } catch {
          // if network failed we clean local state of browser
        } finally {
          clearSession();
          patchState(store, { isLoading: false });
          await router.navigate(['/login']);
        }
      },
    };
  })
);
