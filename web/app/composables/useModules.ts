import type {
  CreateModuleBody,
  Module,
  ModuleScope,
  ModuleSummary,
  UpdateModuleBody,
} from '~/types/curriculum';

export function useModules() {
  const { $api } = useNuxtApp();

  async function list(q = '', scope: ModuleScope | '' = ''): Promise<Module[]> {
    return $api<Module[]>('/modules', {
      query: {
        ...(q ? { q } : {}),
        ...(scope ? { scope } : {}),
      },
    });
  }

  async function get(id: string): Promise<Module> {
    return $api<Module>(`/modules/${id}`);
  }

  async function summary(id: string): Promise<ModuleSummary> {
    return $api<ModuleSummary>(`/modules/${id}/summary`);
  }

  async function create(body: CreateModuleBody): Promise<Module> {
    return $api<Module>('/modules', { method: 'POST', body });
  }

  async function update(id: string, body: UpdateModuleBody): Promise<Module> {
    return $api<Module>(`/modules/${id}`, { method: 'PATCH', body });
  }

  async function remove(id: string): Promise<void> {
    await $api(`/modules/${id}`, { method: 'DELETE' });
  }

  return { list, get, summary, create, update, remove };
}
