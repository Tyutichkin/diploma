import { useEffect, useRef, useState } from 'react';
import { Task } from '../types/task';
import { loadYandexMaps } from '../utils/yandexMaps';
import { Card, CardContent, CardHeader, CardTitle } from './ui/card';
import { MapPin, Loader2 } from 'lucide-react';
import {
  RouteLeg,
  RouteSegment,
  TransportInfo,
  TransportMode,
  TRANSPORT_TYPE_META,
} from '../types/transport';
import { groupTasksByCoord, computeStackPlacements } from './map/stackLayout';
import { getRouteStyleOptions } from './map/routeStyle';
import { createTaskMarker, createStackAnchor } from './map/TaskMarker';
import { createSegmentLabel } from './map/SegmentLabel';
import { RouteSegmentsPanel } from './map/RouteSegmentsPanel';

interface LegBounds {
  from: [number, number];
  to: [number, number];
}

// Ре-экспорт для обратной совместимости с уже существующими импортами.
export type { RouteLeg, RouteSegment, TransportInfo, TransportMode };
export { TRANSPORT_TYPE_META };

const TRANSPORT_MODES: { value: TransportMode; label: string }[] = [
  { value: 'pedestrian', label: 'Пешком' },
  { value: 'masstransit', label: 'Транспорт' },
  { value: 'auto', label: 'Авто' },
];

interface MapViewProps {
  tasks: Task[];
  routeOptimized?: boolean;
  onTransportModeChange?: (mode: TransportMode) => void;
  onRouteLegsChange?: (legs: RouteLeg[]) => void;
}

export function MapView({ tasks, routeOptimized = false, onTransportModeChange, onRouteLegsChange }: MapViewProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const mapRef = useRef<any>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [transportMode, setTransportMode] = useState<TransportMode>('auto');
  const [routeLegs, setRouteLegs] = useState<RouteLeg[]>([]);
  const [legBounds, setLegBounds] = useState<LegBounds[]>([]);

  // дедупликация и дебаунс — лимиты Yandex API
  const lastRouteKeyRef = useRef('');
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout>>();

  const validTasks = tasks.filter(
    (t) => t.latitude !== undefined && t.longitude !== undefined,
  );

  useEffect(() => {
    let destroyed = false;

    loadYandexMaps()
      .then(() => {
        if (destroyed || !containerRef.current) return;

        mapRef.current = new window.ymaps.Map(
          containerRef.current,
          {
            center: [55.7558, 37.6173],
            zoom: 11,
            controls: ['zoomControl', 'fullscreenControl'],
          },
          { suppressMapOpenBlock: true },
        );

        setIsLoading(false);
      })
      .catch((err: Error) => {
        if (!destroyed) setLoadError(err.message);
      });

    return () => {
      destroyed = true;
      if (mapRef.current) {
        mapRef.current.destroy();
        mapRef.current = null;
      }
    };
  }, []);

  useEffect(() => {
    if (!mapRef.current || isLoading) return;

    const newKey = [
      routeOptimized,
      transportMode,
      validTasks.map((t) => `${t.id}|${t.latitude}|${t.longitude}`).join(','),
    ].join('__');

    if (newKey === lastRouteKeyRef.current) return;

    // дебаунс 400 мс — батчим быстрые изменения
    clearTimeout(debounceTimerRef.current);
    debounceTimerRef.current = setTimeout(() => {
      if (!mapRef.current) return;
      lastRouteKeyRef.current = newKey;

    const map = mapRef.current;
    map.geoObjects.removeAll();
    setRouteLegs([]);
    onRouteLegsChange?.([]);
    setLegBounds([]);

    if (validTasks.length === 0) return;

    const groups = groupTasksByCoord(validTasks);
    groups.forEach((group) => {
      const info = computeStackPlacements(group);
      if (info.anchorLineHeightPx !== null) {
        map.geoObjects.add(
          createStackAnchor({
            lat: group.lat,
            lon: group.lon,
            lineHeightPx: info.anchorLineHeightPx,
          }),
        );
      }
      info.placements.forEach((p) => {
        const entry = group.entries.find((e) => e.index === p.taskIndex)!;
        map.geoObjects.add(
          createTaskMarker({
            task: entry.task,
            number: entry.index + 1,
            iconOffset: p.iconOffset,
            isStack: info.anchorLineHeightPx !== null,
          }),
        );
      });
    });

    if (routeOptimized && validTasks.length > 1) {
      const referencePoints = validTasks.map((t) => [t.latitude!, t.longitude!]);

      const routeOptions = getRouteStyleOptions(transportMode);

      const multiRoute = new window.ymaps.multiRouter.MultiRoute(
        { referencePoints, params: { routingMode: transportMode, optimizeWaypoints: false } },
        routeOptions,
      );

      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      multiRoute.model.events.add('requestsuccess', () => {
        try {
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          const routes: any[] = multiRoute.model.getRoutes();
          if (!routes.length) return;

          const legs: RouteLeg[] = [];
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          routes[0].getLegs().forEach((leg: any, legIndex: number) => {
            const fromTitle = validTasks[legIndex]?.title ?? `Точка ${legIndex + 1}`;
            const toTitle = validTasks[legIndex + 1]?.title ?? `Точка ${legIndex + 2}`;

            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const legProps = leg.properties as any;
            const legDurationObj = legProps.get('duration');
            const legDistanceObj = legProps.get('distance');
            const legDuration: string | undefined =
              typeof legDurationObj === 'object' ? legDurationObj?.text : legDurationObj;
            const legDistance: string | undefined =
              typeof legDistanceObj === 'object' ? legDistanceObj?.text : legDistanceObj;

            const segments: RouteSegment[] = [];

            if (transportMode === 'masstransit') {
              // детальные сегменты: метро, автобус, пешком
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              leg.getSegments().forEach((seg: any) => {
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                const props = seg.properties as any;
                const kind: 'pedestrian' | 'transport' =
                  props.get('type') === 'transport' ? 'transport' : 'pedestrian';
                const durationObj = props.get('duration');
                const distanceObj = props.get('distance');
                const duration: string | undefined =
                  typeof durationObj === 'object' ? durationObj?.text : durationObj;
                const distance: string | undefined =
                  typeof distanceObj === 'object' ? distanceObj?.text : distanceObj;

                if (kind === 'transport') {
                  // eslint-disable-next-line @typescript-eslint/no-explicit-any
                  const rawTransports: any[] = props.get('transports') ?? [];
                  const stopFrom: string | undefined =
                    props.get('departureStop')?.name ??
                    props.get('departureStop')?.properties?.get?.('name');
                  const stopTo: string | undefined =
                    props.get('arrivalStop')?.name ??
                    props.get('arrivalStop')?.properties?.get?.('name');

                  const transports: TransportInfo[] = rawTransports.map((t) => ({
                    name: t.name ?? t.number ?? '',
                    type: t.type ?? t.transportType ?? 'bus',
                    color: t.style?.color ?? t.color,
                  }));

                  const stopsCount: number | undefined = props.get('stopsCount') ?? undefined;

                  segments.push({ kind, duration, distance, transports, stopFrom, stopTo, stopsCount });
                } else {
                  segments.push({ kind, duration, distance });
                }
              });
            }

            const fromTask = validTasks[legIndex];
            const toTask = validTasks[legIndex + 1];
            if (fromTask && toTask) {
              const label = createSegmentLabel({
                fromTask, toTask, legIndex,
                duration: legDuration, distance: legDistance,
                mode: transportMode, segments,
                allTasks: validTasks,
              });
              if (label) map.geoObjects.add(label);
            }

            legs.push({ fromTitle, toTitle, duration: legDuration, distance: legDistance, segments });
          });

          setRouteLegs(legs);
          onRouteLegsChange?.(legs);

          const bounds: LegBounds[] = [];
          for (let i = 0; i < validTasks.length - 1; i++) {
            const a = validTasks[i];
            const b = validTasks[i + 1];
            bounds.push({ from: [a.latitude!, a.longitude!], to: [b.latitude!, b.longitude!] });
          }
          setLegBounds(bounds);
        } catch {
          // не распарсили — панель просто не показываем
        }
      });

      map.geoObjects.add(multiRoute);
    } else {
      const bounds = map.geoObjects.getBounds();
      if (bounds) {
        map.setBounds(bounds, { checkZoomRange: true, zoomMargin: 60 });
      }
    }
    }, 400);

    return () => clearTimeout(debounceTimerRef.current);
  }, [validTasks, routeOptimized, transportMode, isLoading]);

  return (
    <Card className="h-full flex flex-col">
      <CardHeader className="flex-shrink-0">
        <div className="flex items-center justify-between flex-wrap gap-2">
          <CardTitle className="flex items-center gap-2">
            <MapPin className="h-5 w-5" />
            Карта маршрута
            {routeOptimized && (
              <span className="text-sm font-normal text-green-600">
                (оптимизирован)
              </span>
            )}
          </CardTitle>

          {routeOptimized && (
            <div className="flex items-center gap-1 bg-gray-100 rounded-lg p-1">
              {TRANSPORT_MODES.map((mode) => (
                <button
                  key={mode.value}
                  onClick={() => { setTransportMode(mode.value); onTransportModeChange?.(mode.value); }}
                  className={`px-3 py-1 text-sm rounded-md transition-colors ${
                    transportMode === mode.value
                      ? 'bg-white shadow text-blue-600 font-medium'
                      : 'text-gray-600 hover:text-gray-900'
                  }`}
                >
                  {mode.label}
                </button>
              ))}
            </div>
          )}
        </div>
      </CardHeader>

      <CardContent className="flex-1 p-4 overflow-hidden flex flex-col gap-3">
        {loadError ? (
          <div className="h-full flex items-center justify-center bg-gray-100 rounded-lg">
            <div className="text-center text-red-500 px-4">
              <MapPin className="h-10 w-10 mx-auto mb-2 text-red-400" />
              <p className="font-medium">Ошибка загрузки карты</p>
              <p className="text-sm mt-1">{loadError}</p>
            </div>
          </div>
        ) : (
          <>
            <div className="relative flex-1 rounded-lg overflow-hidden min-h-0">
              <div ref={containerRef} className="h-full w-full" />

              {isLoading && (
                <div className="absolute inset-0 flex items-center justify-center bg-gray-100 rounded-lg">
                  <div className="text-center text-gray-500">
                    <Loader2 className="h-8 w-8 mx-auto mb-2 animate-spin text-blue-500" />
                    <p>Загрузка карты…</p>
                  </div>
                </div>
              )}

              {!isLoading && validTasks.length === 0 && (
                <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
                  <div className="text-center text-gray-500 bg-white/80 rounded-xl p-4 shadow">
                    <MapPin className="h-10 w-10 mx-auto mb-2 text-gray-400" />
                    <p>Добавьте задачи с адресами для отображения на карте</p>
                  </div>
                </div>
              )}


            </div>

            {routeOptimized && routeLegs.length > 0 && (
              <RouteSegmentsPanel
                legs={routeLegs}
                mode={transportMode}
                onLegFocus={(li) => {
                  if (!mapRef.current) return;
                  const lb = legBounds[li];
                  if (!lb) return;
                  const minLat = Math.min(lb.from[0], lb.to[0]);
                  const maxLat = Math.max(lb.from[0], lb.to[0]);
                  const minLon = Math.min(lb.from[1], lb.to[1]);
                  const maxLon = Math.max(lb.from[1], lb.to[1]);
                  mapRef.current.setBounds(
                    [[minLat, minLon], [maxLat, maxLon]],
                    { checkZoomRange: true, zoomMargin: 60 },
                  );
                }}
              />
            )}
          </>
        )}
      </CardContent>
    </Card>
  );
}
