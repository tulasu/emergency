import { FormArray, FormBuilder, FormControl, FormGroup } from '@angular/forms';
import { ProvisionUserInput } from '../../core/auth/auth.models';

export type UserRowGroup = FormGroup<{
  full_name: FormControl<string>;
  login: FormControl<string>;
}>;

export function newUserRow(fb: FormBuilder, user?: ProvisionUserInput): UserRowGroup {
  return fb.nonNullable.group({
    full_name: [user?.full_name ?? ''],
    login: [user?.login ?? ''],
  });
}

export function filledUsers(users: FormArray<UserRowGroup>): ProvisionUserInput[] {
  return users.controls
    .map((control) => control.getRawValue())
    .filter((row) => row.full_name.trim() || row.login.trim())
    .map((row) => ({ full_name: row.full_name.trim(), login: row.login.trim() }));
}

export function replaceUsers(users: FormArray<UserRowGroup>, fb: FormBuilder, next: ProvisionUserInput[]): void {
  users.clear();
  for (const user of next) {
    users.push(newUserRow(fb, user));
  }
  if (users.length === 0) {
    users.push(newUserRow(fb));
  }
}
