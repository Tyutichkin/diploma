import { TransportMode } from '../../types/transport';

export interface RouteStyleOptions {
  boundsAutoApply: true;
  wayPointVisible: false;
  routeActiveStrokeWidth?: number;
  routeActiveStrokeColor?: string;
  routeActiveStrokeStyle?: 'solid' | 'dash';
  routeStrokeWidth?: number;
  routeStrokeColor?: string;
}

export function getRouteStyleOptions(mode: TransportMode): RouteStyleOptions {
  if (mode === 'pedestrian') {
    return {
      boundsAutoApply: true,
      wayPointVisible: false,
      routeActiveStrokeWidth: 4,
      routeActiveStrokeColor: '#16a34a',
      routeActiveStrokeStyle: 'dash',
      routeStrokeWidth: 2,
      routeStrokeColor: '#86efac',
    };
  }
  if (mode === 'auto') {
    return {
      boundsAutoApply: true,
      wayPointVisible: false,
      routeActiveStrokeWidth: 4,
      routeActiveStrokeColor: '#3b82f6',
      routeActiveStrokeStyle: 'solid',
      routeStrokeWidth: 2,
      routeStrokeColor: '#93c5fd',
    };
  }
  // masstransit — оставляем дефолты ymaps
  return { boundsAutoApply: true, wayPointVisible: false };
}
