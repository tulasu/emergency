import { Component, inject, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { Router } from '@angular/router';
import { apiErrorMessage } from '../../../core/api/api-error';
import { APP_SETTINGS } from '../../../core/config/app-settings';
import { AuthStore } from '../../../core/auth/auth.store';
import { TbButton } from '../../../shared/ui/button/button';
import { TbCard } from '../../../shared/ui/card/card';
import { TbField } from '../../../shared/ui/field/field';
import { TbInput } from '../../../shared/ui/input/input';

@Component({
  selector: 'tb-login-page',
  imports: [ReactiveFormsModule, TbButton, TbCard, TbField, TbInput],
  templateUrl: './login-page.html',
  styleUrl: './login-page.css',
})
export class LoginPage {
  private readonly fb = inject(FormBuilder);
  private readonly store = inject(AuthStore);
  private readonly router = inject(Router);
  readonly companyName = inject(APP_SETTINGS).companyName;
  readonly year = new Date().getFullYear();

  readonly error = signal('');
  readonly pending = signal(false);

  readonly form = this.fb.nonNullable.group({
    login: ['', [Validators.required, Validators.minLength(3), Validators.maxLength(64)]],
    password: ['', [Validators.required, Validators.minLength(8)]],
  });

  async submit(): Promise<void> {
    if (this.form.invalid || this.pending()) {
      this.form.markAllAsTouched();
      return;
    }
    this.pending.set(true);
    this.form.disable({ emitEvent: false });
    try {
      const { login, password } = this.form.getRawValue();
      await this.store.login(login, password);
      await this.router.navigateByUrl('/');
    } catch (err) {
      this.error.set(apiErrorMessage(err));
    } finally {
      this.form.enable({ emitEvent: false });
      this.pending.set(false);
    }
  }
}
