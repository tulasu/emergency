import type { Group, GroupMember, UserProfile } from '~/types/groups';

export function useGroups() {
  const groups = ref<Group[]>([]);
  const groupOptions = computed(() =>
    groups.value.map((group) => ({ value: group.id, label: group.name })),
  );

  async function load(): Promise<void> {
    const { $api } = useNuxtApp();
    groups.value = await $api<Group[]>('/groups');
  }

  function groupName(groupId: string): string {
    return groups.value.find((group) => group.id === groupId)?.name ?? '';
  }

  async function get(groupId: string): Promise<Group> {
    const { $api } = useNuxtApp();
    return $api<Group>(`/groups/${groupId}`);
  }

  async function create(name: string): Promise<Group> {
    const { $api } = useNuxtApp();
    return $api<Group>('/groups', { method: 'POST', body: { name } });
  }

  async function rename(groupId: string, name: string): Promise<Group> {
    const { $api } = useNuxtApp();
    return $api<Group>(`/groups/${groupId}`, { method: 'PATCH', body: { name } });
  }

  async function remove(groupId: string): Promise<void> {
    const { $api } = useNuxtApp();
    await $api(`/groups/${groupId}`, { method: 'DELETE' });
  }

  async function addMember(
    groupId: string,
    userId: string,
    role: GroupMember['role'],
  ): Promise<void> {
    const { $api } = useNuxtApp();
    await $api(`/groups/${groupId}/members`, {
      method: 'POST',
      body: { user_id: userId, role },
    });
  }

  async function removeMember(groupId: string, userId: string): Promise<void> {
    const { $api } = useNuxtApp();
    await $api(`/groups/${groupId}/members/${userId}`, { method: 'DELETE' });
  }

  return {
    groups,
    groupOptions,
    load,
    groupName,
    get,
    create,
    rename,
    remove,
    addMember,
    removeMember,
  };
}

export function useUsers() {
  const { $api } = useNuxtApp();

  async function list(): Promise<UserProfile[]> {
    return $api<UserProfile[]>('/auth/users');
  }

  async function get(userId: string): Promise<UserProfile> {
    return $api<UserProfile>(`/auth/users/${userId}`);
  }

  async function setBlocked(userId: string, blocked: boolean): Promise<void> {
    await $api(`/auth/users/${userId}/block`, {
      method: 'POST',
      body: { blocked },
    });
  }

  async function setRole(userId: string, role: string): Promise<void> {
    await $api(`/auth/users/${userId}/role`, {
      method: 'POST',
      body: { role },
    });
  }

  async function changePassword(current: string, next: string): Promise<void> {
    await $api('/auth/me/password', {
      method: 'POST',
      body: { current_password: current, new_password: next },
    });
  }

  return { list, get, setBlocked, setRole, changePassword };
}
