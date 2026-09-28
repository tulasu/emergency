import { describe, expect, it } from 'vitest';
import {
  attemptStatusLabel,
  audioStatusLabel,
  compareVariantHeader,
  formatOpened,
  formatOpenedFor,
  formatPassedFraction,
  formatSuccessRate,
  lessonCountLabel,
  moduleDisplayStatus,
  moduleStatusLabel,
  ruGroups,
  ruUsers,
  shortVariantKey,
  variantLabels,
  variantStatusLabel,
} from './curriculum-labels';

describe('curriculum labels', () => {
  it('maps module statuses', () => {
    expect(moduleStatusLabel('draft')).toBe('Черновик');
    expect(moduleStatusLabel('active')).toBe('Идёт');
    expect(moduleStatusLabel('completed')).toBe('Завершён');
    expect(moduleStatusLabel('archived')).toBe('Архив');
  });

  it('derives completed display status', () => {
    expect(
      moduleDisplayStatus({
        id: '1',
        title: 't',
        description: '',
        status: 'active',
        success_threshold: 70,
        created_by: 'u',
        created_at: '',
        opened_done: 4,
        opened_total: 4,
      }),
    ).toBe('completed');
  });

  it('maps variant statuses', () => {
    expect(variantStatusLabel('draft')).toBe('Черновик');
    expect(variantStatusLabel('approved')).toBe('Утверждён');
  });

  it('formats lesson counts', () => {
    expect(lessonCountLabel(1)).toBe('1 занятие');
    expect(lessonCountLabel(2)).toBe('2 занятия');
    expect(lessonCountLabel(5)).toBe('5 занятий');
  });

  it('formats groups and users', () => {
    expect(ruGroups(1)).toBe('1 группа');
    expect(ruGroups(3)).toBe('3 группы');
    expect(ruUsers(1)).toBe('1 ученик');
    expect(ruUsers(3)).toBe('3 ученика');
  });

  it('formats opened and success', () => {
    expect(formatOpened(2, 5)).toBe('2/5');
    expect(formatOpened(null, null)).toBe('—');
    expect(formatSuccessRate(71.2)).toBe('71%');
    expect(formatSuccessRate(null)).toBe('—');
    expect(formatOpenedFor(0, 38)).toEqual({ label: 'не открыто', tone: 'neutral' });
    expect(formatOpenedFor(38, 38)).toEqual({ label: 'всем · 38', tone: 'info' });
    expect(formatOpenedFor(12, 38)).toEqual({ label: '12 из 38', tone: 'warning' });
    expect(formatPassedFraction(50, 38)).toBe('19 / 38');
    expect(formatPassedFraction(null, 38)).toBe('—');
  });

  it('maps attempt and audio statuses', () => {
    expect(attemptStatusLabel('granted')).toBe('Можно приступить');
    expect(attemptStatusLabel('available')).toBe('Можно приступить');
    expect(attemptStatusLabel('finished')).toBe('Сдано');
    expect(audioStatusLabel('ready')).toBe('Аудио готово');
    expect(audioStatusLabel('')).toBe('Без аудио');
  });

  it('formats short variant labels', () => {
    expect(shortVariantKey('Вариант A')).toBe('A');
    expect(
      variantLabels(
        [
          { title: 'Вариант A', is_primary: true, status: 'approved' },
          { title: 'Вариант B', status: 'approved' },
          { title: 'Вариант C', status: 'draft' },
        ],
        'short',
      ),
    ).toBe('A★, B, C (черновик)');
    expect(
      compareVariantHeader({ title: 'Вариант A', is_primary: true }),
    ).toBe('A ★');
  });
});
