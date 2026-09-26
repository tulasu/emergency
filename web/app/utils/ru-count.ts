export function ruCount(n: number, one: string, few: string, many: string): string {
  const n100 = Math.abs(n) % 100;
  const n10 = n100 % 10;
  if (n100 > 10 && n100 < 20) {
    return `${n} ${many}`;
  }
  if (n10 === 1) {
    return `${n} ${one}`;
  }
  if (n10 >= 2 && n10 <= 4) {
    return `${n} ${few}`;
  }
  return `${n} ${many}`;
}
