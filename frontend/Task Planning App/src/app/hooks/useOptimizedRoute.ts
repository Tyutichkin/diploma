import { useCallback, useState } from 'react';
import { OptimizedRoute } from '../types/task';
import { RouteStopTiming } from '../types/route';
import { RouteLeg } from '../types/transport';

export interface OptimizedRouteState {
  routeOptimized: boolean;
  optimizedInfo: OptimizedRoute | null;
  routeStops: RouteStopTiming[];
  activeRouteId: string | null;
  routeLegs: RouteLeg[];
}

const EMPTY_STATE: OptimizedRouteState = {
  routeOptimized: false,
  optimizedInfo: null,
  routeStops: [],
  activeRouteId: null,
  routeLegs: [],
};

export function useOptimizedRoute() {
  const [state, setState] = useState<OptimizedRouteState>(EMPTY_STATE);

  const apply = useCallback((next: Partial<OptimizedRouteState> & { routeOptimized: true }) => {
    setState((prev) => ({ ...prev, ...next }));
  }, []);

  const clear = useCallback(() => {
    setState((prev) => (prev.routeOptimized || prev.activeRouteId ? EMPTY_STATE : prev));
  }, []);

  const setActiveRouteId = useCallback((id: string | null) => {
    setState((prev) => ({ ...prev, activeRouteId: id }));
  }, []);

  const setRouteLegs = useCallback((legs: RouteLeg[]) => {
    setState((prev) => ({ ...prev, routeLegs: legs }));
  }, []);

  return { ...state, apply, clear, setActiveRouteId, setRouteLegs };
}
