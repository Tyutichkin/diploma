import { Task, formatWindowBound, windowBoundMs } from '../types/task';
import { ApiError } from '../api/client';

export function normalizeTasksOrder(tasks: Task[]): Task[] {
  return tasks.map((task, index) => ({ ...task, order: index }));
}

export function reorderTasksByIds(tasks: Task[], orderedTaskIds: string[]): Task[] {
  if (orderedTaskIds.length === 0) return tasks;

  const byId = new Map(tasks.map((task) => [task.id, task]));
  const ordered = orderedTaskIds
    .map((id) => byId.get(id))
    .filter((task): task is Task => Boolean(task));
  const missing = tasks.filter((task) => !orderedTaskIds.includes(task.id));

  return [...ordered, ...missing];
}

// Для ApiError (бэкенд) показываем заготовленный фолбэк,
// для обычной Error — её message (там русский текст фронта).
export function userErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof ApiError) return fallback;
  if (error instanceof Error) return error.message;
  return fallback;
}

export function validateStartEndConstraints(
  tasks: Task[],
  startTaskId: string,
  endTaskId: string,
): string[] {
  const issues: string[] = [];
  const startTask = startTaskId ? tasks.find((t) => t.id === startTaskId) : null;
  const endTask = endTaskId ? tasks.find((t) => t.id === endTaskId) : null;

  if (!startTask && !endTask) return issues;

  if (startTask && endTask) {
    const sWinStart = windowBoundMs(startTask.windowStartDate, startTask.windowStartTime, 'start');
    const sWinEnd = windowBoundMs(startTask.windowEndDate, startTask.windowEndTime, 'end');
    const eWinEnd = windowBoundMs(endTask.windowEndDate, endTask.windowEndTime, 'end');
    const eWinStart = windowBoundMs(endTask.windowStartDate, endTask.windowStartTime, 'start');

    if (sWinStart !== null && eWinEnd !== null && sWinStart >= eWinEnd) {
      issues.push(
        `Начальная точка "${startTask.title}" начинается в ${formatWindowBound(startTask.windowStartDate, startTask.windowStartTime)}, но конечная точка "${endTask.title}" заканчивается в ${formatWindowBound(endTask.windowEndDate, endTask.windowEndTime)}. Маршрут заведомо невозможен.`,
      );
    }

    if (sWinStart !== null && eWinEnd !== null) {
      const earliestDoneMs = sWinStart + (startTask.duration ?? 0) * 60000;
      if (earliestDoneMs >= eWinEnd) {
        issues.push(
          `После выполнения начальной точки "${startTask.title}" временное окно конечной точки "${endTask.title}" уже закрыто. Маршрут заведомо невозможен.`,
        );
      }
    }

    if (sWinEnd !== null && eWinStart !== null && sWinEnd > eWinStart) {
      issues.push(
        `Окно начальной точки "${startTask.title}" перекрывается с окном конечной точки "${endTask.title}". Маршрут может быть невыполним.`,
      );
    }
  }

  if (startTask?.windowStartDate) {
    const sWinStart = windowBoundMs(startTask.windowStartDate, startTask.windowStartTime, 'start');
    if (sWinStart !== null) {
      for (const task of tasks) {
        if (task.id === startTaskId || task.id === endTaskId) continue;
        const tWinEnd = windowBoundMs(task.windowEndDate, task.windowEndTime, 'end');
        if (tWinEnd !== null && tWinEnd <= sWinStart) {
          issues.push(
            `Задача "${task.title}" должна быть завершена до ${formatWindowBound(task.windowEndDate, task.windowEndTime)}, но начальная точка "${startTask.title}" не откроется раньше ${formatWindowBound(startTask.windowStartDate, startTask.windowStartTime)}.`,
          );
        }
      }
    }
  }

  if (endTask?.windowEndDate) {
    const eWinEnd = windowBoundMs(endTask.windowEndDate, endTask.windowEndTime, 'end');
    if (eWinEnd !== null) {
      for (const task of tasks) {
        if (task.id === startTaskId || task.id === endTaskId) continue;
        const tWinStart = windowBoundMs(task.windowStartDate, task.windowStartTime, 'start');
        if (tWinStart !== null && tWinStart >= eWinEnd) {
          issues.push(
            `Задача "${task.title}" начинается не ранее ${formatWindowBound(task.windowStartDate, task.windowStartTime)}, но конечная точка "${endTask.title}" закрывается в ${formatWindowBound(endTask.windowEndDate, endTask.windowEndTime)}.`,
          );
        }
      }
    }
  }

  return issues;
}
