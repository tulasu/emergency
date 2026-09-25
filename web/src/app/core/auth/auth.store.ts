import { computed, inject, Injectable, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';
import { AuthApi } from './auth.api';
import { User } from './auth.models';
import { TokenStorage } from './token.storage';

@Injectable({ providedIn: 'root' })
export class AuthStore {
  private readonly api = inject(AuthApi);
  private readonly tokens = inject(TokenStorage);

  readonly user = signal<User | null>(null);
  readonly token = signal<string | null>(this.tokens.read());
  readonly hydrated = signal(false);
  readonly isAuthenticated = computed(() => !!this.token() && !!this.user());

  async hydrate(): Promise<void> {
    const token = this.tokens.read();
    if (!token) {
      this.user.set(null);
      this.token.set(null);
      this.hydrated.set(true);
      return;
    }
    this.token.set(token);
    try {
      const user = await firstValueFrom(this.api.me());
      this.user.set(user);
    } catch {
      this.clear();
    } finally {
      this.hydrated.set(true);
    }
  }

  async login(login: string, password: string): Promise<void> {
    const res = await firstValueFrom(this.api.login({ login, password }));
    this.tokens.write(res.token);
    this.token.set(res.token);
    this.user.set(res.user);
  }

  async logout(): Promise<void> {
    try {
      await firstValueFrom(this.api.logout());
    } catch {
      // session is dropped locally anyway
    }
    this.clear();
  }

  clear(): void {
    this.tokens.clear();
    this.token.set(null);
    this.user.set(null);
  }
}
