import type { CreateVariantBody, UpdateVariantBody, Variant } from '~/types/curriculum';

export function useVariants() {
  const { $api } = useNuxtApp();

  async function list(lessonId: string): Promise<Variant[]> {
    return $api<Variant[]>(`/lessons/${lessonId}/variants`);
  }

  async function create(lessonId: string, body: CreateVariantBody): Promise<Variant> {
    return $api<Variant>(`/lessons/${lessonId}/variants`, { method: 'POST', body });
  }

  async function update(variantId: string, body: UpdateVariantBody): Promise<Variant> {
    return $api<Variant>(`/variants/${variantId}`, { method: 'PATCH', body });
  }

  async function clone(variantId: string, title = ''): Promise<Variant> {
    return $api<Variant>(`/variants/${variantId}/clone`, {
      method: 'POST',
      body: { title },
    });
  }

  async function remove(variantId: string): Promise<void> {
    await $api(`/variants/${variantId}`, { method: 'DELETE' });
  }

  return { list, create, update, clone, remove };
}
