import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { LevelsResponse } from '../models/lms.model';

@Injectable({ providedIn: 'root' })
export class LmsApiService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = '/api/lms';

  getLevels(): Observable<LevelsResponse> {
    return this.http.get<LevelsResponse>(`${this.baseUrl}/levels`);
  }
}
