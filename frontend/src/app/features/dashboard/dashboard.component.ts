import { ChangeDetectionStrategy, Component, OnInit, inject, signal } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatToolbarModule } from '@angular/material/toolbar';
import { firstValueFrom } from 'rxjs';
import { CEFRLevel } from '../../core/models/lms.model';
import { LmsApiService } from '../../core/services/lms-api.service';
import { AuthStore } from '../../core/store/auth.store';

@Component({
  selector: 'app-dashboard',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatToolbarModule,
    MatButtonModule,
    MatIconModule,
    MatCardModule,
    MatProgressSpinnerModule,
  ],
  styleUrl: './dashboard.component.scss',
  templateUrl: './dashboard.component.html',
})
export class DashboardComponent implements OnInit {
  protected readonly authStore = inject(AuthStore);
  private readonly lmsApi = inject(LmsApiService);

  protected readonly levels = signal<CEFRLevel[]>([]);
  protected readonly isLoadingLevels = signal(true);
  protected readonly levelsError = signal<string | null>(null);

  async ngOnInit(): Promise<void> {
    await this.loadLevels();
  }

  protected async loadLevels(): Promise<void> {
    this.isLoadingLevels.set(true);
    this.levelsError.set(null);

    try {
      const response = await firstValueFrom(this.lmsApi.getLevels());
      this.levels.set(response.levels);
    } catch {
      this.levelsError.set('Unable to reach LMS Core service. Make sure backend services are running.');
    } finally {
      this.isLoadingLevels.set(false);
    }
  }
}
