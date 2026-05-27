import { Task } from "../types/task";
import { RouteStopTiming } from "../types/route";

export function isStopConflict(
    stop: RouteStopTiming | undefined,
    task: Task,
): boolean {
    if (!stop?.arriveTime) return false;
    if (!task.windowEndDate && !task.windowEndTime) return false;

    const arrivalMs = new Date(stop.arriveTime).getTime();
    const deadlineMs = new Date(
        task.windowEndDate && task.windowEndTime
            ? `${task.windowEndDate}T${task.windowEndTime}:00Z`
            : task.windowEndDate
              ? `${task.windowEndDate}T23:59:59Z`
              : stop.arriveTime,
    ).getTime();

    if (isNaN(deadlineMs)) return false;

    const serviceDurationMs = (task.duration ?? 0) * 60 * 1000;
    return arrivalMs > deadlineMs - serviceDurationMs;
}

export function getRouteConflictIds(
    tasks: Task[],
    stops: RouteStopTiming[],
): Set<string> {
    if (stops.length === 0) return new Set();
    const stopMap = new Map(stops.map((s) => [s.taskId, s]));
    const ids = new Set<string>();
    for (const task of tasks) {
        if (isStopConflict(stopMap.get(task.id), task)) ids.add(task.id);
    }
    return ids;
}
