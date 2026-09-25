import { exportAccessesCsv, parseUsersFile } from './users-file';

describe('parseUsersFile', () => {
  it('parses CSV with Russian headers', () => {
    const result = parseUsersFile('ФИО,Логин\nИванова Анна,ivanova.ap\n', 'users.csv');
    expect(result).toEqual({
      ok: true,
      kind: 'csv',
      users: [{ full_name: 'Иванова Анна', login: 'ivanova.ap' }],
    });
  });

  it('parses CSV with English headers', () => {
    const result = parseUsersFile('full_name,login\nSokolov Ivan,sokolov.ip\n', 'people.csv');
    expect(result).toEqual({
      ok: true,
      kind: 'csv',
      users: [{ full_name: 'Sokolov Ivan', login: 'sokolov.ip' }],
    });
  });

  it('keeps quoted commas in names and skips empty rows', () => {
    const csv = 'ФИО,Логин\n"Соколов, Иван",sokolov.ip\n,\n  ,  \nПетрова Елена,petrova.em\n';
    const result = parseUsersFile(csv, 'users.csv');
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.users).toEqual([
        { full_name: 'Соколов, Иван', login: 'sokolov.ip' },
        { full_name: 'Петрова Елена', login: 'petrova.em' },
      ]);
    }
  });

  it('ignores a password column', () => {
    const result = parseUsersFile(
      'ФИО,Логин,Пароль\nИванова Анна,ivanova.ap,secret\n',
      'accesses.csv',
    );
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.users).toEqual([{ full_name: 'Иванова Анна', login: 'ivanova.ap' }]);
    }
  });

  it('rejects more than 200 rows', () => {
    const rows = Array.from({ length: 201 }, (_, i) => `User ${i},user${i}`);
    const result = parseUsersFile(['ФИО,Логин', ...rows].join('\n'), 'big.csv');
    expect(result).toEqual({
      ok: false,
      error: 'В одном файле не больше 200 пользователей',
    });
  });

  it('rejects CSV without required columns', () => {
    const result = parseUsersFile('name,email\nAnn,ann@x\n', 'users.csv');
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.error).toContain('ФИО и Логин');
    }
  });

  it('parses a JSON array', () => {
    const result = parseUsersFile(
      JSON.stringify([{ full_name: 'Иванова Анна', login: 'ivanova.ap' }]),
      'users.json',
    );
    expect(result).toEqual({
      ok: true,
      kind: 'json',
      users: [{ full_name: 'Иванова Анна', login: 'ivanova.ap' }],
    });
  });

  it('parses a JSON object with users', () => {
    const result = parseUsersFile(
      JSON.stringify({ users: [{ ФИО: 'Петрова Елена', Логин: 'petrova.em' }] }),
      'users.json',
    );
    expect(result).toEqual({
      ok: true,
      kind: 'json',
      users: [{ full_name: 'Петрова Елена', login: 'petrova.em' }],
    });
  });

  it('rejects invalid JSON', () => {
    const result = parseUsersFile('{', 'users.json');
    expect(result).toEqual({ ok: false, error: 'Не удалось прочитать JSON' });
  });

  it('rejects a JSON object without users', () => {
    const result = parseUsersFile(JSON.stringify({ items: [] }), 'users.json');
    expect(result.ok).toBe(false);
  });

  it('rejects an empty file', () => {
    const result = parseUsersFile('ФИО,Логин\n,\n', 'users.csv');
    expect(result).toEqual({ ok: false, error: 'В файле нет записей' });
  });
});

describe('exportAccessesCsv', () => {
  it('writes a UTF-8 BOM, Russian headers, and escapes quotes', () => {
    const csv = exportAccessesCsv([
      { full_name: 'Соколов, "Иван"', login: 'sokolov.ip', password: 'p,1' },
    ]);
    expect(csv.startsWith('\uFEFF')).toBe(true);
    expect(csv).toContain('ФИО,Логин,Пароль');
    expect(csv).toContain('"Соколов, ""Иван"""');
    expect(csv).toContain('sokolov.ip');
    expect(csv).toContain('"p,1"');
    expect(parseUsersFile(csv, 'accesses.csv')).toEqual({
      ok: true,
      kind: 'csv',
      users: [{ full_name: 'Соколов, "Иван"', login: 'sokolov.ip' }],
    });
  });
});
