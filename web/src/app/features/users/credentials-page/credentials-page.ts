import { Component, computed, inject } from '@angular/core';
import { Router } from '@angular/router';
import { errorCodeMessage } from '../../../core/api/api-error';
import { ruCount } from '../../../core/users/ru-count';
import { ProvisionResultStore } from '../../../core/users/provision-result.store';
import { exportAccessesCsv } from '../../../core/users/users-file';
import { TbButton } from '../../../shared/ui/button/button';
import { TbIcon } from '../../../shared/ui/icon/icon';

@Component({
  selector: 'tb-credentials-page',
  imports: [TbButton, TbIcon],
  templateUrl: './credentials-page.html',
  styleUrl: './credentials-page.css',
})
export class CredentialsPage {
  private readonly results = inject(ProvisionResultStore);
  private readonly router = inject(Router);

  readonly result = this.results.result;
  readonly groupName = this.results.groupName;
  readonly created = computed(() => this.result()?.created ?? []);
  readonly failed = computed(() => this.result()?.failed ?? []);

  constructor() {
    if (!this.result()) {
      void this.router.navigateByUrl('/users/new', { replaceUrl: true });
    }
  }

  subtitle(): string {
    const count = ruCount(this.created().length, 'пользователь', 'пользователя', 'пользователей');
    const group = this.groupName();
    const groupPart = group ? ` · группа ${group}` : '';
    return `${count}${groupPart}. Пароли больше не покажем — скопируйте или скачайте сейчас.`;
  }

  failMessage(code: string): string {
    return errorCodeMessage(code);
  }

  async copyRow(login: string, password: string): Promise<void> {
    await navigator.clipboard.writeText(`${login}\t${password}`);
  }

  async copyAll(): Promise<void> {
    const text = this.created()
      .map((user) => `${user.full_name}\t${user.login}\t${user.password}`)
      .join('\n');
    await navigator.clipboard.writeText(text);
  }

  download(kind: 'csv' | 'txt'): void {
    const rows = this.created();
    const body =
      kind === 'csv' ? exportAccessesCsv(rows) : rows.map((user) => `${user.full_name}\t${user.login}\t${user.password}`).join('\n');
    const blob = new Blob([body], { type: 'text/plain;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = kind === 'csv' ? 'accesses.csv' : 'accesses.txt';
    a.click();
    URL.revokeObjectURL(url);
  }
}
