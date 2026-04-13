import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import { Task, OptimizedRoute } from '../types/task';
import { SavedRouteSummary } from '../types/route';
import { TaskList } from './TaskList';
import { TaskForm, TaskRole } from './TaskForm';
import { MapView, TransportMode, RouteLeg } from './MapView';
import { RouteStepList } from './RouteStepList';
import { Button } from './ui/button';
import { Card, CardContent, CardHeader, CardTitle } from './ui/card';
import { Tooltip, TooltipContent, TooltipTrigger } from './ui/tooltip';
import {
  Plus,
  Route,
  Download,
  Loader2,
  Info,
  Clock,
  Navigation,
  History,
  Trash2,
  Pencil,
  Check,
  X,

} from 'lucide-react';
import { toast } from 'sonner';
import { buildYandexDistanceMatrix, geocodeAddressSuggestions } from '../utils/routeOptimizer';
import {
  ApiError,
  PrecedenceConstraint,
  createTask,
  deleteAllRoutes,
  deleteRoute,
  deleteTask,
  getCompletedTaskIdsStorageKey,
  getRoute,
  listRoutes,
  listTasks,
  optimizeRoute,
  renameRoute,
  reorderTasks,
  updateTask,
} from '../api/client';
import { useAuth } from '../context/auth-context';

interface PrecedencePair {
  beforeId: string;
  afterId: string;
}

function normalizeTasksOrder(tasks: Task[]) {
  return tasks.map((task, index) => ({
    ...task,
    order: index,
  }));
}

function mergeCompletedState(tasks: Task[], completedIds: string[]) {
  const completedSet = new Set(completedIds);
  return tasks.map((task) => ({
    ...task,
    completed: completedSet.has(task.id),
  }));
}

function readCompletedTaskIds(email: string) {
  if (typeof window === 'undefined') {
    return [];
  }

  const raw = window.localStorage.getItem(getCompletedTaskIdsStorageKey(email));
  if (!raw) {
    return [];
  }

  try {
    const parsed = JSON.parse(raw) as string[];
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

function timeToMins(t: string): number {
  const [h, m] = t.split(':').map(Number);
  return h * 60 + m;
}

function minsToTime(mins: number): string {
  const h = Math.floor(mins / 60);
  const m = mins % 60;
  return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}`;
}

function validateStartEndConstraints(
  tasks: Task[],
  startTaskId: string,
  endTaskId: string,
  startTimeMins: number,
): string[] {
  const issues: string[] = [];
  const startTask = startTaskId ? tasks.find((t) => t.id === startTaskId) : null;
  const endTask = endTaskId ? tasks.find((t) => t.id === endTaskId) : null;

  if (!startTask && !endTask) return issues;

  // Начальная точка: её окно должно быть доступно в момент старта маршрута
  if (startTask?.timeWindowEnd) {
    const winEnd = timeToMins(startTask.timeWindowEnd);
    if (winEnd < startTimeMins) {
      issues.push(
        `Временное окно начальной точки "${startTask.title}" заканчивается в ${startTask.timeWindowEnd}, раньше времени начала маршрута (${minsToTime(startTimeMins)}).`,
      );
    }
  }

  // Начальная точка vs конечная точка
  if (startTask && endTask) {
    const sWinStart = startTask.timeWindowStart ? timeToMins(startTask.timeWindowStart) : null;
    const sWinEnd = startTask.timeWindowEnd ? timeToMins(startTask.timeWindowEnd) : null;
    const eWinEnd = endTask.timeWindowEnd ? timeToMins(endTask.timeWindowEnd) : null;
    const eWinStart = endTask.timeWindowStart ? timeToMins(endTask.timeWindowStart) : null;

    // Начало старта позже конца финиша
    if (sWinStart !== null && eWinEnd !== null && sWinStart >= eWinEnd) {
      issues.push(
        `Начальная точка "${startTask.title}" начинается в ${startTask.timeWindowStart}, но конечная точка "${endTask.title}" заканчивается в ${endTask.timeWindowEnd}. Маршрут заведомо невозможен.`,
      );
    }

    // После завершения начальной задачи окно конечной уже закрыто
    if (sWinStart !== null && eWinEnd !== null) {
      const earliestDoneStart = sWinStart + startTask.duration;
      if (earliestDoneStart >= eWinEnd) {
        issues.push(
          `После выполнения начальной точки "${startTask.title}" (не ранее ${minsToTime(earliestDoneStart)}) временное окно конечной точки "${endTask.title}" уже закрыто (до ${endTask.timeWindowEnd}).`,
        );
      }
    }

    // Конец окна начальной точки позже начала окна конечной — возможный конфликт
    if (sWinEnd !== null && eWinStart !== null && sWinEnd > eWinStart) {
      issues.push(
        `Окно начальной точки "${startTask.title}" (до ${startTask.timeWindowEnd}) перекрывается с окном конечной точки "${endTask.title}" (с ${endTask.timeWindowStart}). Маршрут может быть невыполним.`,
      );
    }
  }

  // Промежуточные задачи vs начальная точка
  if (startTask?.timeWindowStart) {
    const sWinStart = timeToMins(startTask.timeWindowStart);
    for (const task of tasks) {
      if (task.id === startTaskId || task.id === endTaskId) continue;
      if (task.timeWindowEnd) {
        const tWinEnd = timeToMins(task.timeWindowEnd);
        if (tWinEnd <= sWinStart) {
          issues.push(
            `Задача "${task.title}" должна быть завершена до ${task.timeWindowEnd}, но начальная точка "${startTask.title}" не откроется раньше ${startTask.timeWindowStart}. Задача не может быть выполнена после начальной точки.`,
          );
        }
      }
    }
  }

  // Промежуточные задачи vs конечная точка
  if (endTask?.timeWindowEnd) {
    const eWinEnd = timeToMins(endTask.timeWindowEnd);
    for (const task of tasks) {
      if (task.id === startTaskId || task.id === endTaskId) continue;
      if (task.timeWindowStart) {
        const tWinStart = timeToMins(task.timeWindowStart);
        if (tWinStart >= eWinEnd) {
          issues.push(
            `Задача "${task.title}" начинается не ранее ${task.timeWindowStart}, но конечная точка "${endTask.title}" закрывается в ${endTask.timeWindowEnd}. Задача не может быть выполнена до конечной точки.`,
          );
        }
      }
    }
  }

  return issues;
}

function reorderTasksByIds(tasks: Task[], orderedTaskIds: string[]) {
  if (orderedTaskIds.length === 0) {
    return tasks;
  }

  const byId = new Map(tasks.map((task) => [task.id, task]));
  const ordered = orderedTaskIds
    .map((taskId) => byId.get(taskId))
    .filter((task): task is Task => Boolean(task));
  const missing = tasks.filter((task) => !orderedTaskIds.includes(task.id));

  return [...ordered, ...missing];
}

export function MainPage() {
  const { session, setSession, logout } = useAuth();
  const tasksRef = useRef<Task[]>([]);

  const [tasks, setTasks] = useState<Task[]>([]);
  const [savedRoutes, setSavedRoutes] = useState<SavedRouteSummary[]>([]);
  const [editingTask, setEditingTask] = useState<Task | null>(null);
  const [isFormOpen, setIsFormOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [isSavingTask, setIsSavingTask] = useState(false);
  const [isPersistingOrder, setIsPersistingOrder] = useState(false);
  const [isOptimizing, setIsOptimizing] = useState(false);
  const [routeOptimized, setRouteOptimized] = useState(false);
  const [optimizedInfo, setOptimizedInfo] = useState<OptimizedRoute | null>(null);
  const [completedTaskIds, setCompletedTaskIds] = useState<string[]>(() =>
    session ? readCompletedTaskIds(session.email) : [],
  );
  const [activeRouteId, setActiveRouteId] = useState<string | null>(null);
  const [loadingRouteId, setLoadingRouteId] = useState<string | null>(null);
  const [editingRouteId, setEditingRouteId] = useState<string | null>(null);
  const [editingRouteName, setEditingRouteName] = useState('');
  const [transportMode, setTransportMode] = useState<TransportMode>('auto');
  const [routeLegs, setRouteLegs] = useState<RouteLeg[]>([]);
  const [startTaskId, setStartTaskId] = useState<string>('');
  const [endTaskId, setEndTaskId] = useState<string>('');
  const [precedences, setPrecedences] = useState<PrecedencePair[]>([]);

  const handleUnauthorized = useCallback(async () => {
    toast.error('Сессия истекла. Выполните вход заново.');
    await logout();
  }, [logout]);

  const requestOptions = useMemo(() => {
    if (!session) {
      return null;
    }

    return {
      session,
      onSessionChange: setSession,
      onUnauthorized: handleUnauthorized,
    };
  }, [handleUnauthorized, session, setSession]);

  const replaceTasks = useCallback((nextTasks: Task[]) => {
    const normalized = normalizeTasksOrder(nextTasks);
    tasksRef.current = normalized;
    setTasks(normalized);
  }, []);

  useEffect(() => {
    if (!session) {
      return;
    }

    setCompletedTaskIds(readCompletedTaskIds(session.email));
  }, [session]);

  useEffect(() => {
    if (!session || typeof window === 'undefined' || isLoading) {
      return;
    }

    const availableIds = new Set(tasks.map((task) => task.id));
    const cleaned = completedTaskIds.filter((id) => availableIds.has(id));
    const storageKey = getCompletedTaskIdsStorageKey(session.email);

    window.localStorage.setItem(storageKey, JSON.stringify(cleaned));

    if (cleaned.length !== completedTaskIds.length) {
      setCompletedTaskIds(cleaned);
    }
  }, [completedTaskIds, session, tasks, isLoading]);

  const loadData = useCallback(async () => {
    if (!requestOptions || !session) {
      return;
    }

    setIsLoading(true);

    try {
      const [serverTasks, routes] = await Promise.all([
        listTasks(requestOptions),
        listRoutes(requestOptions),
      ]);

      const completedIds = readCompletedTaskIds(session.email);
      replaceTasks(mergeCompletedState(serverTasks, completedIds));
      setCompletedTaskIds(completedIds);
      setSavedRoutes(routes);
    } catch (error) {
      if (!(error instanceof ApiError && error.status === 401)) {
        const message = error instanceof Error ? error.message : 'Не удалось загрузить данные';
        toast.error(message);
      }
    } finally {
      setIsLoading(false);
    }
  }, [replaceTasks, requestOptions, session]);

  useEffect(() => {
    void loadData();
  }, [loadData]);

  const persistTaskOrder = useCallback(
    async (orderedTasks: Task[]) => {
      if (!requestOptions) {
        return false;
      }

      setIsPersistingOrder(true);

      try {
        await reorderTasks(
          orderedTasks.map((task, index) => ({ id: task.id, sortIndex: index })),
          requestOptions,
        );

        return true;
      } catch (error) {
        if (!(error instanceof ApiError && error.status === 401)) {
          const message = error instanceof Error ? error.message : 'Не удалось сохранить порядок задач';
          toast.error(message);
        }

        await loadData();
        return false;
      } finally {
        setIsPersistingOrder(false);
      }
    },
    [loadData, requestOptions],
  );

  const handleAddTask = () => {
    setEditingTask(null);
    setIsFormOpen(true);
  };

  const handleEditTask = (task: Task) => {
    setEditingTask(task);
    setIsFormOpen(true);
  };

  const handleSetRole = useCallback((id: string, role: 'start' | 'end' | null) => {
    if (role === 'start') {
      setStartTaskId(id);
      if (endTaskId === id) setEndTaskId('');
    } else if (role === 'end') {
      setEndTaskId(id);
      if (startTaskId === id) setStartTaskId('');
    } else {
      if (startTaskId === id) setStartTaskId('');
      if (endTaskId === id) setEndTaskId('');
    }
  }, [startTaskId, endTaskId]);

  const handleSaveTask = async (task: Task, role: TaskRole) => {
    if (!requestOptions) {
      return;
    }

    if (task.latitude === undefined || task.longitude === undefined) {
      throw new Error('Не удалось определить координаты адреса.');
    }

    setIsSavingTask(true);

    try {
      let savedId: string;
      if (editingTask) {
        const savedTask = await updateTask(
          task.id,
          {
            title: task.title,
            address: task.address,
            latitude: task.latitude,
            longitude: task.longitude,
            duration: task.duration,
            timeWindowStart: task.timeWindowStart,
            timeWindowEnd: task.timeWindowEnd,
            sortIndex: task.order ?? 0,
          },
          requestOptions,
        );

        replaceTasks(
          tasksRef.current.map((currentTask) =>
            currentTask.id === task.id
              ? { ...savedTask, completed: task.completed }
              : currentTask,
          ),
        );
        savedId = task.id;
        toast.success('Задача обновлена');
      } else {
        const savedTask = await createTask(
          {
            title: task.title,
            address: task.address,
            latitude: task.latitude,
            longitude: task.longitude,
            duration: task.duration,
            timeWindowStart: task.timeWindowStart,
            timeWindowEnd: task.timeWindowEnd,
            sortIndex: tasksRef.current.length,
          },
          requestOptions,
        );

        replaceTasks([...tasksRef.current, savedTask]);
        savedId = savedTask.id;
        toast.success('Задача добавлена');
      }

      if (role === 'start') {
        setStartTaskId(savedId);
        if (endTaskId === savedId) setEndTaskId('');
      } else if (role === 'end') {
        setEndTaskId(savedId);
        if (startTaskId === savedId) setStartTaskId('');
      } else {
        if (startTaskId === savedId) setStartTaskId('');
        if (endTaskId === savedId) setEndTaskId('');
      }

      setRouteOptimized(false);
      setOptimizedInfo(null);
      setActiveRouteId(null);
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Не удалось сохранить задачу';
      toast.error(message);
      throw error;
    } finally {
      setIsSavingTask(false);
    }
  };

  const handleDeleteTask = async (id: string) => {
    if (!requestOptions) {
      return;
    }

    try {
      await deleteTask(id, requestOptions);
      replaceTasks(tasksRef.current.filter((task) => task.id !== id));
      setCompletedTaskIds((currentIds) => currentIds.filter((taskId) => taskId !== id));
      if (startTaskId === id) setStartTaskId('');
      if (endTaskId === id) setEndTaskId('');
      setRouteOptimized(false);
      setOptimizedInfo(null);
      setActiveRouteId(null);
      toast.success('Задача удалена');
    } catch (error) {
      if (!(error instanceof ApiError && error.status === 401)) {
        const message = error instanceof Error ? error.message : 'Не удалось удалить задачу';
        toast.error(message);
      }
    }
  };

  const handleToggleComplete = (id: string) => {
    setCompletedTaskIds((currentIds) =>
      currentIds.includes(id)
        ? currentIds.filter((taskId) => taskId !== id)
        : [...currentIds, id],
    );

    replaceTasks(
      tasksRef.current.map((task) =>
        task.id === id ? { ...task, completed: !task.completed } : task,
      ),
    );
  };

  const handleReorderTasks = (dragIndex: number, hoverIndex: number) => {
    if (dragIndex === hoverIndex) {
      return;
    }

    const reordered = [...tasksRef.current];
    const [draggedTask] = reordered.splice(dragIndex, 1);
    reordered.splice(hoverIndex, 0, draggedTask);

    replaceTasks(reordered);
    setRouteOptimized(false);
    setOptimizedInfo(null);
    setActiveRouteId(null);
  };

  const handlePersistReorder = async () => {
    await persistTaskOrder(tasksRef.current);
  };

  const handleOptimizeRoute = async () => {
    if (!requestOptions || tasksRef.current.length < 2) {
      return;
    }

    const START_TIME_MINS = 540;
    const constraintIssues = validateStartEndConstraints(
      tasksRef.current,
      startTaskId,
      endTaskId,
      START_TIME_MINS,
    );
    if (constraintIssues.length > 0) {
      for (const issue of constraintIssues) {
        toast.warning(issue, { duration: 8000 });
      }
      return;
    }

    setIsOptimizing(true);
    toast.info('Построение матрицы расстояний через Яндекс...');

    try {
      // Шаг 1: строим матрицу расстояний через Yandex Maps JS API —
      // тот же движок, что рисует маршрут на карте (учёт пробок, реальная сеть).
      // Это гарантирует, что порядок задач при оптимизации совпадает с
      // маршрутом, отображаемым на экране.
      const distanceMatrix = await buildYandexDistanceMatrix(tasksRef.current, transportMode);

      toast.info('Оптимизация маршрута...');

      // Шаг 2: бэкенд строит граф из матрицы, запускает алгоритм NNH-TW
      // и возвращает оптимальный порядок задач.
      const constraintPairs: PrecedenceConstraint[] = precedences
        .filter((p) => p.beforeId && p.afterId && p.beforeId !== p.afterId)
        .map((p) => ({ beforeTaskId: p.beforeId, afterTaskId: p.afterId }));

      const result = await optimizeRoute(
        tasksRef.current.map((t) => t.id),
        requestOptions,
        540,
        distanceMatrix,
        startTaskId || undefined,
        endTaskId || undefined,
        constraintPairs.length > 0 ? constraintPairs : undefined,
      );

      const orderedTasks = normalizeTasksOrder(reorderTasksByIds(tasksRef.current, result.orderedTaskIds));
      replaceTasks(orderedTasks);

      const orderSaved = await persistTaskOrder(orderedTasks);
      if (!orderSaved) {
        return;
      }

      setSavedRoutes((currentRoutes) => [
        { id: result.id, status: result.status, source: result.source, createdAt: result.createdAt },
        ...currentRoutes,
      ]);
      setOptimizedInfo({
        tasks: orderedTasks,
        totalDistance: result.totalDistanceKm ?? 0,
        totalTravelTime: result.totalTravelTimeMin ?? 0,
        totalDuration:
          result.totalDurationMin ?? orderedTasks.reduce((sum, t) => sum + t.duration, 0),
      });
      setRouteOptimized(true);
      setActiveRouteId(result.id);
      toast.success('Маршрут оптимизирован и сохранён');
    } catch (error) {
      if (!(error instanceof ApiError && error.status === 401)) {
        const message = error instanceof Error ? error.message : 'Ошибка при оптимизации маршрута';
        toast.error(message);
      }
    } finally {
      setIsOptimizing(false);
    }
  };

  const handleApplyRoute = async (routeId: string) => {
    if (!requestOptions) {
      return;
    }

    setLoadingRouteId(routeId);

    try {
      const route = await getRoute(routeId, requestOptions);
      const orderedTasks = normalizeTasksOrder(reorderTasksByIds(tasksRef.current, route.orderedTaskIds));
      replaceTasks(orderedTasks);

      const orderSaved = await persistTaskOrder(orderedTasks);
      if (!orderSaved) {
        return;
      }

      setRouteOptimized(true);
      setActiveRouteId(routeId);
      setOptimizedInfo(
        route.totalDurationMin !== undefined ||
          route.totalDistanceKm !== undefined ||
          route.totalTravelTimeMin !== undefined
          ? {
              tasks: orderedTasks,
              totalDistance: route.totalDistanceKm ?? 0,
              totalDuration: route.totalDurationMin ?? orderedTasks.reduce((sum, task) => sum + task.duration, 0),
              totalTravelTime: route.totalTravelTimeMin ?? 0,
            }
          : null,
      );
      toast.success('Сохранённый маршрут применён');
    } catch (error) {
      if (!(error instanceof ApiError && error.status === 401)) {
        const message = error instanceof Error ? error.message : 'Не удалось загрузить маршрут';
        toast.error(message);
      }
    } finally {
      setLoadingRouteId(null);
    }
  };

  const handleDeleteRoute = async (routeId: string) => {
    if (!requestOptions) return;
    try {
      await deleteRoute(routeId, requestOptions);
      setSavedRoutes((current) => current.filter((r) => r.id !== routeId));
      if (activeRouteId === routeId) {
        setActiveRouteId(null);
        setRouteOptimized(false);
        setOptimizedInfo(null);
      }
      toast.success('Маршрут удалён');
    } catch (error) {
      if (!(error instanceof ApiError && error.status === 401)) {
        toast.error(error instanceof Error ? error.message : 'Не удалось удалить маршрут');
      }
    }
  };

  const handleDeleteAllRoutes = async () => {
    if (!requestOptions) return;
    try {
      await deleteAllRoutes(requestOptions);
      setSavedRoutes([]);
      setActiveRouteId(null);
      setRouteOptimized(false);
      setOptimizedInfo(null);
      toast.success('Все маршруты удалены');
    } catch (error) {
      if (!(error instanceof ApiError && error.status === 401)) {
        toast.error(error instanceof Error ? error.message : 'Не удалось очистить маршруты');
      }
    }
  };

  const handleStartRename = (routeId: string, currentName: string | undefined, source: string) => {
    setEditingRouteId(routeId);
    setEditingRouteName(currentName ?? (source === 'optimized' ? 'Оптимизированный маршрут' : 'Маршрут'));
  };

  const handleConfirmRename = async () => {
    if (!requestOptions || !editingRouteId) return;
    const name = editingRouteName.trim();
    if (!name) {
      setEditingRouteId(null);
      return;
    }
    try {
      await renameRoute(editingRouteId, name, requestOptions);
      setSavedRoutes((current) =>
        current.map((r) => (r.id === editingRouteId ? { ...r, name } : r)),
      );
    } catch (error) {
      if (!(error instanceof ApiError && error.status === 401)) {
        toast.error(error instanceof Error ? error.message : 'Не удалось переименовать маршрут');
      }
    } finally {
      setEditingRouteId(null);
    }
  };

  const handleExportRoute = () => {
    const exportData = {
      exportDate: new Date().toISOString(),
      optimized: routeOptimized,
      routeId: activeRouteId,
      tasks: tasks.map((task, index) => ({
        order: index + 1,
        id: task.id,
        title: task.title,
        address: task.address,
        coordinates: {
          latitude: task.latitude,
          longitude: task.longitude,
        },
        duration: task.duration,
        timeWindow: {
          start: task.timeWindowStart,
          end: task.timeWindowEnd,
        },
        completed: task.completed,
      })),
      summary: optimizedInfo
        ? {
            totalDistance: optimizedInfo.totalDistance,
            totalDuration: optimizedInfo.totalDuration,
            totalTravelTime: optimizedInfo.totalTravelTime,
          }
        : null,
    };

    const dataStr = JSON.stringify(exportData, null, 2);
    const dataBlob = new Blob([dataStr], { type: 'application/json' });
    const url = URL.createObjectURL(dataBlob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `route-${new Date().toISOString().split('T')[0]}.json`;
    link.click();
    URL.revokeObjectURL(url);

    toast.success('Маршрут экспортирован');
  };

  if (isLoading) {
    return (
      <div className="container mx-auto px-4 py-10">
        <Card>
          <CardContent className="flex items-center justify-center gap-3 py-12 text-gray-600">
            <Loader2 className="h-5 w-5 animate-spin" />
            Загрузка задач и маршрутов...
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <DndProvider backend={HTML5Backend}>
      <div className="container mx-auto px-4 py-6">
        <div className="sticky top-[80px] z-30 mb-6 flex flex-wrap gap-3 bg-gray-50 py-2 border-b border-gray-100">
          <Tooltip>
            <TooltipTrigger asChild>
              <Button onClick={handleAddTask}>
                <Plus className="mr-2 h-4 w-4" />
                Добавить задачу
              </Button>
            </TooltipTrigger>
            <TooltipContent>
              <p>Создать новую задачу с адресом и временными рамками</p>
            </TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                onClick={handleOptimizeRoute}
                disabled={isOptimizing || isPersistingOrder || tasks.length < 2}
                variant="default"
                className="bg-blue-600 hover:bg-blue-700"
              >
                {isOptimizing ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <Route className="mr-2 h-4 w-4" />
                )}
                Оптимизировать маршрут
              </Button>
            </TooltipTrigger>
            <TooltipContent>
              <p>Построить маршрут и сохранить его в бэкенде</p>
            </TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                onClick={handleExportRoute}
                disabled={tasks.length === 0}
                variant="outline"
              >
                <Download className="mr-2 h-4 w-4" />
                Экспорт в JSON
              </Button>
            </TooltipTrigger>
            <TooltipContent>
              <p>Скачать текущий маршрут в формате JSON</p>
            </TooltipContent>
          </Tooltip>
        </div>

        {optimizedInfo && routeOptimized && (
          <Card className="mb-6 border-blue-200 bg-blue-50">
            <CardContent className="p-4">
              <div className="flex items-start gap-3">
                <Info className="mt-0.5 h-5 w-5 flex-shrink-0 text-blue-600" />
                <div className="flex-1">
                  <h3 className="mb-2 font-semibold text-blue-900">Информация о маршруте</h3>
                  <div className="grid grid-cols-1 gap-4 text-sm md:grid-cols-3">
                    <div className="flex items-center gap-2">
                      <Navigation className="h-4 w-4 text-blue-600" />
                      <span className="text-gray-700">
                        Расстояние: <strong>{optimizedInfo.totalDistance} км</strong>
                      </span>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <Info className="h-3.5 w-3.5 text-gray-400 cursor-help flex-shrink-0" />
                        </TooltipTrigger>
                        <TooltipContent>
                          <p>Суммарное расстояние по дорогам между всеми точками маршрута</p>
                        </TooltipContent>
                      </Tooltip>
                    </div>
                    <div className="flex items-center gap-2">
                      <Clock className="h-4 w-4 text-blue-600" />
                      <span className="text-gray-700">
                        Время в пути: <strong>{optimizedInfo.totalTravelTime} мин</strong>
                      </span>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <Info className="h-3.5 w-3.5 text-gray-400 cursor-help flex-shrink-0" />
                        </TooltipTrigger>
                        <TooltipContent>
                          <p>Суммарное время переездов между точками (без учёта выполнения задач)</p>
                        </TooltipContent>
                      </Tooltip>
                    </div>
                    <div className="flex items-center gap-2">
                      <Clock className="h-4 w-4 text-blue-600" />
                      <span className="text-gray-700">
                        Общее время: <strong>{optimizedInfo.totalDuration} мин</strong>
                      </span>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <Info className="h-3.5 w-3.5 text-gray-400 cursor-help flex-shrink-0" />
                        </TooltipTrigger>
                        <TooltipContent>
                          <p>Время в пути + суммарная длительность выполнения всех задач</p>
                        </TooltipContent>
                      </Tooltip>
                    </div>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        )}

        <Card className="mb-6">
          <CardHeader className="flex flex-row items-center justify-between gap-4 space-y-0">
            <CardTitle className="flex items-center gap-2">
              <History className="h-5 w-5" />
              Сохранённые маршруты
              <span className="text-sm font-normal text-gray-500">({savedRoutes.length})</span>
            </CardTitle>
            {savedRoutes.length > 0 && (
              <Button
                variant="outline"
                size="sm"
                className="text-red-600 hover:text-red-700 hover:border-red-300"
                onClick={() => void handleDeleteAllRoutes()}
              >
                <Trash2 className="mr-1.5 h-3.5 w-3.5" />
                Очистить всё
              </Button>
            )}
          </CardHeader>
          <CardContent>
            {savedRoutes.length === 0 ? (
              <p className="text-sm text-gray-500">Маршруты ещё не сохранялись.</p>
            ) : (
              <div className="max-h-64 overflow-y-auto pr-1">
                <div className="space-y-3">
                  {savedRoutes.map((route) => (
                    <div
                      key={route.id}
                      className="flex flex-col gap-3 rounded-lg border p-3 md:flex-row md:items-center md:justify-between"
                    >
                      <div className="flex-1 min-w-0">
                        {editingRouteId === route.id ? (
                          <div className="flex items-center gap-2">
                            <input
                              autoFocus
                              className="flex-1 min-w-0 rounded border border-gray-300 px-2 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                              value={editingRouteName}
                              onChange={(e) => setEditingRouteName(e.target.value)}
                              onKeyDown={(e) => {
                                if (e.key === 'Enter') void handleConfirmRename();
                                if (e.key === 'Escape') setEditingRouteId(null);
                              }}
                            />
                            <button
                              className="text-green-600 hover:text-green-700"
                              onClick={() => void handleConfirmRename()}
                            >
                              <Check className="h-4 w-4" />
                            </button>
                            <button
                              className="text-gray-400 hover:text-gray-600"
                              onClick={() => setEditingRouteId(null)}
                            >
                              <X className="h-4 w-4" />
                            </button>
                          </div>
                        ) : (
                          <div className="flex items-center gap-1.5">
                            <div className="font-medium text-gray-900 truncate">
                              {route.name ?? (route.source === 'optimized' ? 'Оптимизированный маршрут' : 'Маршрут')}
                            </div>
                            {route.source === 'optimized' && (
                              <button
                                className="flex-shrink-0 text-gray-400 hover:text-gray-600"
                                onClick={() => handleStartRename(route.id, route.name, route.source)}
                              >
                                <Pencil className="h-3.5 w-3.5" />
                              </button>
                            )}
                          </div>
                        )}
                        <div className="text-sm text-gray-500">
                          {new Date(route.createdAt).toLocaleString('ru-RU')}
                        </div>
                      </div>
                      <div className="flex items-center gap-2 flex-shrink-0">
                        <span className="rounded-full bg-gray-100 px-2 py-1 text-xs uppercase text-gray-600">
                          {route.status}
                        </span>
                        <Button
                          variant={route.id === activeRouteId ? 'default' : 'outline'}
                          size="sm"
                          disabled={loadingRouteId === route.id || tasks.length === 0}
                          onClick={() => void handleApplyRoute(route.id)}
                        >
                          {loadingRouteId === route.id && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                          Применить
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="text-red-400 hover:text-red-600 hover:bg-red-50 px-2"
                          onClick={() => void handleDeleteRoute(route.id)}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </CardContent>
        </Card>

        {tasks.length >= 2 && (
          <Card className="mb-6 border-gray-200">
            <CardContent className="p-4 space-y-4">
              <div>
                <div className="mb-2 flex items-center justify-between">
                  <div className="flex items-center gap-1.5">
                    <span className="text-sm font-medium text-gray-700">
                      Ограничения порядка выполнения задач
                    </span>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <Info className="h-3.5 w-3.5 text-gray-400 cursor-help flex-shrink-0" />
                      </TooltipTrigger>
                      <TooltipContent>
                        <p>Первая задача выполняется раньше второй.<br />Ограничения учитываются при оптимизации.</p>
                      </TooltipContent>
                    </Tooltip>
                  </div>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setPrecedences((prev) => [...prev, { beforeId: '', afterId: '' }])}
                  >
                    <Plus className="mr-1 h-3.5 w-3.5" />
                    Добавить
                  </Button>
                </div>
                {precedences.length === 0 && (
                  <p className="text-sm text-gray-400">Ограничения не заданы.</p>
                )}
                <div className="space-y-2">
                  {precedences.map((pair, idx) => (
                    <div key={idx} className="flex items-center gap-2">
                      <select
                        className="flex-1 rounded border border-gray-300 px-2 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                        value={pair.beforeId}
                        onChange={(e) =>
                          setPrecedences((prev) =>
                            prev.map((p, i) => (i === idx ? { ...p, beforeId: e.target.value } : p)),
                          )
                        }
                      >
                        <option value="" disabled>Задача A</option>
                        {tasks.map((t) => (
                          <option key={t.id} value={t.id} disabled={t.id === pair.afterId}>
                            {t.title}
                          </option>
                        ))}
                      </select>
                      <span className="text-sm text-gray-500">до</span>
                      <select
                        className="flex-1 rounded border border-gray-300 px-2 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                        value={pair.afterId}
                        onChange={(e) =>
                          setPrecedences((prev) =>
                            prev.map((p, i) => (i === idx ? { ...p, afterId: e.target.value } : p)),
                          )
                        }
                      >
                        <option value="" disabled>Задача B</option>
                        {tasks.map((t) => (
                          <option key={t.id} value={t.id} disabled={t.id === pair.beforeId}>
                            {t.title}
                          </option>
                        ))}
                      </select>
                      <button
                        className="text-gray-400 hover:text-red-500"
                        onClick={() => setPrecedences((prev) => prev.filter((_, i) => i !== idx))}
                      >
                        <X className="h-4 w-4" />
                      </button>
                    </div>
                  ))}
                </div>
              </div>
            </CardContent>
          </Card>
        )}

        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className="h-[calc(100vh-420px)] min-h-[400px]">
            <TaskList
              tasks={tasks}
              startTaskId={startTaskId || undefined}
              endTaskId={endTaskId || undefined}
              onEdit={handleEditTask}
              onDelete={handleDeleteTask}
              onToggleComplete={handleToggleComplete}
              onReorder={handleReorderTasks}
              onReorderEnd={handlePersistReorder}
              onSetRole={handleSetRole}
            />
          </div>

          <div className="h-[calc(100vh-420px)] min-h-[400px]">
            <MapView
              tasks={tasks}
              routeOptimized={routeOptimized}
              onTransportModeChange={setTransportMode}
              onRouteLegsChange={setRouteLegs}
            />
          </div>
        </div>

        {routeOptimized && (
          <RouteStepList
            tasks={tasks}
            transportMode={transportMode}
            routeLegs={routeLegs}
          />
        )}

        <TaskForm
          task={editingTask}
          isOpen={isFormOpen}
          isSaving={isSavingTask}
          initialRole={
            editingTask
              ? editingTask.id === startTaskId
                ? 'start'
                : editingTask.id === endTaskId
                  ? 'end'
                  : null
              : null
          }
          canSetStart={!startTaskId || editingTask?.id === startTaskId}
          canSetEnd={!endTaskId || editingTask?.id === endTaskId}
          onClose={() => setIsFormOpen(false)}
          onSave={handleSaveTask}
          onGeocodeMultiple={geocodeAddressSuggestions}
        />
      </div>
    </DndProvider>
  );
}
