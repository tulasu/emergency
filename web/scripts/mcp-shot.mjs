import { chromium } from '@playwright/test';
import { mkdirSync } from 'node:fs';
import { dirname, resolve } from 'node:path';

const webBase = process.env.WEB_BASE_URL ?? 'http://localhost:3000';
const login = process.env.SHOT_LOGIN ?? 'admin';
const password = process.env.SHOT_PASSWORD ?? 'adminadmin';
const path = process.argv[2] ?? '/modules';
const out = resolve(process.argv[3] ?? 'evidence/live/shot.png');

mkdirSync(dirname(out), { recursive: true });

const browser = await chromium.launch({ headless: true });
const page = await browser.newPage({ viewport: { width: 1920, height: 900 } });

await page.goto(`${webBase}/login`, { waitUntil: 'networkidle' });
await page.getByPlaceholder(/логин/i).fill(login);
await page.getByPlaceholder(/пароль/i).fill(password);
await page.getByRole('button', { name: /войти/i }).click();
await page.waitForTimeout(1200);
await page.goto(`${webBase}${path}`, { waitUntil: 'networkidle' });
await page.waitForTimeout(1500);
await page.screenshot({ path: out, fullPage: false });
const text = await page.locator('body').innerText();
console.log(JSON.stringify({ url: page.url(), out, text: text.slice(0, 1200) }, null, 2));
await browser.close();
