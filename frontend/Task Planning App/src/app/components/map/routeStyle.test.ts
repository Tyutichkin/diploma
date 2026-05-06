import { describe, expect, it } from 'vitest';
import { getRouteStyleOptions } from './routeStyle';

describe('getRouteStyleOptions', () => {
  it('pedestrian — пунктирная зелёная', () => {
    const o = getRouteStyleOptions('pedestrian');
    expect(o.routeActiveStrokeColor).toBe('#16a34a');
    expect(o.routeActiveStrokeWidth).toBe(4);
    expect(o.routeActiveStrokeStyle).toBe('dash');
  });

  it('auto — сплошная синяя', () => {
    const o = getRouteStyleOptions('auto');
    expect(o.routeActiveStrokeColor).toBe('#3b82f6');
    expect(o.routeActiveStrokeWidth).toBe(4);
    expect(o.routeActiveStrokeStyle).toBe('solid');
  });

  it('masstransit — без переопределений (дефолты ymaps)', () => {
    const o = getRouteStyleOptions('masstransit');
    expect(o.routeActiveStrokeColor).toBeUndefined();
    expect(o.routeActiveStrokeStyle).toBeUndefined();
  });

  it('boundsAutoApply и wayPointVisible выставлены везде', () => {
    (['pedestrian', 'auto', 'masstransit'] as const).forEach((mode) => {
      const o = getRouteStyleOptions(mode);
      expect(o.boundsAutoApply).toBe(true);
      expect(o.wayPointVisible).toBe(false);
    });
  });
});
