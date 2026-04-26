import { useEffect, useRef, useState } from 'react';
import { Task } from '../types/task';
import { loadYandexMaps } from '../utils/yandexMaps';
import { Card, CardContent, CardHeader, CardTitle } from './ui/card';
import { MapPin, Loader2, ChevronDown, ChevronUp } from 'lucide-react';
import {
  RouteLeg,
  RouteSegment,
  TransportInfo,
  TransportMode,
  TRANSPORT_TYPE_META,
} from '../types/transport';

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
  const [panelOpen, setPanelOpen] = useState(true);

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

    if (validTasks.length === 0) return;

    // группируем по координатам, чтобы маркеры не перекрывались
    const tasksByCoord = new Map<string, { task: Task; index: number }[]>();
    validTasks.forEach((task, index) => {
      const key = `${task.latitude}|${task.longitude}`;
      if (!tasksByCoord.has(key)) tasksByCoord.set(key, []);
      tasksByCoord.get(key)!.push({ task, index });
    });

    tasksByCoord.forEach((entries) => {
      const isSingle = entries.length === 1;
      const numbers = entries.map((e) => e.index + 1);
      const iconContent = numbers.length > 2 ? '…' : numbers.join(',');

      const balloonHeader = isSingle
        ? `#${numbers[0]} ${entries[0].task.title}`
        : `#${iconContent} — ${entries.length} задачи по одному адресу`;

      const balloonBody = entries
        .map(({ task, index }) =>
          [
            isSingle ? '' : `<div style="font-weight:600;margin-top:8px">#${index + 1} ${task.title}</div>`,
            `<div style="color:#555">${task.address}</div>`,
            task.duration != null ? `<div style="color:#555;margin-top:2px">Длительность: ${task.duration} мин</div>` : '',
            (task.windowStartDate || task.windowStartTime || task.windowEndDate || task.windowEndTime)
              ? `<div style="color:#555">Окно: ${[task.windowStartDate, task.windowStartTime].filter(Boolean).join(' ') || '—'} – ${[task.windowEndDate, task.windowEndTime].filter(Boolean).join(' ') || '—'}</div>`
              : '',
          ]
            .filter(Boolean)
            .join(''),
        )
        .join('<hr style="margin:6px 0;border-color:#ddd">');

      const hintContent = isSingle
        ? entries[0].task.title
        : `Задачи ${iconContent}: ${entries.map((e) => e.task.title).join(', ')}`;

      const placemark = new window.ymaps.Placemark(
        [entries[0].task.latitude!, entries[0].task.longitude!],
        { iconContent, balloonContentHeader: balloonHeader, balloonContentBody: balloonBody, hintContent },
        { preset: isSingle ? 'islands#blueCircleIcon' : 'islands#orangeCircleIcon' },
      );
      map.geoObjects.add(placemark);
    });

    if (routeOptimized && validTasks.length > 1) {
      const referencePoints = validTasks.map((t) => [t.latitude!, t.longitude!]);

      // в masstransit цвета линий задаются ymaps (метро, автобус и т.д.) — не переопределяем
      const routeOptions =
        transportMode === 'masstransit'
          ? { boundsAutoApply: true, wayPointVisible: false }
          : {
              boundsAutoApply: true,
              wayPointVisible: false,
              routeActiveStrokeWidth: 4,
              routeActiveStrokeColor: transportMode === 'auto' ? '#3b82f6' : '#16a34a',
              routeActiveStrokeStyle: 'solid',
              routeStrokeWidth: 2,
              routeStrokeColor: transportMode === 'auto' ? '#93c5fd' : '#86efac',
            };

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

            // метка на середине отрезка — показывает время в пути
            if (legIndex < validTasks.length - 1) {
              const lat1 = validTasks[legIndex].latitude!;
              const lon1 = validTasks[legIndex].longitude!;
              const lat2 = validTasks[legIndex + 1].latitude!;
              const lon2 = validTasks[legIndex + 1].longitude!;

              const midPlacemark = new window.ymaps.Placemark(
                [(lat1 + lat2) / 2, (lon1 + lon2) / 2],
                {
                  iconContent: legDuration || '',
                  balloonContentHeader: `${fromTitle} &rarr; ${toTitle}`,
                  balloonContentBody:
                    `<div style="font-size:13px">` +
                    `<div>Время в пути: <b>${legDuration || '\u2014'}</b></div>` +
                    `<div>Расстояние: <b>${legDistance || '\u2014'}</b></div>` +
                    `</div>`,
                  hintContent: `${fromTitle} \u2192 ${toTitle}: ${legDuration || '\u2014'}`,
                },
                { preset: 'islands#grayStretchyIcon' },
              );
              map.geoObjects.add(midPlacemark);
            }

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

            legs.push({ fromTitle, toTitle, duration: legDuration, distance: legDistance, segments });
          });

          setRouteLegs(legs);
          onRouteLegsChange?.(legs);
          if (transportMode === 'masstransit') setPanelOpen(true);
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

  const showTransitPanel =
    transportMode === 'masstransit' && routeLegs.length > 0;

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

            {showTransitPanel && (
              <div className="flex-shrink-0 border border-blue-200 rounded-lg bg-blue-50 overflow-hidden">
                <button
                  onClick={() => setPanelOpen((v) => !v)}
                  className="w-full flex items-center justify-between px-4 py-2 text-sm font-medium text-blue-800 hover:bg-blue-100 transition-colors"
                >
                  <span>Маршрут общественного транспорта</span>
                  {panelOpen ? (
                    <ChevronUp className="h-4 w-4" />
                  ) : (
                    <ChevronDown className="h-4 w-4" />
                  )}
                </button>

                {panelOpen && (
                  <div className="max-h-52 overflow-y-auto px-4 pb-3 space-y-4">
                    {routeLegs.map((leg, li) => (
                      <div key={li}>
                        <div className="text-xs font-semibold text-blue-700 mb-1 pt-2">
                          {leg.fromTitle} → {leg.toTitle}
                        </div>

                        <div className="space-y-1">
                          {leg.segments.map((seg, si) => (
                            <div key={si} className="flex items-start gap-2">
                              {seg.kind === 'pedestrian' ? (
                                <>
                                  <span className="text-base leading-none mt-0.5">🚶</span>
                                  <div className="text-sm text-gray-700">
                                    <span>Пешком</span>
                                    {seg.distance && (
                                      <span className="text-gray-500"> · {seg.distance}</span>
                                    )}
                                    {seg.duration && (
                                      <span className="text-gray-500"> · {seg.duration}</span>
                                    )}
                                  </div>
                                </>
                              ) : (
                                <>
                                  <div className="flex flex-wrap gap-1 mt-0.5">
                                    {(seg.transports ?? []).map((tr, ti) => {
                                      const meta =
                                        TRANSPORT_TYPE_META[tr.type] ??
                                        TRANSPORT_TYPE_META['bus'];
                                      const bgStyle = tr.color
                                        ? { backgroundColor: tr.color }
                                        : undefined;
                                      return (
                                        <span
                                          key={ti}
                                          className={`inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded text-xs font-bold ${meta.text} ${!tr.color ? meta.bg : ''}`}
                                          style={bgStyle}
                                          title={`${meta.label} ${tr.name}`}
                                        >
                                          {tr.type === 'subway' || tr.type === 'underground'
                                            ? 'М'
                                            : meta.icon}{' '}
                                          {tr.name}
                                        </span>
                                      );
                                    })}
                                  </div>
                                  <div className="text-sm text-gray-700">
                                    {seg.stopFrom && (
                                      <span>
                                        от <span className="font-medium">{seg.stopFrom}</span>
                                      </span>
                                    )}
                                    {seg.stopTo && (
                                      <span>
                                        {' '}до <span className="font-medium">{seg.stopTo}</span>
                                      </span>
                                    )}
                                    {seg.duration && (
                                      <span className="text-gray-500"> · {seg.duration}</span>
                                    )}
                                  </div>
                                </>
                              )}
                            </div>
                          ))}
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}
          </>
        )}
      </CardContent>
    </Card>
  );
}
