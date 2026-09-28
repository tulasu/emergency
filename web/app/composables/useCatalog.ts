import type {
  EmergencyService,
  IncidentType,
  ReferenceAnswer,
  TagGroup,
} from '~/types/curriculum';

export function useCatalog() {
  const { $api } = useNuxtApp();

  async function incidentTypes(): Promise<IncidentType[]> {
    return $api<IncidentType[]>('/catalog/incident-types');
  }

  async function tags(typeCode: string): Promise<TagGroup[]> {
    return $api<TagGroup[]>(`/catalog/incident-types/${typeCode}/tags`);
  }

  async function services(): Promise<EmergencyService[]> {
    return $api<EmergencyService[]>('/catalog/services');
  }

  async function recommend(typeCode: string, tagCodes: string[]): Promise<string[]> {
    const res = await $api<{ service_codes: string[] }>('/catalog/recommend', {
      query: { type: typeCode, tags: tagCodes.join(',') },
    });
    return res.service_codes || [];
  }

  async function getReference(ticketId: string): Promise<ReferenceAnswer | null> {
    try {
      return await $api<ReferenceAnswer>(`/tickets/${ticketId}/reference`);
    } catch {
      return null;
    }
  }

  async function setReference(
    ticketId: string,
    body: Omit<ReferenceAnswer, 'ticket_id'>,
  ): Promise<ReferenceAnswer> {
    return $api<ReferenceAnswer>(`/tickets/${ticketId}/reference`, {
      method: 'PUT',
      body,
    });
  }

  return { incidentTypes, tags, services, recommend, getReference, setReference };
}
