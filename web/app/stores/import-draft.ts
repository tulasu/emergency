import type { ProvisionUserInput } from '~/types/auth';
import type { UsersFileKind } from '~/utils/users-file';

export interface ImportDraft {
  fileName: string;
  kind: UsersFileKind;
  users: ProvisionUserInput[];
}

export const useImportDraftStore = defineStore('import-draft', () => {
  const draft = ref<ImportDraft | null>(null);

  function set(next: ImportDraft): void {
    draft.value = next;
  }

  function clear(): void {
    draft.value = null;
  }

  return { draft, set, clear };
});
