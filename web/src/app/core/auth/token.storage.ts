import { Injectable } from '@angular/core';

const KEY = 'traineebox.session';

@Injectable({ providedIn: 'root' })
export class TokenStorage {
  read(): string | null {
    return localStorage.getItem(KEY);
  }

  write(token: string): void {
    localStorage.setItem(KEY, token);
  }

  clear(): void {
    localStorage.removeItem(KEY);
  }
}
