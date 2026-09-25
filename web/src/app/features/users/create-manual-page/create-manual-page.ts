import { Component, computed, inject, OnInit, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { Router } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { apiErrorMessage } from '../../../core/api/api-error';
import { AuthApi } from '../../../core/auth/auth.api';
import { Group, GroupsApi } from '../../../core/groups/groups.api';
import { ruCount } from '../../../core/users/ru-count';
import { ProvisionResultStore } from '../../../core/users/provision-result.store';
import { TbButton } from '../../../shared/ui/button/button';
import { TbField } from '../../../shared/ui/field/field';
import { TbSelect, TbSelectOption } from '../../../shared/ui/select/select';
import { filledUsers, newUserRow } from '../users-form';
import { UsersTable } from '../users-table/users-table';

@Component({
  selector: 'tb-create-manual-page',
  imports: [ReactiveFormsModule, TbButton, TbField, TbSelect, UsersTable],
  templateUrl: './create-manual-page.html',
  styleUrl: '../provision-sheet.css',
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
  readonly pending = signal(false);
  readonly error = signal('');

  readonly form = this.fb.nonNullable.group({
    group_id: ['', Validators.required],
    users: this.fb.array([newUserRow(this.fb)]),
  });

  get users() {
    return this.form.controls.users;
  }

  async ngOnInit(): Promise<void> {
    this.groups.set(await firstValueFrom(this.groupsApi.list()));
  }

  countLabel(): string {
    return ruCount(this.users.length, 'учётная запись', 'учётные записи', 'учётных записей');
  }

  addRow(): void {
    this.users.push(newUserRow(this.fb));
  }

  removeRow(index: number): void {
    this.users.removeAt(index);
    if (this.users.length === 0) {
      this.addRow();
    }
  }

  async submit(): Promise<void> {
    const filled = filledUsers(this.users);
    if (!this.form.controls.group_id.value || filled.length === 0 || this.pending()) {
      this.form.markAllAsTouched();
      this.error.set('Выберите группу и заполните хотя бы одну учётную запись');
      return;
    }
    this.pending.set(true);
    this.error.set('');
    try {
      const groupId = this.form.controls.group_id.value;
      const result = await firstValueFrom(this.authApi.provision({ group_id: groupId, users: filled }));
      this.results.set(result, this.groupName(groupId));
      await this.router.navigateByUrl('/users/credentials');
    } catch (err) {
      this.error.set(apiErrorMessage(err));
    } finally {
      this.pending.set(false);
    }
  }

  private groupName(groupId: string): string {
    return this.groups().find((group) => group.id === groupId)?.name ?? '';
  }
}
