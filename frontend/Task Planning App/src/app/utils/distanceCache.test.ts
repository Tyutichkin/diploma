import { beforeEach, describe, expect, it } from 'vitest';
import { clearDistanceCache, getEdge, setEdge } from './distanceCache';

const A = { latitude: 55.751244, longitude: 37.618423 };
const B = { latitude: 59.93428, longitude: 30.335099 };

describe('distanceCache', () => {
  beforeEach(() => clearDistanceCache());

  it('возвращает undefined для промаха', () => {
    expect(getEdge('auto', A, B)).toBeUndefined();
  });

  it('отдаёт сохранённое ребро при попадании', () => {
    setEdge('auto', A, B, { distanceM: 1000, durationSec: 120 });
    expect(getEdge('auto', A, B)).toEqual({ distanceM: 1000, durationSec: 120 });
  });

  it('направленный: A→B и B→A — разные рёбра', () => {
    setEdge('auto', A, B, { distanceM: 1000, durationSec: 120 });
    expect(getEdge('auto', B, A)).toBeUndefined();
  });

  it('режим транспорта входит в ключ', () => {
    setEdge('auto', A, B, { distanceM: 1000, durationSec: 120 });
    expect(getEdge('pedestrian', A, B)).toBeUndefined();
  });

  it('джиттер координат за пределами 5 знаков схлопывается в один ключ', () => {
    setEdge('auto', A, B, { distanceM: 1000, durationSec: 120 });
    const aJitter = { latitude: A.latitude + 0.0000004, longitude: A.longitude - 0.0000003 };
    expect(getEdge('auto', aJitter, B)).toEqual({ distanceM: 1000, durationSec: 120 });
  });

  it('разные координаты на уровне 5 знаков — разные рёбра', () => {
    setEdge('auto', A, B, { distanceM: 1000, durationSec: 120 });
    const aShifted = { latitude: A.latitude + 0.001, longitude: A.longitude };
    expect(getEdge('auto', aShifted, B)).toBeUndefined();
  });

  it('пропускает точки без координат', () => {
    const noCoords = { latitude: null, longitude: null };
    setEdge('auto', noCoords, B, { distanceM: 1000, durationSec: 120 });
    expect(getEdge('auto', noCoords, B)).toBeUndefined();
  });
});
