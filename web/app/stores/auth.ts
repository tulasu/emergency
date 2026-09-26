import type { LoginResponse, User } from '~/types/auth';
import { clearToken, readToken, writeToken } from '~/utils/token-storage';

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null);
  const token = ref<string | null>(readToken());
  const hydrated = ref(false);
  const isAuthenticated = computed(() => !!token.value && !!user.value);

  async function hydrate(): Promise<void> {
    const saved = readToken();
    if (!saved) {
      user.value = null;
      token.value = null;
      hydrated.value = true;
      return;
    }
    token.value = saved;
    try {
      const { $api } = useNuxtApp();
      user.value = await $api<User>('/auth/me');
    } catch {
      clear();
    } finally {
      hydrated.value = true;
    }
  }

  async function login(loginName: string, password: string): Promise<void> {
    const { $api } = useNuxtApp();
    const res = await $api<LoginResponse>('/auth/login', {
      method: 'POST',
      body: { login: loginName, password },
    });
    writeToken(res.token);
    token.value = res.token;
    user.value = res.user;
  }

  async function logout(): Promise<void> {
    try {
      const { $api } = useNuxtApp();
      await $api('/auth/logout', { method: 'POST', body: {} });
    } catch {
      // session is dropped locally anyway
    }
    clear();
  }

  function clear(): void {
    clearToken();
    token.value = null;
    user.value = null;
  }

  return { user, token, hydrated, isAuthenticated, hydrate, login, logout, clear };
});
