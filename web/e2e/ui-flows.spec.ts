/**
 * UI flow smoke — requires WEB_BASE_URL (Nuxt) and seeded teacher/student.
 * Visual CTA copy after MCP pixel pass for Flow B.
 */
import { expect, test } from '@playwright/test';

const teacherLogin = process.env.E2E_TEACHER_LOGIN ?? 'teacher';
const teacherPassword = process.env.E2E_TEACHER_PASSWORD ?? 'password1';

test.describe('UI curriculum navigation', () => {
  test('teacher can open modules and tickets from sidebar', async ({ page }) => {
    await page.goto('/login');
    await page.getByPlaceholder('Введите логин').fill(teacherLogin);
    await page.getByPlaceholder(/пароль/i).fill(teacherPassword);
    await page.getByRole('button', { name: /войти/i }).click();

    await page.waitForURL((url) => !url.pathname.includes('/login'), { timeout: 15_000 });

    await page.getByRole('link', { name: /модули/i }).click();
    await expect(page.getByRole('heading', { name: 'Модули' })).toBeVisible({ timeout: 15_000 });
    await expect(page.getByRole('button', { name: /создать модуль/i })).toBeVisible();
    await expect(page.getByRole('tab', { name: /^идут/i })).toBeVisible();
    await expect(page.getByText('Сначала недавно изменённые')).toBeVisible();
    await expect(page.getByRole('button', { name: /настройки/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /выйти/i })).toBeVisible();

    await page.getByRole('button', { name: /создать модуль/i }).click();
    await expect(page).toHaveURL(/\/modules\/new/);
    await expect(page.getByRole('heading', { name: 'Новый модуль' })).toBeVisible();
    await expect(page.getByText('Основное и занятия')).toBeVisible();
    await expect(page.getByText('Готовность модуля')).toBeVisible();
    await expect(page.getByRole('button', { name: /отмена/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /сохранить черновик/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /далее: назначение/i })).toBeVisible();

    await page.getByRole('link', { name: /карточки/i }).click();
    await expect(page.getByRole('heading', { name: /карточки/i })).toBeVisible({ timeout: 15_000 });
    await expect(page.getByRole('button', { name: /^создать$/i })).toBeVisible();
  });
});
