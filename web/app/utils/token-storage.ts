const KEY = 'traineebox.session';

export function readToken(): string | null {
  if (!import.meta.client) {
    return null;
  }
  return localStorage.getItem(KEY);
}

export function writeToken(token: string): void {
  if (!import.meta.client) {
    return;
  }
  localStorage.setItem(KEY, token);
}

export function clearToken(): void {
  if (!import.meta.client) {
    return;
  }
  localStorage.removeItem(KEY);
}
