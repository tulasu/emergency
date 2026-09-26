import type { Group } from '~/types/groups';
import type { TbSelectOption } from '~/types/ui';

export function useGroups() {
  const groups = ref<Group[]>([]);
  const groupOptions = computed<TbSelectOption[]>(() =>
    groups.value.map((group) => ({ value: group.id, label: group.name })),
  );

  async function load(): Promise<void> {
    const { $api } = useNuxtApp();
    groups.value = await $api<Group[]>('/groups');
  }

  function groupName(groupId: string): string {
    return groups.value.find((group) => group.id === groupId)?.name ?? '';
  }

  return { groups, groupOptions, load, groupName };
}
