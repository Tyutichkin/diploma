export interface Task {
    id: string;
    title: string;
    address?: string;
    latitude?: number;
    longitude?: number;
    duration?: number; // минуты; undefined = мгновенная задача

    windowStartDate?: string; // "YYYY-MM-DD"
    windowStartTime?: string; // "HH:mm"
    windowEndDate?: string; // "YYYY-MM-DD"
    windowEndTime?: string; // "HH:mm"

    priority?: number;
    completed?: boolean;
    order?: number;
}

export function taskHasAddress(task: Task): boolean {
    return (
        Boolean(task.address) &&
        task.latitude !== undefined &&
        task.longitude !== undefined
    );
}

export function formatWindowBound(
    date?: string,
    time?: string,
): string | undefined {
    if (!date && !time) return undefined;
    if (date && time) return `${date} ${time}`;
    if (date) return date;
    return time;
}

export function windowBoundMs(
    date: string | undefined,
    time: string | undefined,
    kind: "start" | "end" = "start",
): number | null {
    if (!date) return null;
    const effectiveTime = time ? time : kind === "end" ? "23:59" : "00:00";
    const ms = new Date(`${date}T${effectiveTime}`).getTime();
    return isNaN(ms) ? null : ms;
}

export function isTaskWindowConflict(task: Task): boolean {
    const startMs = windowBoundMs(
        task.windowStartDate,
        task.windowStartTime,
        "start",
    );
    const endMs = windowBoundMs(task.windowEndDate, task.windowEndTime, "end");
    if (startMs == null || endMs == null) return false;

    if (endMs <= startMs) return true;

    // Длительность задачи не укладывается в окно.
    if (task.duration != null) {
        const durationMs = task.duration * 60 * 1000;
        if (endMs - startMs < durationMs) return true;
    }

    return false;
}

export function getWindowConflictIds(tasks: Task[]): Set<string> {
    const ids = new Set<string>();

    interface WindowedTask {
        task: Task;
        startMs: number;
        endMs: number;
        durationMs: number;
    }

    const windowed: WindowedTask[] = [];
    for (const task of tasks) {
        if (isTaskWindowConflict(task)) {
            ids.add(task.id);
        }

        const startMs = windowBoundMs(
            task.windowStartDate,
            task.windowStartTime,
            "start",
        );
        const endMs = windowBoundMs(
            task.windowEndDate,
            task.windowEndTime,
            "end",
        );
        if (startMs != null && endMs != null && endMs > startMs) {
            windowed.push({
                task,
                startMs,
                endMs,
                durationMs: (task.duration ?? 0) * 60 * 1000,
            });
        }
    }

    for (const a of windowed) {
        if (ids.has(a.task.id)) continue;

        const windowMs = a.endMs - a.startMs;
        let totalDemandMs = 0;

        for (const b of windowed) {
            const intStart = Math.max(a.startMs, b.startMs);
            const intEnd = Math.min(a.endMs, b.endMs);
            const intersectionMs = intEnd - intStart;
            if (intersectionMs <= 0) continue;

            const bOutsideA = b.endMs - b.startMs - intersectionMs;
            const demandInA = Math.max(0, b.durationMs - bOutsideA);
            totalDemandMs += demandInA;
        }

        if (totalDemandMs > windowMs) {
            ids.add(a.task.id);
        }
    }

    return ids;
}

export interface OptimizedRoute {
    tasks: Task[];
    totalDistance: number; // км
    totalDuration: number; // минуты
    totalTravelTime: number; // минуты
}
