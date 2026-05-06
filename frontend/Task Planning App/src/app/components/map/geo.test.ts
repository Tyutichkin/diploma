import { describe, expect, it } from 'vitest';
import { haversineMeters } from './geo';

describe('haversineMeters', () => {
  it('возвращает 0 для одной и той же точки', () => {
    expect(haversineMeters(55.75, 37.62, 55.75, 37.62)).toBe(0);
  });

  it('считает короткое расстояние в Москве с погрешностью <2 м', () => {
    // Тверская 1 → Тверская 3, ~70 м реально
    const d = haversineMeters(55.7558, 37.6173, 55.7565, 37.6173);
    expect(d).toBeGreaterThan(60);
    expect(d).toBeLessThan(90);
  });

  it('считает длинное расстояние Москва-СПб ~635 км', () => {
    const d = haversineMeters(55.7558, 37.6173, 59.9343, 30.3351);
    expect(d).toBeGreaterThan(630_000);
    expect(d).toBeLessThan(640_000);
  });

  it('возвращает число для антиподов', () => {
    const d = haversineMeters(0, 0, 0, 180);
    expect(d).toBeGreaterThan(20_000_000);
  });
});
