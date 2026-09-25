import Papa from 'papaparse';
import { ProvisionedUser, ProvisionUserInput } from '../auth/auth.models';

export const MAX_PROVISION_USERS = 200;

export type UsersFileKind = 'csv' | 'json';

export type ParseUsersResult =
  | { ok: true; kind: UsersFileKind; users: ProvisionUserInput[] }
  | { ok: false; error: string };

const FULL_NAME_ALIASES = new Set(['фио', 'fullname', 'name']);
const LOGIN_ALIASES = new Set(['логин', 'login']);

export function detectUsersFileKind(filename: string): UsersFileKind | null {
  const lower = filename.toLowerCase();
  if (lower.endsWith('.csv')) {
    return 'csv';
  }
  if (lower.endsWith('.json')) {
    return 'json';
  }
  return null;
}

export function parseUsersFile(content: string, filename: string): ParseUsersResult {
  const kind = detectUsersFileKind(filename) ?? guessKind(content);
  if (!kind) {
    return { ok: false, error: 'Нужен файл CSV или JSON' };
  }
  try {
    const users = kind === 'csv' ? parseUsersCsv(content) : parseUsersJson(content);
    return finishParse(kind, users);
  } catch (err) {
    return { ok: false, error: err instanceof Error ? err.message : 'Не удалось прочитать файл' };
  }
}

export function parseUsersCsv(content: string): ProvisionUserInput[] {
  const parsed = Papa.parse<Record<string, string>>(stripBom(content), {
    header: true,
    skipEmptyLines: 'greedy',
  });
  if (parsed.errors.length && !parsed.data.length && !parsed.meta.fields?.length) {
    throw new Error('Не удалось прочитать CSV');
  }
  const fields = parsed.meta.fields ?? [];
  const fullNameKey = findColumn(fields, FULL_NAME_ALIASES);
  const loginKey = findColumn(fields, LOGIN_ALIASES);
  if (!fullNameKey || !loginKey) {
    throw new Error('В файле нужны колонки ФИО и Логин');
  }
  return parsed.data
    .map((row) => ({
      full_name: String(row[fullNameKey] ?? '').trim(),
      login: String(row[loginKey] ?? '').trim(),
    }))
    .filter((row) => row.full_name || row.login);
}

export function parseUsersJson(content: string): ProvisionUserInput[] {
  let data: unknown;
  try {
    data = JSON.parse(stripBom(content));
  } catch {
    throw new Error('Не удалось прочитать JSON');
  }
  const rows = Array.isArray(data) ? data : isUsersObject(data) ? data.users : null;
  if (!rows) {
    throw new Error('В JSON нужен массив пользователей или поле users');
  }
  const users: ProvisionUserInput[] = [];
  for (const row of rows) {
    const user = readUser(row);
    if (user) {
      users.push(user);
    }
  }
  return users;
}

export function exportAccessesCsv(rows: Pick<ProvisionedUser, 'full_name' | 'login' | 'password'>[]): string {
  return `\uFEFF${Papa.unparse({
    fields: ['ФИО', 'Логин', 'Пароль'],
    data: rows.map((row) => [row.full_name, row.login, row.password]),
  })}`;
}

function finishParse(kind: UsersFileKind, users: ProvisionUserInput[]): ParseUsersResult {
  if (users.length === 0) {
    return { ok: false, error: 'В файле нет записей' };
  }
  if (users.length > MAX_PROVISION_USERS) {
    return { ok: false, error: `В одном файле не больше ${MAX_PROVISION_USERS} пользователей` };
  }
  return { ok: true, kind, users };
}

function guessKind(content: string): UsersFileKind | null {
  const trimmed = stripBom(content).trim();
  if (!trimmed) {
    return null;
  }
  if (trimmed.startsWith('{') || trimmed.startsWith('[')) {
    return 'json';
  }
  return 'csv';
}

function findColumn(fields: string[], aliases: Set<string>): string | undefined {
  return fields.find((field) => aliases.has(normalizeHeader(field)));
}

function normalizeHeader(header: string): string {
  return header.replace(/^\uFEFF/, '').trim().toLowerCase().replace(/[\s._-]+/g, '');
}

function readUser(raw: unknown): ProvisionUserInput | null {
  if (!raw || typeof raw !== 'object') {
    return null;
  }
  const rec = raw as Record<string, unknown>;
  const full_name = pickString(rec, ['full_name', 'ФИО', 'fio', 'name']);
  const login = pickString(rec, ['login', 'Логин']);
  if (!full_name && !login) {
    return null;
  }
  return { full_name, login };
}

function pickString(rec: Record<string, unknown>, keys: string[]): string {
  for (const key of keys) {
    const value = rec[key];
    if (typeof value === 'string' && value.trim()) {
      return value.trim();
    }
  }
  for (const [key, value] of Object.entries(rec)) {
    if (typeof value === 'string' && keys.some((alias) => normalizeHeader(alias) === normalizeHeader(key))) {
      return value.trim();
    }
  }
  return '';
}

function isUsersObject(data: unknown): data is { users: unknown[] } {
  return !!data && typeof data === 'object' && Array.isArray((data as { users?: unknown }).users);
}

function stripBom(content: string): string {
  return content.replace(/^\uFEFF/, '');
}
