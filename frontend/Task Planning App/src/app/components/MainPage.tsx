import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import { Loader2 } from 'lucide-react';
import { toast } from 'sonner';

import { Task, taskHasAddress, getWindowConflictIds } from '../types/task';
import { TransportMode } from '../types/transport';
import {
  ApiError,
  PrecedenceConstraint,
  createTask,
  createTasksBatch,
  deleteAllTasks,
  deleteTask,
  listTasks,
  optimizeRoute,
  reorderTasks,
  updateTask,
} from '../api/client';
import { useAuth } from '../context/auth-context';
import { useOptimizedRoute } from '../hooks/useOptimizedRoute';
import { useTaskExport } from '../hooks/useTaskExport';
import { buildYandexDistanceMatrix, geocodeAddressSuggestions } from '../utils/routeOptimizer';
import { getRouteConflictIds } from '../utils/routeConflicts';
import {
  normalizeTasksOrder,
  reorderTasksByIds,
  userErrorMessage,
  validateStartEndConstraints,
} from '../utils/tasks';

import { TaskList } from './TaskList';
import { TaskForm, TaskRole } from './TaskForm';
import { TaskImport, ImportedTaskRow } from './TaskImport';
import { MapView } from './MapView';
import { RouteStepList } from './RouteStepList';
import { Card, CardContent } from './ui/card';

import { MainToolbar } from './main/MainToolbar';
import { OptimizedRouteSummary } from './main/OptimizedRouteSummary';
import { PrecedenceConstraintsPanel, PrecedencePair } from './main/PrecedenceConstraintsPanel';
import { ExportFormatDialog } from './main/ExportFormatDialog';

export function MainPage() {
  const { session, setSession, logout } = useAuth();
  const tasksRef = useRef<Task[]>([]);

  const [tasks, setTasks] = useState<Task[]>([]);
  const [editingTask, setEditingTask] = useState<Task | null>(null);
  const [isFormOpen, setIsFormOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [isSavingTask, setIsSavingTask] = useState(false);
  const [isPersistingOrder, setIsPersistingOrder] = useState(false);
  const [isOptimizing, setIsOptimizing] = useState(false);
  const [transportMode, setTransportMode] = useState<TransportMode>('auto');
  const [startTaskId, setStartTaskId] = useState<string>('');
  const [endTaskId, setEndTaskId] = useState<string>('');
  const [precedences, setPrecedences] = useState<PrecedencePair[]>([]);
  const [isImportOpen, setIsImportOpen] = useState(false);
  const [isExportOpen, setIsExportOpen] = useState(false);

  const route = useOptimizedRoute();
  const {
    routeOptimized,
    optimizedInfo,
    routeStops,
    routeLegs,
    activeRouteId,
    apply: applyRoute,
    clear: clearRoute,
    setRouteLegs,
  } = route;

  const staticConflictIds = useMemo(() => getWindowConflictIds(tasks), [tasks]);
  const hasWindowConflicts = staticConflictIds.size > 0;
  const routeConflictIds = useMemo(() => {
    const ids = getRouteConflictIds(tasks, routeStops);
    return ids.size > 0 ? ids : undefined;
  }, [tasks, routeStops]);

  const handleUnauthorized = useCallback(async () => {
    toast.error('Сессия истекла. Выполните вход заново.');
    await logout();
  }, [logout]);

  const requestOptions = useMemo(() => {
    if (!session) return null;
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

  // 401 пробрасывается ApiError — toast не показываем (logout уже отработал).
  const reportError = useCallback((error: unknown, fallback: string) => {
    if (error instanceof ApiError && error.status === 401) return;
    toast.error(userErrorMessage(error, fallback));
  }, []);

  const loadData = useCallback(async () => {
    if (!requestOptions || !session) return;
    setIsLoading(true);
    try {
      const serverTasks = await listTasks(requestOptions);
      replaceTasks(serverTasks);
    } catch (error) {
      reportError(error, 'Не удалось загрузить данные');
    } finally {
      setIsLoading(false);
    }
  }, [replaceTasks, reportError, requestOptions, session]);

  useEffect(() => {
    void loadData();
  }, [loadData]);

  const persistTaskOrder = useCallback(
    async (orderedTasks: Task[]) => {
      if (!requestOptions) return false;
      setIsPersistingOrder(true);
      try {
        await reorderTasks(
          orderedTasks.map((task, index) => ({ id: task.id, sortIndex: index })),
          requestOptions,
        );
        return true;
      } catch (error) {
        reportError(error, 'Не удалось сохранить порядок задач');
        await loadData();
        return false;
      } finally {
        setIsPersistingOrder(false);
      }
    },
    [loadData, reportError, requestOptions],
  );

  const releaseRole = useCallback((id: string) => {
    setStartTaskId((prev) => (prev === id ? '' : prev));
    setEndTaskId((prev) => (prev === id ? '' : prev));
  }, []);

  const applyRoleChange = useCallback((id: string, role: TaskRole) => {
    if (role === 'start') {
      setStartTaskId(id);
      setEndTaskId((prev) => (prev === id ? '' : prev));
    } else if (role === 'end') {
      setEndTaskId(id);
      setStartTaskId((prev) => (prev === id ? '' : prev));
    } else {
      releaseRole(id);
    }
  }, [releaseRole]);

  const handleSetRole = useCallback(
    (id: string, role: TaskRole) => applyRoleChange(id, role),
    [applyRoleChange],
  );

  const handleAddTask = () => {
    setEditingTask(null);
    setIsFormOpen(true);
  };

  const handleEditTask = useCallback((task: Task) => {
    setEditingTask(task);
    setIsFormOpen(true);
  }, []);

  const handleImportTasks = async (rows: ImportedTaskRow[]) => {
    if (!requestOptions) return;

    const geocodeResults = await Promise.all(
      rows.map(async (row) => {
        if (!row.address) return { lat: undefined, lng: undefined, address: undefined, failed: false };
        try {
          const suggestions = await geocodeAddressSuggestions(row.address);
          if (suggestions.length > 0) {
            return {
              lat: suggestions[0].lat,
              lng: suggestions[0].lng,
              address: suggestions[0].displayName,
              failed: false,
            };
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
    clearRoute();

    toast.success(
      geocodeFailed > 0
        ? `Импортировано ${savedTasks.length} задач. Геокодирование не удалось для ${geocodeFailed} адресов — они сохранены без координат.`
        : `Импортировано ${savedTasks.length} задач`,
    );
  };

  const handleSaveTask = async (task: Task, role: TaskRole) => {
    if (!requestOptions) return;

    // TaskForm гарантирует координаты для задач с адресом — защита от гонок
    if (task.address && (task.latitude === undefined || task.longitude === undefined)) {
      throw new Error('Не удалось определить координаты адреса.');
    }

    setIsSavingTask(true);
    try {
      const payload = {
        title: task.title,
        address: task.address,
        latitude: task.latitude,
        longitude: task.longitude,
        duration: task.duration,
        windowStartDate: task.windowStartDate,
        windowStartTime: task.windowStartTime,
        windowEndDate: task.windowEndDate,
        windowEndTime: task.windowEndTime,
      };

      let savedId: string;
      if (editingTask) {
        const savedTask = await updateTask(
          task.id,
          { ...payload, sortIndex: task.order ?? 0 },
          requestOptions,
        );
        replaceTasks(tasksRef.current.map((t) => (t.id === task.id ? savedTask : t)));
        savedId = task.id;
        toast.success('Задача обновлена');
      } else {
        const savedTask = await createTask(
          { ...payload, sortIndex: tasksRef.current.length },
          requestOptions,
        );
        replaceTasks([...tasksRef.current, savedTask]);
        savedId = savedTask.id;
        toast.success('Задача добавлена');
      }

      applyRoleChange(savedId, role);
      clearRoute();
    } catch (error) {
      toast.error(userErrorMessage(error, 'Не удалось сохранить задачу'));
      throw error;
    } finally {
      setIsSavingTask(false);
    }
  };

  const handleDeleteTask = async (id: string) => {
    if (!requestOptions) return;
    try {
      await deleteTask(id, requestOptions);
      replaceTasks(tasksRef.current.filter((task) => task.id !== id));
      releaseRole(id);
      clearRoute();
      toast.success('Задача удалена');
    } catch (error) {
      reportError(error, 'Не удалось удалить задачу');
    }
  };

  const handleDeleteAllTasks = async () => {
    if (!requestOptions) return;
    if (tasksRef.current.length === 0) return;
    try {
      await deleteAllTasks(requestOptions);
      replaceTasks([]);
      setStartTaskId('');
      setEndTaskId('');
      setPrecedences([]);
      clearRoute();
      toast.success('Все задачи удалены');
    } catch (error) {
      reportError(error, 'Не удалось удалить задачи');
    }
  };

  const handleCloneTask = async (task: Task) => {
    if (!requestOptions) return;
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
      reportError(error, 'Не удалось клонировать задачу');
    }
  };

  const handleToggleComplete = async (id: string) => {
    if (!requestOptions) return;
    const task = tasksRef.current.find((t) => t.id === id);
    if (!task) return;

    const newCompleted = !task.completed;
    replaceTasks(tasksRef.current.map((t) => (t.id === id ? { ...t, completed: newCompleted } : t)));

    try {
      await updateTask(id, { completed: newCompleted }, requestOptions);
    } catch (error) {
      replaceTasks(tasksRef.current.map((t) => (t.id === id ? { ...t, completed: task.completed } : t)));
      if (!(error instanceof ApiError && error.status === 401)) {
        toast.error('Не удалось сохранить статус задачи');
      }
    }
  };

  const handleReorderTasks = (dragIndex: number, hoverIndex: number) => {
    if (dragIndex === hoverIndex) return;
    const reordered = [...tasksRef.current];
    const [draggedTask] = reordered.splice(dragIndex, 1);
    reordered.splice(hoverIndex, 0, draggedTask);
    replaceTasks(reordered);
    clearRoute();
  };

  const handlePersistReorder = async () => {
    await persistTaskOrder(tasksRef.current);
  };

  const runOptimization = async (mode: TransportMode) => {
    if (!requestOptions) return;

    const addressTasks = tasksRef.current.filter(taskHasAddress);
    const addresslessTasks = tasksRef.current.filter((t) => !taskHasAddress(t));

    if (addressTasks.length < 2) {
      toast.warning('Для оптимизации нужно минимум 2 задачи с адресом.');
      return;
    }

    const constraintIssues = validateStartEndConstraints(addressTasks, startTaskId, endTaskId);
    if (constraintIssues.length > 0) {
      for (const issue of constraintIssues) toast.warning(issue, { duration: 8000 });
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

      const orderedAddressTasks = reorderTasksByIds(addressTasks, result.orderedTaskIds);
      const orderedTasks = normalizeTasksOrder([...orderedAddressTasks, ...addresslessTasks]);
      replaceTasks(orderedTasks);

      const orderSaved = await persistTaskOrder(orderedTasks);
      if (!orderSaved) return;

      applyRoute({
        routeOptimized: true,
        optimizedInfo: {
          tasks: orderedAddressTasks,
          totalDistance: result.totalDistanceKm ?? 0,
          totalTravelTime: result.totalTravelTimeMin ?? 0,
          totalDuration:
            result.totalDurationMin ?? orderedAddressTasks.reduce((sum, t) => sum + t.duration, 0),
        },
        routeStops: result.stops ?? [],
        activeRouteId: result.id,
      });
      toast.success('Маршрут оптимизирован и сохранён');
    } catch (error) {
      reportError(error, 'Ошибка при оптимизации маршрута');
    } finally {
      setIsOptimizing(false);
    }
  };

  const handleOptimizeRoute = () => runOptimization(transportMode);

  const handleTransportModeChange = (mode: TransportMode) => {
    setTransportMode(mode);
    if (routeOptimized) void runOptimization(mode);
  };

  const { exportCSV, exportXLSX, exportRouteJSON } = useTaskExport(
    tasks,
    optimizedInfo,
    routeOptimized,
    activeRouteId,
  );

  const handleEditByTaskId = useCallback(
    (taskId: string) => {
      const task = tasks.find((t) => t.id === taskId);
      if (task) handleEditTask(task);
    },
    [tasks, handleEditTask],
  );

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

  const showSummary = !!(optimizedInfo && routeOptimized);
  const showPrecedence = tasks.length >= 2;
  const extraOffset = (showSummary ? 56 : 0) + (showPrecedence ? 84 : 0);
  const gridHeight = `calc(100vh - ${220 + extraOffset}px)`;

  return (
    <DndProvider backend={HTML5Backend}>
      <div className="container mx-auto px-3 sm:px-4 py-2 sm:py-3 max-w-full">
        <MainToolbar
          taskCount={tasks.length}
          isOptimizing={isOptimizing}
          isPersistingOrder={isPersistingOrder}
          hasWindowConflicts={hasWindowConflicts}
          onAddTask={handleAddTask}
          onOpenImport={() => setIsImportOpen(true)}
          onOptimize={handleOptimizeRoute}
          onOpenExport={() => setIsExportOpen(true)}
        />

        {showSummary && <OptimizedRouteSummary info={optimizedInfo!} />}

        {showPrecedence && (
          <PrecedenceConstraintsPanel
            tasks={tasks}
            precedences={precedences}
            onChange={setPrecedences}
          />
        )}

        <div
          className="grid grid-cols-1 gap-3 lg:gap-4 lg:grid-cols-[minmax(380px,1fr)_minmax(460px,1.1fr)] xl:grid-cols-[minmax(420px,1fr)_minmax(520px,1.1fr)]"
          style={{ ['--grid-h' as string]: gridHeight }}
        >
          <div className="min-w-0 h-[55vh] min-h-[360px] lg:h-[var(--grid-h)] lg:min-h-[420px]">
            <TaskList
              tasks={tasks}
              startTaskId={startTaskId || undefined}
              endTaskId={endTaskId || undefined}
              staticConflictIds={staticConflictIds}
              routeConflictIds={routeConflictIds}
              onEdit={handleEditTask}
              onDelete={handleDeleteTask}
              onDeleteAll={handleDeleteAllTasks}
              onToggleComplete={handleToggleComplete}
              onReorder={handleReorderTasks}
              onReorderEnd={handlePersistReorder}
              onSetRole={handleSetRole}
              onClone={handleCloneTask}
            />
          </div>

          <div className="min-w-0 h-[60vh] min-h-[400px] lg:h-[var(--grid-h)] lg:min-h-[420px]">
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
            onEditTask={handleEditByTaskId}
            onDeleteTask={(taskId) => void handleDeleteTask(taskId)}
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

        <TaskImport open={isImportOpen} onOpenChange={setIsImportOpen} onImport={handleImportTasks} />

        <ExportFormatDialog
          open={isExportOpen}
          onOpenChange={setIsExportOpen}
          onCSV={exportCSV}
          onXLSX={exportXLSX}
          onJSON={exportRouteJSON}
        />
      </div>
    </DndProvider>
  );
}
