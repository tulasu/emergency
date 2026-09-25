import { Injectable, signal } from '@angular/core';
import { ProvisionResult } from '../auth/auth.models';

@Injectable({ providedIn: 'root' })
export class ProvisionResultStore {
  readonly result = signal<ProvisionResult | null>(null);
  readonly groupName = signal('');

  set(result: ProvisionResult, groupName: string): void {
    this.result.set(result);
    this.groupName.set(groupName);
  }
}
