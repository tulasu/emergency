import type { CreateTicketBody, Ticket } from '~/types/curriculum';

export function useTickets() {
  const { $api } = useNuxtApp();

  async function library(q = ''): Promise<Ticket[]> {
    return $api<Ticket[]>('/tickets', {
      query: q ? { q } : undefined,
    });
  }

  async function createLibrary(body: CreateTicketBody): Promise<Ticket> {
    return $api<Ticket>('/tickets', { method: 'POST', body });
  }

  async function listByVariant(variantId: string): Promise<Ticket[]> {
    return $api<Ticket[]>(`/variants/${variantId}/tickets`);
  }

  async function copyToVariant(variantId: string, ticketId: string): Promise<Ticket> {
    return $api<Ticket>(`/variants/${variantId}/tickets/from-pool`, {
      method: 'POST',
      body: { ticket_id: ticketId },
    });
  }

  async function get(ticketId: string): Promise<Ticket> {
    return $api<Ticket>(`/tickets/${ticketId}`);
  }

  async function update(ticketId: string, body: CreateTicketBody): Promise<Ticket> {
    return $api<Ticket>(`/tickets/${ticketId}`, { method: 'PATCH', body });
  }

  return { library, createLibrary, listByVariant, copyToVariant, get, update };
}
