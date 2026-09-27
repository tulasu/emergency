import { writeFileSync } from 'node:fs';
import { execFileSync } from 'node:child_process';

const dir = 'd:/Repositories/emergency/web/evidence';

function curlJson(args) {
  const out = execFileSync('curl.exe', args, { encoding: 'utf8' });
  return out;
}

const login = JSON.parse(
  curlJson([
    '-s',
    '-X',
    'POST',
    'http://127.0.0.1:8080/auth/login',
    '-H',
    'Content-Type: application/json',
    '-d',
    '{"login":"teacher","password":"password1"}',
  ]),
);
const token = login.token;

const modules = [
  {
    id: '9e21a005-ed8a-49b7-a8e2-35600d8c0530',
    title: 'Работа с карточкой происшествия',
    description: 'Карточка 112: адрес, тип происшествия, опросная карта',
    status: 'active',
  },
  {
    id: 'bd0fe44f-9d82-4521-9c09-3b1cecac5f39',
    title: 'Приём вызова и первичная обработка',
    description: 'Коммуникация, эмпатия, базовый скрипт',
    status: 'active',
  },
  {
    id: '50770eb7-4770-401b-b87e-ef7470845aa2',
    title: 'Модуль mujvx190',
    description: 'e2e',
    status: 'active',
  },
];

for (const m of modules) {
  const path = `${dir}/fix-${m.id.slice(0, 8)}.json`;
  writeFileSync(
    path,
    JSON.stringify({
      title: m.title,
      description: m.description,
      status: m.status,
      success_threshold: 70,
    }),
    'utf8',
  );
  const out = curlJson([
    '-s',
    '-X',
    'PATCH',
    `http://127.0.0.1:8080/modules/${m.id}`,
    '-H',
    `Authorization: Bearer ${token}`,
    '-H',
    'Content-Type: application/json; charset=utf-8',
    '--data-binary',
    `@${path}`,
  ]);
  const j = JSON.parse(out);
  console.log('module', j.title);
}

const lessons = [
  { id: '880b1705-63a6-47a4-8036-f57fec674b5a', title: 'Адрес и ориентиры', position: 0 },
  { id: '4cf61df6-a3bc-4102-861d-e0056851f284', title: 'Занятие mujvx190', position: 0 },
];

for (const l of lessons) {
  const path = `${dir}/fix-lesson-${l.id.slice(0, 8)}.json`;
  writeFileSync(path, JSON.stringify({ title: l.title, position: l.position }), 'utf8');
  const out = curlJson([
    '-s',
    '-w',
    '\n%{http_code}',
    '-X',
    'PATCH',
    `http://127.0.0.1:8080/lessons/${l.id}`,
    '-H',
    `Authorization: Bearer ${token}`,
    '-H',
    'Content-Type: application/json; charset=utf-8',
    '--data-binary',
    `@${path}`,
  ]);
  console.log('lesson', out);
}
