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

export interface OptimizedRoute {
  tasks: Task[];
  totalDistance: number; // km
  totalDuration: number; // minutes
  totalTravelTime: number; // minutes
}
