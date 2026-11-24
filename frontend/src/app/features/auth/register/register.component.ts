import { ChangeDetectionStrategy, Component, OnInit, inject, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatSelectModule } from '@angular/material/select';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { Router, RouterLink } from '@angular/router';
import { CEFRCode } from '../../../core/models/auth.model';
import { AuthStore } from '../../../core/store/auth.store';

interface LevelOption {
  code: CEFRCode;
  label: string;
}

@Component({
  selector: 'app-register',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    ReactiveFormsModule,
    RouterLink,
    MatCardModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule,
    MatSnackBarModule,
  ],
  styleUrl: './register.component.scss',
  templateUrl: './register.component.html',
})
export class RegisterComponent implements OnInit {
  protected readonly authStore = inject(AuthStore);
  private readonly fb = inject(FormBuilder);
  private readonly router = inject(Router);
  private readonly snackBar = inject(MatSnackBar);

  protected readonly hidePassword = signal(true);

  protected readonly levelOptions: LevelOption[] = [
    { code: 'A1', label: 'A1 — Beginner' },
    { code: 'A2', label: 'A2 — Elementary' },
    { code: 'B1', label: 'B1 — Intermediate' },
    { code: 'B2', label: 'B2 — Upper Intermediate' },
    { code: 'C1', label: 'C1 — Advanced' },
    { code: 'C2', label: 'C2 — Proficiency' },
  ];

  protected readonly form = this.fb.nonNullable.group({
    full_name: ['', [Validators.required, Validators.minLength(2)]],
    email: ['', [Validators.required, Validators.email]],
    password: ['', [Validators.required, Validators.minLength(8), Validators.maxLength(72)]],
    cefr_level: ['A1' as CEFRCode, [Validators.required]],
  });

  ngOnInit(): void {
    this.authStore.clearError();
  }

  protected togglePasswordVisibility(): void {
    this.hidePassword.update((v) => !v);
  }

  protected async onSubmit(): Promise<void> {
    if (this.form.invalid) return;

    const success = await this.authStore.register(this.form.getRawValue());
    if (success) {
      this.snackBar.open('Account created! Please sign in.', 'OK', { duration: 4000 });
      await this.router.navigate(['/login']);
    }
  }
}
