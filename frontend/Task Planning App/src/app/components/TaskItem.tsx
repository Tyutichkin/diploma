import { Task } from '../types/task';
import { useDrag, useDrop } from 'react-dnd';
import { Card, CardContent } from './ui/card';
import { Button, buttonVariants } from './ui/button';
import { GripVertical, Clock, MapPin, Edit2, Trash2, CheckCircle2, Flag, MoreHorizontal } from 'lucide-react';
import { cn } from './ui/utils';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from './ui/dropdown-menu';
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from './ui/context-menu';

interface TaskItemProps {
  task: Task;
  index: number;
  isStart?: boolean;
  isEnd?: boolean;
  canSetStart?: boolean;
  canSetEnd?: boolean;
  onEdit: (task: Task) => void;
  onDelete: (id: string) => void | Promise<void>;
  onToggleComplete: (id: string) => void;
  onMove: (dragIndex: number, hoverIndex: number) => void;
  onMoveEnd: () => void | Promise<void>;
  onSetRole?: (id: string, role: 'start' | 'end' | null) => void;
}

const ITEM_TYPE = 'TASK';

export function TaskItem({
  task,
  index,
  isStart,
  isEnd,
  canSetStart,
  canSetEnd,
  onEdit,
  onDelete,
  onToggleComplete,
  onMove,
  onMoveEnd,
  onSetRole,
}: TaskItemProps) {
  const [{ isDragging }, drag, preview] = useDrag({
    type: ITEM_TYPE,
    item: { index },
    end: () => {
      void onMoveEnd();
    },
    collect: (monitor) => ({
      isDragging: monitor.isDragging(),
    }),
  });

  const [, drop] = useDrop({
    accept: ITEM_TYPE,
    hover: (item: { index: number }) => {
      if (item.index !== index) {
        onMove(item.index, index);
        item.index = index;
      }
    },
  });

  const startLabel = isStart ? 'Убрать как начальную точку' : 'Назначить начальной точкой';
  const endLabel = isEnd ? 'Убрать как конечную точку' : 'Назначить конечной точкой';
  const startDisabled = !isStart && !canSetStart;
  const endDisabled = !isEnd && !canSetEnd;

  const handleToggleStart = () => onSetRole?.(task.id, isStart ? null : 'start');
  const handleToggleEnd = () => onSetRole?.(task.id, isEnd ? null : 'end');

  const RoleMenuItems = ({ Item, Sep }: {
    Item: React.ComponentType<{ disabled?: boolean; onClick?: () => void; className?: string; children: React.ReactNode }>;
    Sep: React.ComponentType;
  }) => (
    <>
      <Item disabled={startDisabled} onClick={handleToggleStart}>
        <Flag className={cn('mr-2 h-3.5 w-3.5', isStart ? 'text-green-600' : 'text-gray-400')} />
        {startLabel}
      </Item>
      <Item disabled={endDisabled} onClick={handleToggleEnd}>
        <Flag className={cn('mr-2 h-3.5 w-3.5', isEnd ? 'text-orange-500' : 'text-gray-400')} />
        {endLabel}
      </Item>
      <Sep />
      <Item onClick={() => onEdit(task)}>
        <Edit2 className="mr-2 h-3.5 w-3.5 text-gray-400" />
        Редактировать
      </Item>
      <Item onClick={() => void onDelete(task.id)} className="text-destructive focus:text-destructive">
        <Trash2 className="mr-2 h-3.5 w-3.5" />
        Удалить
      </Item>
    </>
  );

  return (
    <ContextMenu>
      <ContextMenuTrigger asChild>
        <div ref={(node) => preview(drop(node))} style={{ opacity: isDragging ? 0.5 : 1 }}>
          <Card className={cn(
            'mb-2 transition-all hover:shadow-md',
            task.completed && 'opacity-60 bg-gray-50',
          )}>
            <CardContent className="p-4">
              <div className="flex items-start gap-3">
                <div
                  ref={drag}
                  className="cursor-move mt-1 text-gray-400 hover:text-gray-600"
                >
                  <GripVertical className="h-5 w-5" />
                </div>

                <div className="flex-1 min-w-0">
                  <div className="flex items-start justify-between gap-2">
                    <div className="flex items-center gap-2 flex-wrap min-w-0">
                      {isStart && (
                        <span className="inline-flex items-center gap-1 rounded bg-green-100 px-1.5 py-0.5 text-xs font-medium text-green-700 flex-shrink-0">
                          <Flag className="h-3 w-3" />
                          Старт
                        </span>
                      )}
                      {isEnd && (
                        <span className="inline-flex items-center gap-1 rounded bg-orange-100 px-1.5 py-0.5 text-xs font-medium text-orange-700 flex-shrink-0">
                          <Flag className="h-3 w-3" />
                          Финиш
                        </span>
                      )}
                      <h3 className={cn(
                        'font-medium',
                        task.completed && 'line-through text-gray-500',
                      )}>
                        {task.title}
                      </h3>
                    </div>
                    <div className="flex gap-1 flex-shrink-0">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => onToggleComplete(task.id)}
                        className="h-8 w-8 p-0"
                        title="Отметить как выполненную"
                      >
                        <CheckCircle2 className={cn(
                          'h-4 w-4',
                          task.completed ? 'text-green-500' : 'text-gray-300',
                        )} />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => onEdit(task)}
                        className="h-8 w-8 p-0"
                        title="Редактировать"
                      >
                        <Edit2 className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => void onDelete(task.id)}
                        className="h-8 w-8 p-0 text-red-500 hover:text-red-700"
                        title="Удалить"
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                      <DropdownMenu>
                        <DropdownMenuTrigger
                          className={cn(
                            buttonVariants({ variant: 'ghost', size: 'sm' }),
                            'h-8 w-8 p-0 text-gray-400 hover:text-gray-600',
                          )}
                          title="Дополнительные действия"
                          onPointerDown={(e) => e.stopPropagation()}
                        >
                          <MoreHorizontal className="h-4 w-4" />
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <RoleMenuItems
                            Item={DropdownMenuItem}
                            Sep={DropdownMenuSeparator}
                          />
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </div>
                  </div>

                  <div className="mt-2 space-y-1 text-sm text-gray-600">
                    <div className="flex items-center gap-2">
                      <MapPin className="h-4 w-4 flex-shrink-0" />
                      <span className="truncate">{task.address}</span>
                    </div>
                    <div className="flex items-center gap-4">
                      <div className="flex items-center gap-2">
                        <Clock className="h-4 w-4" />
                        <span>{task.duration} мин</span>
                      </div>
                      {(task.timeWindowStart || task.timeWindowEnd) && (
                        <span className="text-xs bg-blue-100 text-blue-700 px-2 py-1 rounded">
                          {task.timeWindowStart || '—'}&nbsp;—&nbsp;{task.timeWindowEnd || '—'}
                        </span>
                      )}
                    </div>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      </ContextMenuTrigger>
      <ContextMenuContent>
        <RoleMenuItems
          Item={ContextMenuItem}
          Sep={ContextMenuSeparator}
        />
      </ContextMenuContent>
    </ContextMenu>
  );
}
