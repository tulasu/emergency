import { ruCount } from './ru-count';

describe('ruCount', () => {
  it('picks one/few/many forms', () => {
    expect(ruCount(1, 'учётка', 'учётки', 'учёток')).toBe('1 учётка');
    expect(ruCount(2, 'учётка', 'учётки', 'учёток')).toBe('2 учётки');
    expect(ruCount(5, 'учётка', 'учётки', 'учёток')).toBe('5 учёток');
    expect(ruCount(21, 'учётка', 'учётки', 'учёток')).toBe('21 учётка');
    expect(ruCount(12, 'учётка', 'учётки', 'учёток')).toBe('12 учёток');
  });
});
