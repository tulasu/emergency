import { Component, computed, inject, OnInit, signal } from '@angular/core';
import { FormArray, FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { Router } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { apiErrorMessage } from '../../../core/api/api-error';
import { AuthApi } from '../../../core/auth/auth.api';
import { Group, GroupsApi } from '../../../core/groups/groups.api';
import { ProvisionResultStore } from '../../../core/users/provision-result.store';
import { TbButton } from '../../../shared/ui/button/button';
import { TbCard } from '../../../shared/ui/card/card';
import { TbField } from '../../../shared/ui/field/field';
import { TbInput } from '../../../shared/ui/input/input';
import { TbSelect, TbSelectOption } from '../../../shared/ui/select/select';

@Component({
  selector: 'tb-create-batch-page',
  imports: [ReactiveFormsModule, TbButton, TbCard, TbField, TbInput, TbSelect],
  templateUrl: './create-batch-page.html',
  styleUrl: './create-batch-page.css',
})
export class CreateBatchPage implements OnInit {
  private readonly fb = inject(FormBuilder);
  private readonly groupsApi = inject(GroupsApi);
  private readonly authApi = inject(AuthApi);
  private readonly results = inject(ProvisionResultStore);
  private readonly router = inject(Router);

  readonly groups = signal<Group[]>([]);
  readonly groupOptions = computed<TbSelectOption[]>(() =>
    this.groups().map((group) => ({ value: group.id, label: group.name })),
  );
  readonly pending = signal(false);
  readonly error = signal('');

  readonly form = this.fb.nonNullable.group({
    group_id: ['', Validators.required],
    users: this.fb.array([this.newRow(), this.newRow(), this.newRow(), this.newRow(), this.newRow()]),
  });

  get users(): FormArray {
    return this.form.controls.users;
  }

  async ngOnInit(): Promise<void> {
    this.groups.set(await firstValueFrom(this.groupsApi.list()));
  }

  addRow(): void {
    this.users.push(this.newRow());
  }

  async submit(): Promise<void> {
    const filled = this.users.controls
      .map((c) => c.getRawValue() as { full_name: string; login: string })
      .filter((row) => row.full_name.trim() || row.login.trim());
    if (!this.form.controls.group_id.value || filled.length === 0 || this.pending()) {
      this.form.markAllAsTouched();
      this.error.set('Заполните группу и хотя бы одну строку');
      return;
    }
    this.pending.set(true);
    this.error.set('');
    try {
      const result = await firstValueFrom(
        this.authApi.provision({
          group_id: this.form.controls.group_id.value,
          users: filled,
        }),
      );
      this.results.result.set(result);
      await this.router.navigateByUrl('/users/credentials');
    } catch (err) {
      this.error.set(apiErrorMessage(err));
    } finally {
      this.pending.set(false);
    }
  }

  private newRow() {
    return this.fb.nonNullable.group({
      full_name: [''],
      login: [''],
    });
  }
}
