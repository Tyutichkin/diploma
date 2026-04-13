import { Task } from '../types/task';
import { TaskItem } from './TaskItem';
import { Card, CardContent, CardHeader, CardTitle } from './ui/card';
import { ScrollArea } from './ui/scroll-area';

interface TaskListProps {
  tasks: Task[];
  startTaskId?: string;
  endTaskId?: string;
  onEdit: (task: Task) => void;
  onDelete: (id: string) => void | Promise<void>;
  onToggleComplete: (id: string) => void;
  onReorder: (dragIndex: number, hoverIndex: number) => void;
  onReorderEnd: () => void | Promise<void>;
  onSetRole: (id: string, role: 'start' | 'end' | null) => void;
}

export function TaskList({
  tasks,
  startTaskId,
  endTaskId,
  onEdit,
  onDelete,
  onToggleComplete,
  onReorder,
  onReorderEnd,
  onSetRole,
}: TaskListProps) {
  return (
    <Card className="h-full flex flex-col">
      <CardHeader className="flex-shrink-0">
        <CardTitle>Список задач ({tasks.length})</CardTitle>
      </CardHeader>
      <CardContent className="flex-1 overflow-hidden p-4">
        <ScrollArea className="h-full pr-4">
          {tasks.length === 0 ? (
            <div className="text-center text-gray-500 py-8">
              Нет задач. Добавьте первую задачу.
            </div>
          ) : (
            <div>
              {tasks.map((task, index) => (
                <TaskItem
                  key={task.id}
                  task={task}
                  index={index}
                  isStart={task.id === startTaskId}
                  isEnd={task.id === endTaskId}
                  canSetStart={!startTaskId || task.id === startTaskId}
                  canSetEnd={!endTaskId || task.id === endTaskId}
                  onEdit={onEdit}
                  onDelete={onDelete}
                  onToggleComplete={onToggleComplete}
                  onMove={onReorder}
                  onMoveEnd={onReorderEnd}
                  onSetRole={onSetRole}
                />
              ))}
            </div>
          )}
        </ScrollArea>
      </CardContent>
    </Card>
  );
}
