import { Component, inject, signal } from '@angular/core';
import { Router, RouterLink } from '@angular/router';
import { ImportDraftStore } from '../../../core/users/import-draft.store';
import { parseUsersFile } from '../../../core/users/users-file';
import { TbIcon } from '../../../shared/ui/icon/icon';

@Component({
  selector: 'tb-new-users-page',
  imports: [RouterLink, TbIcon],
  templateUrl: './new-users-page.html',
  styleUrl: './new-users-page.css',
})
export class NewUsersPage {
  private readonly drafts = inject(ImportDraftStore);
  private readonly router = inject(Router);

  readonly error = signal('');

  async onFile(event: Event): Promise<void> {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    input.value = '';
    if (!file) {
      return;
    }
    const parsed = parseUsersFile(await file.text(), file.name);
    if (!parsed.ok) {
      this.error.set(parsed.error);
      return;
    }
    this.error.set('');
    this.drafts.set({ fileName: file.name, kind: parsed.kind, users: parsed.users });
    await this.router.navigateByUrl('/users/new/import');
  }
}
