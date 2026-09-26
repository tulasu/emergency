import type { ProvisionUserInput } from '~/types/auth';

export interface UserRow {
  full_name: string;
  login: string;
}

export function emptyRow(): UserRow {
  return { full_name: '', login: '' };
}

export function filledUsers(users: UserRow[]): ProvisionUserInput[] {
  return users
    .filter((row) => row.full_name.trim() || row.login.trim())
    .map((row) => ({ full_name: row.full_name.trim(), login: row.login.trim() }));
}

export function replaceUsers(next: ProvisionUserInput[]): UserRow[] {
  if (next.length === 0) {
    return [emptyRow()];
  }
  return next.map((user) => ({ full_name: user.full_name, login: user.login }));
}
