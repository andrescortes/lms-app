import { HttpErrorResponse, HttpInterceptorFn, HttpRequest } from '@angular/common/http';
import { inject } from '@angular/core';
import { Router } from '@angular/router';
import { Observable, catchError, finalize, from, shareReplay, switchMap, throwError } from 'rxjs';
import { AuthStore } from '../store/auth.store';

let refreshInProgress$: Observable<boolean> | null = null;

const PUBLIC_AUTH_URLS = ['/api/auth/login', '/api/auth/register', '/api/auth/refresh'];

function isPublicAuthRequest(url: string): boolean {
  return PUBLIC_AUTH_URLS.some((publicUrl) => url.includes(publicUrl));
}

function addBearerToken(req: HttpRequest<unknown>, token: string): HttpRequest<unknown> {
  return req.clone({
    setHeaders: {
      Authorization: `Bearer ${token}`,
    },
  });
}

export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const authStore = inject(AuthStore);
  const router = inject(Router);

  const token = authStore.accessToken();
  const authReq = token && !isPublicAuthRequest(req.url) ? addBearerToken(req, token) : req;

  return next(authReq).pipe(
    catchError((error: unknown) => {
      if (
        !(error instanceof HttpErrorResponse) ||
        error.status !== 401 ||
        isPublicAuthRequest(req.url) ||
        req.url.includes('/api/auth/logout')
      ) {
        return throwError(() => error);
      }

      if (!refreshInProgress$) {
        refreshInProgress$ = from(authStore.restoreSession()).pipe(
          shareReplay(1),
          finalize(() => {
            refreshInProgress$ = null;
          })
        );
      }

      return refreshInProgress$.pipe(
        switchMap((refreshed) => {
          const newToken = authStore.accessToken();
          if (refreshed && newToken) {
            return next(addBearerToken(req, newToken));
          }

          authStore.clearSession();
          void router.navigate(['/login']);
          return throwError(() => error);
        })
      );
    })
  );
};
