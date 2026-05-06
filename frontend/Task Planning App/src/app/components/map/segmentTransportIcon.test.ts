import { describe, expect, it } from 'vitest';
import { RouteSegment } from '../../types/transport';
import { getSegmentIcon } from './segmentTransportIcon';

const segPedestrian = (duration = '5 мин'): RouteSegment => ({ kind: 'pedestrian', duration });
const segTransport = (
  type: string,
  duration = '5 мин',
): RouteSegment => ({
  kind: 'transport',
  duration,
  transports: [{ name: '1', type }],
});

describe('getSegmentIcon', () => {
  it('pedestrian-режим всегда отдаёт иконку пешехода', () => {
    expect(getSegmentIcon('pedestrian', []).icon).toBe('🚶');
    expect(getSegmentIcon('pedestrian', [segTransport('subway')]).icon).toBe('🚶');
  });

  it('auto-режим всегда отдаёт иконку машины', () => {
    expect(getSegmentIcon('auto', []).icon).toBe('🚗');
  });

  it('masstransit без сегментов транспорта — пешеход', () => {
    expect(getSegmentIcon('masstransit', [segPedestrian()]).icon).toBe('🚶');
  });

  it('masstransit: subway имеет наивысший приоритет над автобусом', () => {
    const r = getSegmentIcon('masstransit', [
      segTransport('bus', '20 мин'),
      segTransport('subway', '5 мин'),
    ]);
    expect(r.icon).toBe('М');
    expect(r.type).toBe('subway');
  });

  it('masstransit: при отсутствии метро доминирует наземный транспорт по типу', () => {
    const r = getSegmentIcon('masstransit', [
      segTransport('bus', '5 мин'),
      segTransport('tram', '10 мин'),
    ]);
    expect(r.type).toBe('tram');
    expect(r.icon).toBe('🚋');
  });

  it('masstransit: rail приоритетнее tram', () => {
    const r = getSegmentIcon('masstransit', [
      segTransport('tram', '10 мин'),
      segTransport('rail', '5 мин'),
    ]);
    expect(r.type).toBe('rail');
  });
});
