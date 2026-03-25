import { Task } from '../types/task';
import { TransportMode, RouteLeg, RouteSegment, TRANSPORT_TYPE_META } from './MapView';
import { Card, CardContent, CardHeader, CardTitle } from './ui/card';
import { ListOrdered, MapPin } from 'lucide-react';

interface RouteStepListProps {
  tasks: Task[];
  transportMode: TransportMode;
  routeLegs: RouteLeg[];
}

const MODE_LABEL: Record<TransportMode, string> = {
  pedestrian: 'Пешком',
  masstransit: 'Общественный транспорт',
  auto: 'Авто',
};

function TransitBadge({ name, type, color }: { name: string; type: string; color?: string }) {
  const meta = TRANSPORT_TYPE_META[type] ?? TRANSPORT_TYPE_META['bus'];
  const isSubway = type === 'subway' || type === 'underground';
  return (
    <span
      className={`inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded text-xs font-bold text-white ${!color ? meta.bg : ''}`}
      style={color ? { backgroundColor: color } : undefined}
      title={`${meta.label} ${name}`}
    >
      {isSubway ? 'М' : meta.icon} {name}
    </span>
  );
}

function TransportSegmentCard({ seg }: { seg: RouteSegment & { kind: 'transport' } }) {
  const transports = seg.transports ?? [];

  // Определяем тип первого транспорта (все в сегменте обычно одного типа)
  const primaryType = transports[0]?.type ?? 'bus';
  const meta = TRANSPORT_TYPE_META[primaryType] ?? TRANSPORT_TYPE_META['bus'];
  const isSubway = primaryType === 'subway' || primaryType === 'underground';

  const stopsText = seg.stopsCount != null
    ? `${seg.stopsCount} ${seg.stopsCount === 1 ? 'остановка' : seg.stopsCount < 5 ? 'остановки' : 'остановок'}`
    : null;

  return (
    <div className="rounded-md border border-blue-100 bg-blue-50 px-2.5 py-2 space-y-1.5">
      {/* Заголовок: тип транспорта + маршруты */}
      <div className="flex flex-wrap items-center gap-1.5">
        <span className="text-sm font-semibold text-gray-800">{isSubway ? 'Метро' : meta.label}:</span>
        {transports.map((tr, ti) => (
          <TransitBadge key={ti} name={tr.name} type={tr.type} color={tr.color} />
        ))}
      </div>

      {/* Посадка */}
      {seg.stopFrom && (
        <div className="flex items-start gap-1.5 text-xs text-gray-600">
          <span className="mt-0.5 text-green-600 font-bold leading-none">▶</span>
          <span>
            {isSubway ? 'Войти на станции' : 'Сесть на остановке'}{' '}
            <span className="font-medium text-gray-800">«{seg.stopFrom}»</span>
          </span>
        </div>
      )}

      {/* Высадка */}
      {seg.stopTo && (
        <div className="flex items-start gap-1.5 text-xs text-gray-600">
          <span className="mt-0.5 text-red-500 font-bold leading-none">■</span>
          <span>
            {isSubway ? 'Выйти на станции' : 'Выйти на остановке'}{' '}
            <span className="font-medium text-gray-800">«{seg.stopTo}»</span>
          </span>
        </div>
      )}

      {/* Время и количество остановок */}
      {(seg.duration || stopsText) && (
        <div className="flex items-center gap-2 text-xs text-gray-500">
          {seg.duration && <span>⏱ {seg.duration}</span>}
          {stopsText && <span>· {stopsText}</span>}
        </div>
      )}
    </div>
  );
}

function LegConnector({ leg, transportMode }: { leg: RouteLeg | undefined; transportMode: TransportMode }) {
  if (transportMode !== 'masstransit' || !leg) {
    return (
      <div className="flex items-center gap-2 py-1 pl-4">
        <div className="w-px h-6 bg-gray-300 ml-3" />
      </div>
    );
  }

  const relevantSegments = leg.segments.filter(
    (s) => s.kind === 'transport' || (s.kind === 'pedestrian' && s.distance),
  );

  if (relevantSegments.length === 0) {
    return (
      <div className="flex items-center gap-2 py-1 pl-4">
        <div className="w-px h-6 bg-gray-300 ml-3" />
      </div>
    );
  }

  return (
    <div className="pl-4 py-1.5 space-y-1.5 border-l-2 border-dashed border-blue-200 ml-[18px]">
      {relevantSegments.map((seg, i) => (
        <div key={i}>
          {seg.kind === 'pedestrian' ? (
            <div className="flex items-center gap-2 text-xs text-gray-500 py-0.5">
              <span className="text-sm leading-none">🚶</span>
              <span>
                Пешком{seg.distance ? ` · ${seg.distance}` : ''}
                {seg.duration ? ` · ${seg.duration}` : ''}
              </span>
            </div>
          ) : (
            <TransportSegmentCard seg={seg as RouteSegment & { kind: 'transport' }} />
          )}
        </div>
      ))}
    </div>
  );
}

export function RouteStepList({ tasks, transportMode, routeLegs }: RouteStepListProps) {
  const validTasks = tasks.filter((t) => t.latitude !== undefined && t.longitude !== undefined);

  if (validTasks.length === 0) return null;

  return (
    <Card className="mt-6">
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="flex items-center gap-2 text-base">
            <ListOrdered className="h-5 w-5" />
            Порядок объектов
          </CardTitle>
          <span className="text-sm text-gray-500">{MODE_LABEL[transportMode]}</span>
        </div>
      </CardHeader>
      <CardContent className="pt-0">
        <ol className="space-y-0">
          {validTasks.map((task, index) => (
            <li key={task.id}>
              {/* Карточка задачи */}
              <div className="flex items-start gap-3">
                {/* Номер-кружок */}
                <div className="flex-shrink-0 w-7 h-7 rounded-full bg-blue-600 text-white text-xs font-bold flex items-center justify-center mt-0.5">
                  {index + 1}
                </div>

                <div className="flex-1 min-w-0 pb-1">
                  <div className="font-medium text-gray-900 text-sm leading-snug">{task.title}</div>
                  <div className="flex items-center gap-1 text-xs text-gray-500 mt-0.5">
                    <MapPin className="h-3 w-3 flex-shrink-0" />
                    <span className="truncate">{task.address}</span>
                  </div>
                  <div className="flex flex-wrap gap-3 mt-1 text-xs text-gray-500">
                    <span>Длительность: {task.duration} мин</span>
                    {task.timeWindowStart && task.timeWindowEnd && (
                      <span>Окно: {task.timeWindowStart}–{task.timeWindowEnd}</span>
                    )}
                  </div>
                </div>
              </div>

              {/* Соединитель с транспортом до следующей точки */}
              {index < validTasks.length - 1 && (
                <LegConnector leg={routeLegs[index]} transportMode={transportMode} />
              )}
            </li>
          ))}
        </ol>
      </CardContent>
    </Card>
  );
}
