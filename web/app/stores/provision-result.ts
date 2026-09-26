import type { ProvisionResult } from '~/types/auth';

export const useProvisionResultStore = defineStore('provision-result', () => {
  const result = ref<ProvisionResult | null>(null);
  const groupName = ref('');

  function set(next: ProvisionResult, name: string): void {
    result.value = next;
    groupName.value = name;
  }

  return { result, groupName, set };
});
