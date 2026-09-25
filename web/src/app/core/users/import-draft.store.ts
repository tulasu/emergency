import { Injectable, signal } from '@angular/core';
import { ProvisionUserInput } from '../auth/auth.models';
import { UsersFileKind } from './users-file';

export interface ImportDraft {
  fileName: string;
  kind: UsersFileKind;
  users: ProvisionUserInput[];
}

@Injectable({ providedIn: 'root' })
export class ImportDraftStore {
  readonly draft = signal<ImportDraft | null>(null);

  set(draft: ImportDraft): void {
    this.draft.set(draft);
  }

  clear(): void {
    this.draft.set(null);
  }
}
