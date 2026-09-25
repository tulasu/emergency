import { Component, computed, inject, OnInit, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { Router } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { apiErrorMessage, errorCodeMessage } from '../../../core/api/api-error';
import { AuthApi } from '../../../core/auth/auth.api';
import { Group, GroupsApi } from '../../../core/groups/groups.api';
import { ProvisionResultStore } from '../../../core/users/provision-result.store';
import { TbButton } from '../../../shared/ui/button/button';
import { TbCheckRow } from '../../../shared/ui/check-row/check-row';
import { TbField } from '../../../shared/ui/field/field';
import { TbInput } from '../../../shared/ui/input/input';
import { TbSelect, TbSelectOption } from '../../../shared/ui/select/select';

@Component({
  selector: 'tb-create-manual-page',
  imports: [ReactiveFormsModule, TbButton, TbField, TbCheckRow, TbInput, TbSelect],
  templateUrl: './create-manual-page.html',
  styleUrl: './create-manual-page.css',
})
export class CreateManualPage implements OnInit {
  private readonly fb = inject(FormBuilder);
  private readonly groupsApi = inject(GroupsApi);
  private readonly authApi = inject(AuthApi);
  private readonly results = inject(ProvisionResultStore);
  private readonly router = inject(Router);

  readonly groups = signal<Group[]>([]);
  readonly groupOptions = computed<TbSelectOption[]>(() =>
    this.groups().map((group) => ({ value: group.id, label: group.name })),
  );
  readonly generate = signal(true);
  readonly pending = signal(false);
  readonly error = signal('');

  readonly form = this.fb.nonNullable.group({
    full_name: ['', [Validators.required, Validators.maxLength(128)]],
    login: ['', [Validators.required, Validators.minLength(3), Validators.maxLength(64)]],
    group_id: ['', Validators.required],
  });

  async ngOnInit(): Promise<void> {
    this.groups.set(await firstValueFrom(this.groupsApi.list()));
  }

  async submit(): Promise<void> {
    if (this.form.invalid || this.pending() || !this.generate()) {
      this.form.markAllAsTouched();
      return;
    }
    this.pending.set(true);
    this.error.set('');
    try {
      const value = this.form.getRawValue();
      const result = await firstValueFrom(
        this.authApi.provision({
          group_id: value.group_id,
          users: [{ full_name: value.full_name, login: value.login }],
        }),
      );
      if (result.created.length === 0) {
        this.error.set(errorCodeMessage(result.failed[0]?.error));
        return;
      }
      this.results.result.set(result);
      await this.router.navigateByUrl('/users/credentials');
    } catch (err) {
      this.error.set(apiErrorMessage(err));
    } finally {
      this.pending.set(false);
    }
  }
}
