import { execFileSync } from 'node:child_process';

function curl(args) {
  return execFileSync('curl.exe', args, { encoding: 'utf8' });
}

const login = JSON.parse(
  curl([
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
const t = login.token;
const vars = JSON.parse(
  curl([
    '-s',
    '-H',
    'Authorization: Bearer ' + t,
    'http://127.0.0.1:8080/lessons/880b1705-63a6-47a4-8036-f57fec674b5a/variants',
  ]),
);
console.log('list', vars.length, vars[0] && vars[0].id, vars[0] && vars[0].title);
const id = vars[0].id;
const out = curl(['-s', '-i', '-H', 'Authorization: Bearer ' + t, 'http://127.0.0.1:8080/variants/' + id]);
console.log(out.split('\n').slice(0, 12).join('\n'));
