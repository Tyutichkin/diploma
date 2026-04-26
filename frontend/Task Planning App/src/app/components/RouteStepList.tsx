import { Task } from '../types/task';
import { RouteStopTiming } from '../types/route';
import { TransportMode, RouteLeg, RouteSegment, TRANSPORT_TYPE_META } from '../types/transport';
import { Card, CardContent, CardHeader, CardTitle } from './ui/card';
import { AlertTriangle, Clock, ListOrdered, MapPin, Pencil, Trash2 } from 'lucide-react';
import { Button } from './ui/button';
import { fmtDuration, fmtTime } from '../utils/formatters';
import { isStopConflict } from '../utils/routeConflicts';

interface RouteStepListProps {
  tasks: Task[];
  transportMode: TransportMode;
  routeLegs: RouteLeg[];
  stops?: RouteStopTiming[];
  onEditTask?: (taskId: string) => void;
  onDeleteTask?: (taskId: string) => void;
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

  // в пределах одного сегмента транспорт обычно одного типа
  const primaryType = transports[0]?.type ?? 'bus';
  const meta = TRANSPORT_TYPE_META[primaryType] ?? TRANSPORT_TYPE_META['bus'];
  const isSubway = primaryType === 'subway' || primaryType === 'underground';

  const stopsText = seg.stopsCount != null
    ? `${seg.stopsCount} ${seg.stopsCount === 1 ? 'остановка' : seg.stopsCount < 5 ? 'остановки' : 'остановок'}`
    : null;

  return (
    <div className="rounded-md border border-blue-100 bg-blue-50 px-2.5 py-2 space-y-1.5">
      <div className="flex flex-wrap items-center gap-1.5">
        <span className="text-sm font-semibold text-gray-800">{isSubway ? 'Метро' : meta.label}:</span>
        {transports.map((tr, ti) => (
          <TransitBadge key={ti} name={tr.name} type={tr.type} color={tr.color} />
        ))}
      </div>

      {seg.stopFrom && (
        <div className="flex items-start gap-1.5 text-xs text-gray-600">
          <span className="mt-0.5 text-green-600 font-bold leading-none">▶</span>
          <span>
            {isSubway ? 'Войти на станции' : 'Сесть на остановке'}{' '}
            <span className="font-medium text-gray-800">«{seg.stopFrom}»</span>
          </span>
        </div>
      )}

      {seg.stopTo && (
        <div className="flex items-start gap-1.5 text-xs text-gray-600">
          <span className="mt-0.5 text-red-500 font-bold leading-none">■</span>
          <span>
            {isSubway ? 'Выйти на станции' : 'Выйти на остановке'}{' '}
            <span className="font-medium text-gray-800">«{seg.stopTo}»</span>
          </span>
        </div>
      )}

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
  if (!leg) {
    return (
      <div className="flex items-center gap-2 py-1 pl-4">
        <div className="w-px h-6 bg-gray-300 ml-3" />
      </div>
    );
  }

  if (transportMode !== 'masstransit') {
    if (!leg.duration && !leg.distance) {
      return (
        <div className="flex items-center gap-2 py-1 pl-4">
          <div className="w-px h-6 bg-gray-300 ml-3" />
        </div>
      );
    }
    return (
      <div className="flex items-center gap-2 py-1.5 pl-4 ml-[18px] border-l-2 border-dashed border-gray-200">
        <span className="text-sm leading-none">{transportMode === 'auto' ? '🚗' : '🚶'}</span>
        <span className="text-xs text-gray-500">
          {leg.duration || ''}
          {leg.distance ? ` · ${leg.distance}` : ''}
        </span>
      </div>
    );
  }

  const relevantSegments = leg.segments.filter(
    (s) => s.kind === 'transport' || (s.kind === 'pedestrian' && s.distance),
  );

  if (relevantSegments.length === 0) {
    return (
      <div className="flex items-center gap-2 py-1.5 pl-4 ml-[18px] border-l-2 border-dashed border-blue-200">
        {(leg.duration || leg.distance) && (
          <>
            <span className="text-sm leading-none">🚌</span>
            <span className="text-xs text-gray-500">
              {leg.duration || ''}
              {leg.distance ? ` · ${leg.distance}` : ''}
            </span>
          </>
        )}
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

export function RouteStepList({ tasks, transportMode, routeLegs, stops, onEditTask, onDeleteTask }: RouteStepListProps) {
  const validTasks = tasks.filter((t) => t.latitude !== undefined && t.longitude !== undefined);

  if (validTasks.length === 0) return null;

  const stopMap = new Map<string, RouteStopTiming>();
  if (stops) {
    for (const s of stops) stopMap.set(s.taskId, s);
  }

  const conflicts = validTasks.filter((t) => isStopConflict(stopMap.get(t.id), t));
  const hasTimingData = stops != null && stops.length > 0;

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
        {conflicts.length > 0 && (
          <div className="flex items-center gap-2 mt-2 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-800">
            <AlertTriangle className="h-4 w-4 flex-shrink-0" />
            <span>
              {conflicts.length === 1
                ? '1 задача не укладывается в временное окно'
                : `${conflicts.length} задач не укладываются в временные окна`}
              {' '}— отредактируйте окно/длительность или удалите задачу
            </span>
          </div>
        )}
      </CardHeader>
      <CardContent className="pt-0">
        <ol className="space-y-0">
          {validTasks.map((task, index) => {
            const stop = stopMap.get(task.id);
            const conflict = isStopConflict(stop, task);

            return (
              <li key={task.id}>
                <div className={`flex items-start gap-3 rounded-md px-2 py-1.5 ${conflict ? 'bg-red-50 border border-red-200' : ''}`}>
                  <div className={`flex-shrink-0 w-7 h-7 rounded-full text-white text-xs font-bold flex items-center justify-center mt-0.5 ${conflict ? 'bg-red-500' : 'bg-blue-600'}`}>
                    {index + 1}
                  </div>

                  <div className="flex-1 min-w-0 pb-1">
                    <div className="font-medium text-gray-900 text-sm leading-snug">{task.title}</div>
                    <div className="flex items-center gap-1 text-xs text-gray-500 mt-0.5">
                      <MapPin className="h-3 w-3 flex-shrink-0" />
                      <span className="truncate">{task.address}</span>
                    </div>

                    {hasTimingData && stop && (
                      <div className="flex flex-wrap gap-x-4 gap-y-0.5 mt-1.5 text-xs">
                        {index > 0 && stop.travelFromPrevSec != null && stop.travelFromPrevSec > 0 && (
                          <span className="text-blue-600">
                            <Clock className="inline h-3 w-3 mr-0.5 -mt-0.5" />
                            В пути: {fmtDuration(stop.travelFromPrevSec)}
                          </span>
                        )}
                        <span className="text-gray-600">
                          Прибытие: <strong>{fmtTime(stop.arriveTime)}</strong>
                        </span>
                        {stop.waitSec != null && stop.waitSec > 0 && (
                          <span className="text-amber-600">
                            Ожидание: {fmtDuration(stop.waitSec)}
                          </span>
                        )}
                        <span className="text-gray-600">
                          Обслуживание: {fmtTime(stop.serviceStartTime)}–{fmtTime(stop.serviceEndTime)}
                        </span>
                      </div>
                    )}

                    <div className="flex flex-wrap gap-3 mt-1 text-xs text-gray-500">
                      <span>Длительность: {task.duration} мин</span>
                      {(task.windowStartDate || task.windowStartTime || task.windowEndDate || task.windowEndTime) && (
                        <span>Окно: {[task.windowStartDate, task.windowStartTime].filter(Boolean).join(' ') || '—'}–{[task.windowEndDate, task.windowEndTime].filter(Boolean).join(' ') || '—'}</span>
                      )}
                    </div>

                    {conflict && (
                      <div className="flex items-center gap-2 mt-2">
                        <span className="text-xs font-medium text-red-600 flex items-center gap-1">
                          <AlertTriangle className="h-3.5 w-3.5" />
                          Не укладывается в окно
                        </span>
                        {onEditTask && (
                          <Button variant="outline" size="sm" className="h-6 text-xs px-2" onClick={() => onEditTask(task.id)}>
                            <Pencil className="h-3 w-3 mr-1" />
                            Редактировать
                          </Button>
                        )}
                        {onDeleteTask && (
                          <Button variant="outline" size="sm" className="h-6 text-xs px-2 text-red-600 border-red-200 hover:bg-red-50" onClick={() => onDeleteTask(task.id)}>
                            <Trash2 className="h-3 w-3 mr-1" />
                            Удалить
                          </Button>
                        )}
                      </div>
                    )}
                  </div>
                </div>

                {/* Соединитель с транспортом до следующей точки */}
                {index < validTasks.length - 1 && (
                  <LegConnector leg={routeLegs[index]} transportMode={transportMode} />
                )}
              </li>
            );
          })}
        </ol>
      </CardContent>
    </Card>
  );
}
