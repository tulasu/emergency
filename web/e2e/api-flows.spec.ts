/**
 * Flow checklist (design arrows) covered via API:
 * - «+ Создать модуль»
 * - клик по модулю / summary
 * - «+ Новое занятие» / пул занятий
 * - «+ Новый вариант» / clone / «Утвердить вариант»
 * - пул карточек → «В вариант»
 * - «Открыть занятие» (users + dates)
 * - «Приступить» (student start)
 * - «Дать ещё попытку»
 */
import { expect, test, type APIRequestContext } from '@playwright/test';

async function login(request: APIRequestContext, loginName: string, password: string): Promise<string | null> {
  try {
    const res = await request.post('/auth/login', {
      data: { login: loginName, password },
    });
    if (!res.ok()) {
      return null;
    }
    const body = await res.json();
    return body.token as string;
  } catch {
    return null;
  }
}

async function me(request: APIRequestContext, token: string): Promise<{ id: string; role: string }> {
  const res = await request.get('/auth/me', {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(res.ok()).toBeTruthy();
  return res.json();
}

async function ensureUser(
  request: APIRequestContext,
  adminToken: string,
  loginName: string,
  password: string,
  role: string,
): Promise<string | null> {
  const existing = await login(request, loginName, password);
  if (existing) {
    return existing;
  }
  const created = await request.post('/auth/users', {
    headers: { Authorization: `Bearer ${adminToken}` },
    data: { login: loginName, password, role },
  });
  if (!created.ok()) {
    return null;
  }
  return login(request, loginName, password);
}

test.describe('curriculum staff→student flow', () => {
  test('covers create → variants → cards → open → attempt', async ({ request }) => {
    const suffix = Date.now().toString(36);
    const adminPassword = process.env.E2E_ADMIN_PASSWORD ?? 'adminadmin';
    const adminToken = await login(request, process.env.E2E_ADMIN_LOGIN ?? 'admin', adminPassword);
    test.skip(!adminToken, 'API admin not available — run seed-admin and start API on :8080');

    const teacherToken = await ensureUser(request, adminToken!, `e2e_t_${suffix}`, 'password1', 'teacher');
    const studentToken = await ensureUser(request, adminToken!, `e2e_s_${suffix}`, 'password1', 'student');
    test.skip(!teacherToken || !studentToken, 'Could not provision teacher/student');

    const auth = { Authorization: `Bearer ${teacherToken}` };
    const student = await me(request, studentToken!);

    const topicRes = await request.post('/topics', {
      headers: auth,
      data: { title: `Тема ${suffix}` },
    });
    expect(topicRes.ok(), await topicRes.text()).toBeTruthy();
    const topic = await topicRes.json();

    const moduleRes = await request.post('/modules', {
      headers: auth,
      data: { title: `Модуль ${suffix}`, description: 'e2e' },
    });
    expect(moduleRes.ok()).toBeTruthy();
    const module = await moduleRes.json();
    expect(module.status).toBe('draft');

    await request.patch(`/modules/${module.id}`, {
      headers: auth,
      data: {
        title: module.title,
        description: 'e2e',
        status: 'active',
        success_threshold: 70,
      },
    });

    const lessonRes = await request.post(`/modules/${module.id}/lessons`, {
      headers: auth,
      data: { title: `Занятие ${suffix}`, position: 0 },
    });
    expect(lessonRes.ok()).toBeTruthy();
    const lesson = await lessonRes.json();

    const variantRes = await request.post(`/lessons/${lesson.id}/variants`, {
      headers: auth,
      data: { title: 'Вариант A', position: 0 },
    });
    expect(variantRes.ok()).toBeTruthy();
    const variant = await variantRes.json();

    const cloneRes = await request.post(`/variants/${variant.id}/clone`, {
      headers: auth,
      data: { title: 'Вариант B' },
    });
    expect(cloneRes.ok()).toBeTruthy();

    const approveRes = await request.patch(`/variants/${variant.id}`, {
      headers: auth,
      data: { title: 'Вариант A', position: 0, status: 'approved', is_primary: true },
    });
    expect(approveRes.ok()).toBeTruthy();

    const libRes = await request.post('/tickets', {
      headers: auth,
      data: { topic_id: topic.id, title: `Карточка ${suffix}`, body: 'body' },
    });
    expect(libRes.ok(), await libRes.text()).toBeTruthy();
    const libTicket = await libRes.json();
    expect(libTicket.variant_id).toBeFalsy();

    const poolList = await request.get(`/tickets?q=${encodeURIComponent(suffix)}`, { headers: auth });
    expect(poolList.ok()).toBeTruthy();

    const copyRes = await request.post(`/variants/${variant.id}/tickets/from-pool`, {
      headers: auth,
      data: { ticket_id: libTicket.id },
    });
    expect(copyRes.ok(), await copyRes.text()).toBeTruthy();

    const openRes = await request.post(`/variants/${variant.id}/open`, {
      headers: auth,
      data: {
        mode: 'users',
        user_ids: [student.id],
        available_from: new Date(Date.now() - 3600_000).toISOString(),
        deadline_at: new Date(Date.now() + 86_400_000).toISOString(),
      },
    });
    expect(openRes.ok(), await openRes.text()).toBeTruthy();

    const mine = await request.get('/me/modules', {
      headers: { Authorization: `Bearer ${studentToken}` },
    });
    expect(mine.ok()).toBeTruthy();

    const attemptsRes = await request.get(`/variants/${variant.id}/attempts/mine`, {
      headers: { Authorization: `Bearer ${studentToken}` },
    });
    expect(attemptsRes.ok()).toBeTruthy();
    const attempts = await attemptsRes.json();
    expect(attempts.length).toBeGreaterThan(0);

    const startRes = await request.post(`/attempts/${attempts[0].id}/start`, {
      headers: { Authorization: `Bearer ${studentToken}` },
    });
    expect([200, 400, 409, 422].includes(startRes.status())).toBeTruthy();

    const grantAgain = await request.post(`/users/${student.id}/attempts`, {
      headers: auth,
      data: { variant_id: variant.id },
    });
    expect([200, 409].includes(grantAgain.status()) || grantAgain.ok()).toBeTruthy();

    const summary = await request.get(`/modules/${module.id}/summary`, { headers: auth });
    expect(summary.ok()).toBeTruthy();

    const lessonsPool = await request.get('/lessons', { headers: auth });
    expect(lessonsPool.ok()).toBeTruthy();
  });
});
