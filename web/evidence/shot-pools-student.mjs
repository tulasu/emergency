import { chromium } from '@playwright/test';
import { mkdirSync } from 'node:fs';
import { resolve } from 'node:path';

const webBase = process.env.WEB_BASE_URL ?? 'http://localhost:3000';
mkdirSync(resolve('evidence/live'), { recursive: true });

const browser = await chromium.launch({
  headless: true,
  channel: 'chrome',
});

async function login(page, login, password) {
  await page.goto(`${webBase}/login`, { waitUntil: 'domcontentloaded' });
  await page.getByPlaceholder(/логин/i).fill(login);
  await page.getByPlaceholder(/пароль/i).fill(password);
  await page.getByRole('button', { name: /войти/i }).click();
  await page.waitForTimeout(1500);
}

async function spaGo(page, path) {
  await page.evaluate(async (p) => {
    const app = document.querySelector('#__nuxt')?.__vue_app__;
    const router = app?.config?.globalProperties?.$router;
    if (router?.push) {
      await router.push(p);
      return;
    }
    history.pushState({}, '', p);
    window.dispatchEvent(new PopStateEvent('popstate'));
  }, path);
  await page.waitForTimeout(1800);
}

async function shot(loginName, password, paths) {
  const page = await browser.newPage({ viewport: { width: 1920, height: 900 } });
  await login(page, loginName, password);
  await page.goto(`${webBase}/`, { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(800);

  for (const { path, out } of paths) {
    await spaGo(page, path);
    await page.screenshot({ path: resolve(out), fullPage: false });
    const text = (await page.locator('body').innerText()).slice(0, 400).replace(/\s+/g, ' ');
    console.log(JSON.stringify({ login: loginName, url: page.url(), out, text }));
  }
  await page.close();
}

await shot('teacher', 'password1', [
  { path: '/tickets', out: 'evidence/live/ghGs1-tickets.png' },
  { path: '/lessons', out: 'evidence/live/umr5M-lessons.png' },
]);

await shot('student1', 'student1', [
  { path: '/', out: 'evidence/live/F1NUvg-student-home.png' },
]);

{
  const page = await browser.newPage({ viewport: { width: 1920, height: 900 } });
  await login(page, 'student1', 'student1');
  await page.goto(`${webBase}/`, { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(1500);

  const open = page.getByRole('button', { name: /Открыть модуль|Приступить|Разбор/i }).first();
  if (await open.count()) {
    await open.click();
    await page.waitForTimeout(1500);
  } else {
    // invent a module path if assigned list has rows
    const row = page.locator('.module-row').first();
    if (await row.count()) {
      await row.click();
      await page.waitForTimeout(1500);
    }
  }

  await page.screenshot({
    path: resolve('evidence/live/azbHm-student-module.png'),
    fullPage: false,
  });
  console.log(JSON.stringify({ url: page.url(), out: 'evidence/live/azbHm-student-module.png' }));
  await page.close();
}

await browser.close();
