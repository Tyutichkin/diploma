export interface Task {
  id: string;
  title: string;
  address?: string; // пустая строка или отсутствие = нет адреса
  latitude?: number;
  longitude?: number;
  duration?: number; // minutes; undefined = мгновенная задача

  // Раздельные дата и время для временных окон.
  windowStartDate?: string; // "YYYY-MM-DD"
  windowStartTime?: string; // "HH:mm"
  windowEndDate?: string;   // "YYYY-MM-DD"
  windowEndTime?: string;   // "HH:mm"

  priority?: number; // 1-5
  completed?: boolean;
  order?: number;
}

export function taskHasAddress(task: Task): boolean {
  return Boolean(task.address) && task.latitude !== undefined && task.longitude !== undefined;
}

export function formatWindowBound(date?: string, time?: string): string | undefined {
  if (!date && !time) return undefined;
  if (date && time) return `${date} ${time}`;
  if (date) return date;
  return time;
}

export function windowBoundMs(date?: string, time?: string): number | null {
  if (!date) return null;
  const str = time ? `${date}T${time}` : date;
  const ms = new Date(str).getTime();
  return isNaN(ms) ? null : ms;
}

export function isTaskWindowConflict(task: Task): boolean {
  const startMs = windowBoundMs(task.windowStartDate, task.windowStartTime);
  const endMs = windowBoundMs(task.windowEndDate, task.windowEndTime);
  if (startMs == null || endMs == null) return false;

  // Некорректное окно: конец <= начало
  if (endMs <= startMs) return true;

  // Длительность задачи не укладывается в окно
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
    // Индивидуальный конфликт (некорректное окно / длительность > окно)
    if (isTaskWindowConflict(task)) {
      ids.add(task.id);
    }

    const startMs = windowBoundMs(task.windowStartDate, task.windowStartTime);
    const endMs = windowBoundMs(task.windowEndDate, task.windowEndTime);
    if (startMs != null && endMs != null && endMs > startMs) {
      windowed.push({ task, startMs, endMs, durationMs: (task.duration ?? 0) * 60 * 1000 });
    }
  }

  // Для каждой задачи A считаем, сколько времени другие задачи вынуждены
  // занять внутри окна A. Задача B вынуждена использовать время в окне A,
  // только если её окна за пределами A не хватает для полного выполнения.
  for (const a of windowed) {
    if (ids.has(a.task.id)) continue;

    const windowMs = a.endMs - a.startMs;
    let totalDemandMs = 0;

    for (const b of windowed) {
      const intStart = Math.max(a.startMs, b.startMs);
      const intEnd = Math.min(a.endMs, b.endMs);
      const intersectionMs = intEnd - intStart;
      if (intersectionMs <= 0) continue;

      // Часть окна B, лежащая за пределами окна A
      const bOutsideA = (b.endMs - b.startMs) - intersectionMs;
      // Минимальное время, которое B обязана провести внутри окна A
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
  totalDistance: number; // km
  totalDuration: number; // minutes
  totalTravelTime: number; // minutes
}
