import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { AuthResponse, LoginRequest, RegisterRequest, User } from '../models/auth.model';

@Injectable({ providedIn: 'root' })
export class AuthApiService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = '/api/auth';

  register(payload: RegisterRequest): Observable<User> {
    return this.http.post<User>(`${this.baseUrl}/register`, payload);
  }

  login(payload: LoginRequest): Observable<AuthResponse> {
    return this.http.post<AuthResponse>(`${this.baseUrl}/login`, payload, {
      withCredentials: true,
    });
  }

  refresh(): Observable<AuthResponse> {
    return this.http.post<AuthResponse>(`${this.baseUrl}/refresh`, {}, {
      withCredentials: true,
    });
  }

  me(): Observable<User> {
    return this.http.get<User>(`${this.baseUrl}/me`);
  }

  logout(): Observable<{ message: string }> {
    return this.http.post<{ message: string }>(`${this.baseUrl}/logout`, {}, {
      withCredentials: true,
    });
  }
}
