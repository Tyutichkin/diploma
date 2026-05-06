import { useMemo } from 'react';
import { Task, taskHasAddress, getWindowConflictIds } from '../types/task';
import { TaskItem } from './TaskItem';
import { Button } from './ui/button';
import { Card, CardContent, CardHeader, CardTitle } from './ui/card';
import { ScrollArea } from './ui/scroll-area';
import { MapPin, MapPinOff, Trash2 } from 'lucide-react';

interface TaskListProps {
  tasks: Task[];
  startTaskId?: string;
  endTaskId?: string;
  // Если не передан — TaskList посчитает сам (для обратной совместимости с тестами).
  staticConflictIds?: Set<string>;
  routeConflictIds?: Set<string>;
  onEdit: (task: Task) => void;
  onDelete: (id: string) => void | Promise<void>;
  onDeleteAll?: () => void | Promise<void>;
  onToggleComplete: (id: string) => void;
  onReorder: (dragIndex: number, hoverIndex: number) => void;
  onReorderEnd: () => void | Promise<void>;
  onSetRole: (id: string, role: 'start' | 'end' | null) => void;
  onClone?: (task: Task) => void;
}

export function TaskList({
  tasks,
  startTaskId,
  endTaskId,
  staticConflictIds,
  routeConflictIds,
  onEdit,
  onDelete,
  onDeleteAll,
  onToggleComplete,
  onReorder,
  onReorderEnd,
  onSetRole,
  onClone,
}: TaskListProps) {
  const handleDeleteAll = () => {
    if (!onDeleteAll) return;
    if (!window.confirm(`Удалить все задачи (${tasks.length})? Действие необратимо.`)) return;
    void onDeleteAll();
  };
  const fallbackConflictIds = useMemo(
    () => (staticConflictIds ? null : getWindowConflictIds(tasks)),
    [staticConflictIds, tasks],
  );
  const baseConflictIds = staticConflictIds ?? fallbackConflictIds!;

  const conflictIds = useMemo(
    () =>
      routeConflictIds
        ? new Set([...baseConflictIds, ...routeConflictIds])
        : baseConflictIds,
    [baseConflictIds, routeConflictIds],
  );

  const withAddress = tasks.filter(taskHasAddress);
  const withoutAddress = tasks.filter((t) => !taskHasAddress(t));
  const noClone = onClone ?? (() => {});

  return (
    <Card className="h-full flex flex-col">
      <CardHeader className="flex-shrink-0 flex flex-row items-center justify-between gap-4 space-y-0">
        <CardTitle>Список задач ({tasks.length})</CardTitle>
        {onDeleteAll && tasks.length > 0 && (
          <Button
            variant="outline"
            size="sm"
            className="text-red-600 hover:text-red-700 hover:border-red-300"
            onClick={handleDeleteAll}
          >
            <Trash2 className="mr-1.5 h-3.5 w-3.5" />
            Очистить всё
          </Button>
        )}
      </CardHeader>
      <CardContent className="flex-1 overflow-hidden p-4">
        <ScrollArea className="h-full pr-4">
          {tasks.length === 0 ? (
            <div className="text-center text-gray-500 py-8">
              Нет задач. Добавьте первую задачу.
            </div>
          ) : (
            <div>
              {withAddress.length > 0 && (
                <div>
                  <div className="flex items-center gap-1.5 mb-2 text-xs font-semibold text-muted-foreground uppercase tracking-wide">
                    <MapPin className="h-3.5 w-3.5" />
                    С адресом ({withAddress.length})
                  </div>
                  {withAddress.map((task, mapIdx) => {
                    const globalIndex = tasks.findIndex((t) => t.id === task.id);
                    return (
                      <TaskItem
                        key={task.id}
                        task={task}
                        index={globalIndex}
                        mapNumber={mapIdx + 1}
                        isAddressless={false}
                        isStart={task.id === startTaskId}
                        isEnd={task.id === endTaskId}
                        canSetStart={!startTaskId || task.id === startTaskId}
                        canSetEnd={!endTaskId || task.id === endTaskId}
                        hasWindowConflict={conflictIds.has(task.id)}
                        onEdit={onEdit}
                        onDelete={onDelete}
                        onToggleComplete={onToggleComplete}
                        onMove={onReorder}
                        onMoveEnd={onReorderEnd}
                        onSetRole={onSetRole}
                        onClone={noClone}
                      />
                    );
                  })}
                </div>
              )}

              {withoutAddress.length > 0 && (
                <div className={withAddress.length > 0 ? 'mt-4' : ''}>
                  <div className="flex items-center gap-1.5 mb-2 text-xs font-semibold text-muted-foreground uppercase tracking-wide">
                    <MapPinOff className="h-3.5 w-3.5" />
                    Без адреса ({withoutAddress.length})
                  </div>
                  {withoutAddress.map((task) => {
                    const globalIndex = tasks.findIndex((t) => t.id === task.id);
                    return (
                      <TaskItem
                        key={task.id}
                        task={task}
                        index={globalIndex}
                        isAddressless={true}
                        isStart={false}
                        isEnd={false}
                        canSetStart={false}
                        canSetEnd={false}
                        hasWindowConflict={conflictIds.has(task.id)}
                        onEdit={onEdit}
                        onDelete={onDelete}
                        onToggleComplete={onToggleComplete}
                        onMove={onReorder}
                        onMoveEnd={onReorderEnd}
                        onSetRole={onSetRole}
                        onClone={noClone}
                      />
                    );
                  })}
                </div>
              )}
            </div>
          )}
        </ScrollArea>
      </CardContent>
    </Card>
  );
}
