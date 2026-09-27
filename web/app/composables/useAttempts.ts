import type {
  Attempt,
  AssignedModule,
  GrantAttemptBody,
  OpenVariantBody,
} from '~/types/curriculum';

export function useAttempts() {
  const { $api } = useNuxtApp();

  async function grant(userId: string, body: GrantAttemptBody): Promise<Attempt> {
    return $api<Attempt>(`/users/${userId}/attempts`, { method: 'POST', body });
  }

  async function openVariant(
    variantId: string,
    body: OpenVariantBody,
  ): Promise<{ granted: number }> {
    return $api<{ granted: number }>(`/variants/${variantId}/open`, {
      method: 'POST',
      body,
    });
  }

  async function start(attemptId: string): Promise<Attempt> {
    return $api<Attempt>(`/attempts/${attemptId}/start`, { method: 'POST', body: {} });
  }

  async function listMine(variantId: string): Promise<Attempt[]> {
    return $api<Attempt[]>(`/variants/${variantId}/attempts/mine`);
  }

  async function get(attemptId: string): Promise<Attempt> {
    return $api<Attempt>(`/attempts/${attemptId}`);
  }

  async function listMyModules(): Promise<AssignedModule[]> {
    return $api<AssignedModule[]>('/me/modules');
  }

  return { grant, openVariant, start, listMine, get, listMyModules };
}
