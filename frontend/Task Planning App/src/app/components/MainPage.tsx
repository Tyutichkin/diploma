import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import Papa from 'papaparse';
import * as XLSX from 'xlsx';
import { Task, OptimizedRoute, taskHasAddress, windowBoundMs, formatWindowBound } from '../types/task';
import { SavedRouteSummary, RouteStopTiming } from '../types/route';
import { TaskList } from './TaskList';
import { TaskForm, TaskRole } from './TaskForm';
import { TaskImport, ImportedTaskRow } from './TaskImport';
import { MapView, TransportMode, RouteLeg } from './MapView';
import { RouteStepList } from './RouteStepList';
import { Button } from './ui/button';
import { Card, CardContent, CardHeader, CardTitle } from './ui/card';
import { Tooltip, TooltipContent, TooltipTrigger } from './ui/tooltip';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from './ui/dialog';
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
  Upload,
} from 'lucide-react';
import { toast } from 'sonner';
import { buildYandexDistanceMatrix, geocodeAddressSuggestions } from '../utils/routeOptimizer';
import {
  ApiError,
  PrecedenceConstraint,
  createTask,
  createTasksBatch,
  deleteAllRoutes,
  deleteRoute,
  deleteTask,
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

// Скрываем блок «Сохранённые маршруты» в UI. Бэкенд продолжает сохранять маршрут
// при оптимизации — это нужно для восстановления активного маршрута.
const SHOW_SAVED_ROUTES = false;

function normalizeTasksOrder(tasks: Task[]) {
  return tasks.map((task, index) => ({
    ...task,
    order: index,
  }));
}

// Сообщение пользователю на русском. Для ApiError (ошибка бэкенда) показываем
// заготовленный фолбэк, чтобы не выводить сырой английский текст от сервера.
// Для обычных Error используем их message — там уже русские сообщения фронта.
function userErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof ApiError) return fallback;
  if (error instanceof Error) return error.message;
  return fallback;
}


function validateStartEndConstraints(
  tasks: Task[],
  startTaskId: string,
  endTaskId: string,
): string[] {
  const issues: string[] = [];
  const startTask = startTaskId ? tasks.find((t) => t.id === startTaskId) : null;
  const endTask = endTaskId ? tasks.find((t) => t.id === endTaskId) : null;

  if (!startTask && !endTask) return issues;

  // Начальная точка vs конечная точка
  if (startTask && endTask) {
    const sWinStart = windowBoundMs(startTask.windowStartDate, startTask.windowStartTime);
    const sWinEnd = windowBoundMs(startTask.windowEndDate, startTask.windowEndTime);
    const eWinEnd = windowBoundMs(endTask.windowEndDate, endTask.windowEndTime);
    const eWinStart = windowBoundMs(endTask.windowStartDate, endTask.windowStartTime);

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

  // Промежуточные задачи vs начальная точка
  if (startTask?.windowStartDate) {
    const sWinStart = windowBoundMs(startTask.windowStartDate, startTask.windowStartTime);
    if (sWinStart !== null) {
      for (const task of tasks) {
        if (task.id === startTaskId || task.id === endTaskId) continue;
        const tWinEnd = windowBoundMs(task.windowEndDate, task.windowEndTime);
        if (tWinEnd !== null && tWinEnd <= sWinStart) {
          issues.push(
            `Задача "${task.title}" должна быть завершена до ${formatWindowBound(task.windowEndDate, task.windowEndTime)}, но начальная точка "${startTask.title}" не откроется раньше ${formatWindowBound(startTask.windowStartDate, startTask.windowStartTime)}.`,
          );
        }
      }
    }
  }

  // Промежуточные задачи vs конечная точка
  if (endTask?.windowEndDate) {
    const eWinEnd = windowBoundMs(endTask.windowEndDate, endTask.windowEndTime);
    if (eWinEnd !== null) {
      for (const task of tasks) {
        if (task.id === startTaskId || task.id === endTaskId) continue;
        const tWinStart = windowBoundMs(task.windowStartDate, task.windowStartTime);
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
  const [activeRouteId, setActiveRouteId] = useState<string | null>(null);
  const [loadingRouteId, setLoadingRouteId] = useState<string | null>(null);
  const [editingRouteId, setEditingRouteId] = useState<string | null>(null);
  const [editingRouteName, setEditingRouteName] = useState('');
  const [transportMode, setTransportMode] = useState<TransportMode>('auto');
  const [routeLegs, setRouteLegs] = useState<RouteLeg[]>([]);
  const [routeStops, setRouteStops] = useState<RouteStopTiming[]>([]);
  const [startTaskId, setStartTaskId] = useState<string>('');
  const [endTaskId, setEndTaskId] = useState<string>('');
  const [precedences, setPrecedences] = useState<PrecedencePair[]>([]);
  const [isImportOpen, setIsImportOpen] = useState(false);
  const [isExportOpen, setIsExportOpen] = useState(false);

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

  const loadData = useCallback(async () => {
    if (!requestOptions || !session) {
      return;
    }

    setIsLoading(true);

    try {
      const [serverTasks, routes] = await Promise.all([
        listTasks(requestOptions),
        SHOW_SAVED_ROUTES ? listRoutes(requestOptions) : Promise.resolve([] as SavedRouteSummary[]),
      ]);

      replaceTasks(serverTasks);
      setSavedRoutes(routes);
    } catch (error) {
      if (!(error instanceof ApiError && error.status === 401)) {
        toast.error(userErrorMessage(error, 'Не удалось загрузить данные'));
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
          toast.error(userErrorMessage(error, 'Не удалось сохранить порядок задач'));
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

  const handleImportTasks = async (rows: ImportedTaskRow[]) => {
    if (!requestOptions) return;

    const geocodeResults = await Promise.all(
      rows.map(async (row) => {
        if (!row.address) return { lat: undefined, lng: undefined, address: undefined, failed: false };
        try {
          const suggestions = await geocodeAddressSuggestions(row.address);
          if (suggestions.length > 0) {
            return { lat: suggestions[0].lat, lng: suggestions[0].lng, address: suggestions[0].displayName, failed: false };
          }
          return { lat: undefined, lng: undefined, address: undefined, failed: true };
        } catch {
          return { lat: undefined, lng: undefined, address: undefined, failed: true };
        }
      }),
    );

    const geocodeFailed = geocodeResults.filter((r) => r.failed).length;
    const baseIndex = tasksRef.current.length;

    const inputs = rows.map((row, i) => ({
      title: row.title,
      address: geocodeResults[i].address,
      latitude: geocodeResults[i].lat,
      longitude: geocodeResults[i].lng,
      duration: row.duration,
      windowStartDate: row.windowDate,
      windowStartTime: row.windowStartTime,
      windowEndDate: row.windowDate,
      windowEndTime: row.windowEndTime,
      sortIndex: baseIndex + i,
    }));

    const savedTasks = await createTasksBatch(inputs, requestOptions);
    replaceTasks([...tasksRef.current, ...savedTasks]);

    toast.success(
      geocodeFailed > 0
        ? `Импортировано ${savedTasks.length} задач. Геокодирование не удалось для ${geocodeFailed} адресов — они сохранены без координат.`
        : `Импортировано ${savedTasks.length} задач`,
    );
    setRouteOptimized(false);
    setOptimizedInfo(null);
    setRouteStops([]);
    setActiveRouteId(null);
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

    // TaskForm гарантирует координаты для задач с адресом — защита от гонок
    if (task.address && (task.latitude === undefined || task.longitude === undefined)) {
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
            windowStartDate: task.windowStartDate,
            windowStartTime: task.windowStartTime,
            windowEndDate: task.windowEndDate,
            windowEndTime: task.windowEndTime,
            sortIndex: task.order ?? 0,
          },
          requestOptions,
        );

        replaceTasks(
          tasksRef.current.map((currentTask) =>
            currentTask.id === task.id ? savedTask : currentTask,
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
            windowStartDate: task.windowStartDate,
            windowStartTime: task.windowStartTime,
            windowEndDate: task.windowEndDate,
            windowEndTime: task.windowEndTime,
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
    setRouteStops([]);
      setActiveRouteId(null);
    } catch (error) {
      toast.error(userErrorMessage(error, 'Не удалось сохранить задачу'));
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
      if (startTaskId === id) setStartTaskId('');
      if (endTaskId === id) setEndTaskId('');
      setRouteOptimized(false);
      setOptimizedInfo(null);
    setRouteStops([]);
      setActiveRouteId(null);
      toast.success('Задача удалена');
    } catch (error) {
      if (!(error instanceof ApiError && error.status === 401)) {
        toast.error(userErrorMessage(error, 'Не удалось удалить задачу'));
      }
    }
  };

  const handleCloneTask = async (task: Task) => {
    if (!requestOptions) {
      return;
    }

    try {
      const cloned = await createTask(
        {
          title: `${task.title} (копия)`,
          address: task.address,
          latitude: task.latitude,
          longitude: task.longitude,
          duration: task.duration,
          windowStartDate: task.windowStartDate,
          windowStartTime: task.windowStartTime,
          windowEndDate: task.windowEndDate,
          windowEndTime: task.windowEndTime,
          sortIndex: tasksRef.current.length,
        },
        requestOptions,
      );
      replaceTasks([...tasksRef.current, cloned]);
      toast.success('Задача клонирована');
    } catch (error) {
      if (!(error instanceof ApiError && error.status === 401)) {
        toast.error(userErrorMessage(error, 'Не удалось клонировать задачу'));
      }
    }
  };

  const handleToggleComplete = async (id: string) => {
    if (!requestOptions) {
      return;
    }

    const task = tasksRef.current.find((t) => t.id === id);
    if (!task) {
      return;
    }

    const newCompleted = !task.completed;

    replaceTasks(
      tasksRef.current.map((t) =>
        t.id === id ? { ...t, completed: newCompleted } : t,
      ),
    );

    try {
      await updateTask(id, { completed: newCompleted }, requestOptions);
    } catch (error) {
      replaceTasks(
        tasksRef.current.map((t) =>
          t.id === id ? { ...t, completed: task.completed } : t,
        ),
      );
      if (!(error instanceof ApiError && error.status === 401)) {
        toast.error('Не удалось сохранить статус задачи');
      }
    }
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
    setRouteStops([]);
    setActiveRouteId(null);
  };

  const handlePersistReorder = async () => {
    await persistTaskOrder(tasksRef.current);
  };

  const runOptimization = async (mode: TransportMode) => {
    if (!requestOptions) {
      return;
    }

    const addressTasks = tasksRef.current.filter(taskHasAddress);
    const addresslessTasks = tasksRef.current.filter((t) => !taskHasAddress(t));

    if (addressTasks.length < 2) {
      toast.warning('Для оптимизации нужно минимум 2 задачи с адресом.');
      return;
    }

    const constraintIssues = validateStartEndConstraints(
      addressTasks,
      startTaskId,
      endTaskId,
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
      const distanceMatrix = await buildYandexDistanceMatrix(addressTasks, mode);

      toast.info('Оптимизация маршрута...');

      const constraintPairs: PrecedenceConstraint[] = precedences
        .filter((p) => p.beforeId && p.afterId && p.beforeId !== p.afterId)
        .map((p) => ({ beforeTaskId: p.beforeId, afterTaskId: p.afterId }));

      const result = await optimizeRoute(
        addressTasks.map((t) => t.id),
        requestOptions,
        0,
        distanceMatrix,
        startTaskId || undefined,
        endTaskId || undefined,
        constraintPairs.length > 0 ? constraintPairs : undefined,
      );

      // задачи без адреса уходят в конец списка
      const orderedAddressTasks = reorderTasksByIds(addressTasks, result.orderedTaskIds);
      const orderedTasks = normalizeTasksOrder([...orderedAddressTasks, ...addresslessTasks]);
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
        tasks: orderedAddressTasks,
        totalDistance: result.totalDistanceKm ?? 0,
        totalTravelTime: result.totalTravelTimeMin ?? 0,
        totalDuration:
          result.totalDurationMin ?? orderedAddressTasks.reduce((sum, t) => sum + t.duration, 0),
      });
      setRouteStops(result.stops ?? []);
      setRouteOptimized(true);
      setActiveRouteId(result.id);
      toast.success('Маршрут оптимизирован и сохранён');
    } catch (error) {
      if (!(error instanceof ApiError && error.status === 401)) {
        toast.error(userErrorMessage(error, 'Ошибка при оптимизации маршрута'));
      }
    } finally {
      setIsOptimizing(false);
    }
  };

  const handleOptimizeRoute = () => runOptimization(transportMode);

  const handleTransportModeChange = (mode: TransportMode) => {
    setTransportMode(mode);
    if (routeOptimized) {
      void runOptimization(mode);
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
      setRouteStops(route.stops ?? []);
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
        toast.error(userErrorMessage(error, 'Не удалось загрузить маршрут'));
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
    setRouteStops([]);
      }
      toast.success('Маршрут удалён');
    } catch (error) {
      if (!(error instanceof ApiError && error.status === 401)) {
        toast.error(userErrorMessage(error, 'Не удалось удалить маршрут'));
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
    setRouteStops([]);
      toast.success('Все маршруты удалены');
    } catch (error) {
      if (!(error instanceof ApiError && error.status === 401)) {
        toast.error(userErrorMessage(error, 'Не удалось очистить маршруты'));
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
        toast.error(userErrorMessage(error, 'Не удалось переименовать маршрут'));
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
          startDate: task.windowStartDate,
          startTime: task.windowStartTime,
          endDate: task.windowEndDate,
          endTime: task.windowEndTime,
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

  function buildExportRows() {
    return tasks.map((task) => ({
      title: task.title,
      address: task.address ?? '',
      duration: task.duration ?? '',
      date: task.windowStartDate ?? '',
      window_start: task.windowStartTime ?? '',
      window_end: task.windowEndTime ?? '',
    }));
  }

  const handleExportCSV = () => {
    if (tasks.length === 0) return;
    const csv = Papa.unparse(buildExportRows(), {
      columns: ['title', 'address', 'duration', 'date', 'window_start', 'window_end'],
    });
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `tasks-${new Date().toISOString().split('T')[0]}.csv`;
    link.click();
    URL.revokeObjectURL(url);
    toast.success('Задачи экспортированы в CSV');
  };

  const handleExportXLSX = () => {
    if (tasks.length === 0) return;
    const ws = XLSX.utils.json_to_sheet(buildExportRows(), {
      header: ['title', 'address', 'duration', 'date', 'window_start', 'window_end'],
    });
    ws['!cols'] = [
      { wch: 30 }, // title
      { wch: 35 }, // address
      { wch: 18 }, // duration
      { wch: 14 }, // date
      { wch: 15 }, // window_start
      { wch: 15 }, // window_end
    ];
    const wb = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(wb, ws, 'Задачи');
    XLSX.writeFile(wb, `tasks-${new Date().toISOString().split('T')[0]}.xlsx`);
    toast.success('Задачи экспортированы в Excel');
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
              <Button variant="outline" onClick={() => setIsImportOpen(true)}>
                <Upload className="mr-2 h-4 w-4" />
                Импорт из файла
              </Button>
            </TooltipTrigger>
            <TooltipContent>
              <p>Загрузить задачи из CSV или Excel-файла</p>
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
                onClick={() => setIsExportOpen(true)}
                disabled={tasks.length === 0}
                variant="outline"
              >
                <Download className="mr-2 h-4 w-4" />
                Экспорт
              </Button>
            </TooltipTrigger>
            <TooltipContent>
              <p>Скачать задачи в выбранном формате</p>
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

        {SHOW_SAVED_ROUTES && (
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
        )}

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
              onClone={handleCloneTask}
            />
          </div>

          <div className="h-[calc(100vh-420px)] min-h-[400px]">
            <MapView
              tasks={tasks}
              routeOptimized={routeOptimized}
              onTransportModeChange={handleTransportModeChange}
              onRouteLegsChange={setRouteLegs}
            />
          </div>
        </div>

        {routeOptimized && (
          <RouteStepList
            tasks={tasks}
            transportMode={transportMode}
            routeLegs={routeLegs}
            stops={routeStops}
            onEditTask={(taskId) => {
              const task = tasks.find((t) => t.id === taskId);
              if (task) {
                setEditingTask(task);
                setIsFormOpen(true);
              }
            }}
            onDeleteTask={(taskId) => {
              void handleDeleteTask(taskId);
            }}
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
        <TaskImport
          open={isImportOpen}
          onOpenChange={setIsImportOpen}
          onImport={handleImportTasks}
        />

        <Dialog open={isExportOpen} onOpenChange={setIsExportOpen}>
          <DialogContent className="max-w-sm">
            <DialogHeader>
              <DialogTitle>Экспорт задач</DialogTitle>
              <DialogDescription>
                Выберите формат файла для скачивания
              </DialogDescription>
            </DialogHeader>
            <div className="flex flex-col gap-3 pt-2">
              <Button
                variant="outline"
                className="justify-start"
                onClick={() => { handleExportCSV(); setIsExportOpen(false); }}
              >
                <Download className="mr-2 h-4 w-4" />
                CSV
                <span className="ml-auto text-xs text-gray-400">совместим с импортом</span>
              </Button>
              <Button
                variant="outline"
                className="justify-start"
                onClick={() => { handleExportXLSX(); setIsExportOpen(false); }}
              >
                <Download className="mr-2 h-4 w-4" />
                Excel (.xlsx)
                <span className="ml-auto text-xs text-gray-400">совместим с импортом</span>
              </Button>
              <Button
                variant="outline"
                className="justify-start"
                onClick={() => { handleExportRoute(); setIsExportOpen(false); }}
              >
                <Download className="mr-2 h-4 w-4" />
                JSON
                <span className="ml-auto text-xs text-gray-400">полные данные маршрута</span>
              </Button>
            </div>
          </DialogContent>
        </Dialog>
      </div>
    </DndProvider>
  );
}
