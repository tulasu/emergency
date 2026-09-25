import { Component, computed, inject, OnInit, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { Router } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { apiErrorMessage } from '../../../core/api/api-error';
import { AuthApi } from '../../../core/auth/auth.api';
import { Group, GroupsApi } from '../../../core/groups/groups.api';
import { ImportDraft, ImportDraftStore } from '../../../core/users/import-draft.store';
import { ruCount } from '../../../core/users/ru-count';
import { ProvisionResultStore } from '../../../core/users/provision-result.store';
import { parseUsersFile, UsersFileKind } from '../../../core/users/users-file';
import { TbButton } from '../../../shared/ui/button/button';
import { TbField } from '../../../shared/ui/field/field';
import { TbIcon } from '../../../shared/ui/icon/icon';
import { TbSelect, TbSelectOption } from '../../../shared/ui/select/select';
import { filledUsers, newUserRow, replaceUsers } from '../users-form';
import { UsersTable } from '../users-table/users-table';

@Component({
  selector: 'tb-import-users-page',
  imports: [ReactiveFormsModule, TbButton, TbField, TbIcon, TbSelect, UsersTable],
  templateUrl: './import-users-page.html',
  styleUrl: '../provision-sheet.css',
})
export class ImportUsersPage implements OnInit {
  private readonly fb = inject(FormBuilder);
  private readonly groupsApi = inject(GroupsApi);
  private readonly authApi = inject(AuthApi);
  private readonly results = inject(ProvisionResultStore);
  private readonly drafts = inject(ImportDraftStore);
  private readonly router = inject(Router);

  readonly groups = signal<Group[]>([]);
  readonly groupOptions = computed<TbSelectOption[]>(() =>
    this.groups().map((group) => ({ value: group.id, label: group.name })),
  );
  readonly pending = signal(false);
  readonly error = signal('');
  readonly fileName = signal('');
  readonly fileKind = signal<UsersFileKind | null>(null);

  readonly form = this.fb.nonNullable.group({
    group_id: ['', Validators.required],
    users: this.fb.array([newUserRow(this.fb)]),
  });

  get users() {
    return this.form.controls.users;
  }

  async ngOnInit(): Promise<void> {
    const draft = this.drafts.draft();
    if (!draft) {
      await this.router.navigateByUrl('/users/new', { replaceUrl: true });
      return;
    }
    this.applyDraft(draft);
    this.groups.set(await firstValueFrom(this.groupsApi.list()));
  }

  fileMeta(): string {
    const kind = this.fileKind() === 'json' ? 'JSON' : 'CSV';
    return `${kind} · ${ruCount(this.users.length, 'запись', 'записи', 'записей')}`;
  }

  countLabel(): string {
    const base = ruCount(this.users.length, 'учётка', 'учётки', 'учёток');
    return this.fileName() ? `${base} из ${this.fileName()}` : base;
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

  async onFile(event: Event): Promise<void> {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    input.value = '';
    if (!file) {
      return;
    }
    const content = await file.text();
    const parsed = parseUsersFile(content, file.name);
    if (!parsed.ok) {
      this.error.set(parsed.error);
      return;
    }
    this.error.set('');
    const draft = { fileName: file.name, kind: parsed.kind, users: parsed.users };
    this.drafts.set(draft);
    this.applyDraft(draft);
  }

  async submit(): Promise<void> {
    const filled = filledUsers(this.users);
    if (!this.form.controls.group_id.value || filled.length === 0 || this.pending()) {
      this.form.markAllAsTouched();
      this.error.set('Загрузите файл, выберите группу и оставьте хотя бы одну строку');
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

  private applyDraft(draft: ImportDraft): void {
    this.fileName.set(draft.fileName);
    this.fileKind.set(draft.kind);
    replaceUsers(this.users, this.fb, draft.users);
  }

  private groupName(groupId: string): string {
    return this.groups().find((group) => group.id === groupId)?.name ?? '';
  }
}
