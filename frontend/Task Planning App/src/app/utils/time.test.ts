import { describe, expect, it } from 'vitest';
import { roundUpSecToMinute, secToMinCeil } from './time';

describe('roundUpSecToMinute', () => {
  it('возвращает 0 для неположительных и нечисловых значений', () => {
    expect(roundUpSecToMinute(0)).toBe(0);
    expect(roundUpSecToMinute(-30)).toBe(0);
    expect(roundUpSecToMinute(NaN)).toBe(0);
  });

  it('округляет вверх до ближайшей кратной 60', () => {
    expect(roundUpSecToMinute(1)).toBe(60);
    expect(roundUpSecToMinute(59)).toBe(60);
    expect(roundUpSecToMinute(60)).toBe(60);
    expect(roundUpSecToMinute(61)).toBe(120);
    expect(roundUpSecToMinute(1930)).toBe(1980); // 32:10 → 33:00
  });
});

describe('secToMinCeil', () => {
  it('возвращает 0 для неположительных значений', () => {
    expect(secToMinCeil(0)).toBe(0);
    expect(secToMinCeil(-1)).toBe(0);
  });

  it('округляет до целых минут вверх', () => {
    expect(secToMinCeil(1)).toBe(1);
    expect(secToMinCeil(60)).toBe(1);
    expect(secToMinCeil(61)).toBe(2);
    expect(secToMinCeil(1930)).toBe(33);
  });
});
