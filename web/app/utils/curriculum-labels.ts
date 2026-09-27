import type { Module, ModuleDisplayStatus, ModuleStatus, VariantStatus } from '~/types/curriculum';
import type { TbBadgeTone } from '~/types/ui';

export function moduleDisplayStatus(module: Module): ModuleDisplayStatus {
  if (module.status === 'draft' || module.status === 'archived') {
    return module.status;
  }
  if (
    module.opened_total != null &&
    module.opened_total > 0 &&
    module.opened_done === module.opened_total
  ) {
    return 'completed';
  }
  return 'active';
}

export function moduleStatusLabel(status: ModuleDisplayStatus | ModuleStatus | string): string {
  switch (status) {
    case 'draft':
      return 'Черновик';
    case 'active':
      return 'Идёт';
    case 'completed':
      return 'Завершён';
    case 'archived':
      return 'Архив';
    default:
      return status;
  }
}

export function moduleStatusTone(status: ModuleDisplayStatus | ModuleStatus | string): TbBadgeTone {
  switch (status) {
    case 'active':
      return 'info';
    case 'completed':
      return 'success';
    case 'archived':
      return 'neutral';
    case 'draft':
      return 'warning';
    default:
      return 'info';
  }
}

export function variantStatusLabel(status: VariantStatus | string): string {
  switch (status) {
    case 'approved':
      return 'Утверждён';
    case 'draft':
      return 'Черновик';
    default:
      return status;
  }
}

export function variantStatusTone(status: VariantStatus | string): TbBadgeTone {
  return status === 'approved' ? 'success' : 'warning';
}

export function lessonCountLabel(count: number): string {
  const mod10 = count % 10;
  const mod100 = count % 100;
  if (mod10 === 1 && mod100 !== 11) {
    return `${count} занятие`;
  }
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) {
    return `${count} занятия`;
  }
  return `${count} занятий`;
}

export function ruGroups(count: number): string {
  const mod10 = count % 10;
  const mod100 = count % 100;
  if (mod10 === 1 && mod100 !== 11) {
    return `${count} группа`;
  }
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) {
    return `${count} группы`;
  }
  return `${count} групп`;
}

export function ruUsers(count: number): string {
  const mod10 = count % 10;
  const mod100 = count % 100;
  if (mod10 === 1 && mod100 !== 11) {
    return `${count} ученик`;
  }
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) {
    return `${count} ученика`;
  }
  return `${count} учеников`;
}

export function formatOpened(done?: number | null, total?: number | null): string {
  if (total == null) {
    return '—';
  }
  return `${done ?? 0}/${total}`;
}

export function formatSuccessRate(rate?: number | null): string {
  if (rate == null) {
    return '—';
  }
  return `${Math.round(rate)}%`;
}

export function attemptStatusLabel(status?: string | null): string {
  switch (status) {
    case 'granted':
    case 'available':
      return 'Можно приступить';
    case 'in_progress':
      return 'В работе';
    case 'submitted':
    case 'scored':
    case 'finished':
      return 'Сдано';
    case 'expired':
      return 'Просрочено';
    default:
      return status || 'Не открыто';
  }
}

export function isOpenAttemptStatus(status?: string | null): boolean {
  return status === 'granted' || status === 'available' || status === 'in_progress';
}

export function audioStatusLabel(status?: string | null): string {
  switch (status) {
    case 'ready':
      return 'Аудио готово';
    case 'pending':
      return 'Аудио в очереди';
    case 'failed':
      return 'Ошибка аудио';
    case 'none':
    case '':
    case undefined:
    case null:
      return 'Без аудио';
    default:
      return status;
  }
}

export function variantLabels(
  variants: Array<{ title: string; is_primary?: boolean; status?: string }>,
): string {
  if (!variants.length) {
    return '—';
  }
  return variants
    .map((v) => {
      const star = v.is_primary ? '★' : '';
      const draft = v.status === 'draft' ? ' черн.' : '';
      return `${v.title}${star}${draft}`.trim();
    })
    .join(' · ');
}
