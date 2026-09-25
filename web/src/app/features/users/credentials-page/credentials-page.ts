import { Component, computed, inject } from '@angular/core';
import { Router } from '@angular/router';
import { errorCodeMessage } from '../../../core/api/api-error';
import { ProvisionResultStore } from '../../../core/users/provision-result.store';
import { TbButton } from '../../../shared/ui/button/button';
import { TbCard } from '../../../shared/ui/card/card';
import { TbIcon } from '../../../shared/ui/icon/icon';

@Component({
  selector: 'tb-credentials-page',
  imports: [TbButton, TbCard, TbIcon],
  templateUrl: './credentials-page.html',
  styleUrl: './credentials-page.css',
})
export class CredentialsPage {
  private readonly results = inject(ProvisionResultStore);
  private readonly router = inject(Router);

  readonly result = this.results.result;
  readonly created = computed(() => this.result()?.created ?? []);
  readonly failed = computed(() => this.result()?.failed ?? []);

  constructor() {
    if (!this.result()) {
      void this.router.navigateByUrl('/users/new');
    }
  }

  failMessage(code: string): string {
    return errorCodeMessage(code);
  }

  async copyRow(login: string, password: string): Promise<void> {
    await navigator.clipboard.writeText(`${login}\t${password}`);
  }

  async copyAll(): Promise<void> {
    const text = this.created()
      .map((u) => `${u.full_name}\t${u.login}\t${u.password}`)
      .join('\n');
    await navigator.clipboard.writeText(text);
  }

  download(kind: 'csv' | 'txt'): void {
    const rows = this.created();
    const body =
      kind === 'csv'
        ? ['ФИО,Логин,Пароль', ...rows.map((u) => `${escapeCsv(u.full_name)},${u.login},${u.password}`)].join('\n')
        : rows.map((u) => `${u.full_name}\t${u.login}\t${u.password}`).join('\n');
    const blob = new Blob([body], { type: 'text/plain;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = kind === 'csv' ? 'accesses.csv' : 'accesses.txt';
    a.click();
    URL.revokeObjectURL(url);
  }
}

function escapeCsv(value: string): string {
  if (!/[",\n]/.test(value)) {
    return value;
  }
  return `"${value.replaceAll('"', '""')}"`;
}
