import { Task } from '../types/task';
import { useDrag, useDrop } from 'react-dnd';
import { Card, CardContent } from './ui/card';
import { Button, buttonVariants } from './ui/button';
import { GripVertical, Clock, MapPin, Edit2, Trash2, CheckCircle2, Flag, MoreHorizontal, Copy, AlertTriangle } from 'lucide-react';
import { cn } from './ui/utils';
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from './ui/tooltip';
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
  mapNumber?: number;
  isAddressless?: boolean;
  isStart?: boolean;
  isEnd?: boolean;
  canSetStart?: boolean;
  canSetEnd?: boolean;
  hasWindowConflict?: boolean;
  onEdit: (task: Task) => void;
  onDelete: (id: string) => void | Promise<void>;
  onToggleComplete: (id: string) => void;
  onMove: (dragIndex: number, hoverIndex: number) => void;
  onMoveEnd: () => void | Promise<void>;
  onSetRole?: (id: string, role: 'start' | 'end' | null) => void;
  onClone?: (task: Task) => void;
}

// Раздельные типы DnD предотвращают перетаскивание между группами.
const ITEM_TYPE_WITH_ADDR = 'TASK_WITH_ADDR';
const ITEM_TYPE_NO_ADDR = 'TASK_NO_ADDR';

export function TaskItem({
  task,
  index,
  mapNumber,
  isAddressless = false,
  isStart,
  isEnd,
  canSetStart,
  canSetEnd,
  hasWindowConflict = false,
  onEdit,
  onDelete,
  onToggleComplete,
  onMove,
  onMoveEnd,
  onSetRole,
  onClone,
}: TaskItemProps) {
  const itemType = isAddressless ? ITEM_TYPE_NO_ADDR : ITEM_TYPE_WITH_ADDR;

  const [{ isDragging }, drag, preview] = useDrag({
    type: itemType,
    item: { index },
    end: () => {
      void onMoveEnd();
    },
    collect: (monitor) => ({
      isDragging: monitor.isDragging(),
    }),
  });

  const [, drop] = useDrop({
    accept: itemType,
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

  const requestDelete = () => {
    if (!window.confirm(`Удалить задачу «${task.title}»?`)) return;
    void onDelete(task.id);
  };

  const RoleMenuItems = ({ Item, Sep }: {
    Item: React.ComponentType<{ disabled?: boolean; onClick?: () => void; className?: string; children: React.ReactNode }>;
    Sep: React.ComponentType;
  }) => (
    <>
      {!isAddressless && (
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
        </>
      )}
      <Item onClick={() => onEdit(task)}>
        <Edit2 className="mr-2 h-3.5 w-3.5 text-gray-400" />
        Редактировать
      </Item>
      <Item onClick={() => onClone?.(task)}>
        <Copy className="mr-2 h-3.5 w-3.5 text-gray-400" />
        Клонировать
      </Item>
      <Item onClick={requestDelete} className="text-destructive focus:text-destructive">
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
            hasWindowConflict && 'border-red-400 bg-red-50/50',
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
                      {mapNumber !== undefined && (
                        <span className="inline-flex items-center justify-center rounded-full bg-blue-500 text-white text-xs font-bold w-5 h-5 flex-shrink-0">
                          {mapNumber}
                        </span>
                      )}
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
                        onClick={requestDelete}
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
                    {task.address ? (
                      <div className="flex items-center gap-2">
                        <MapPin className="h-4 w-4 flex-shrink-0" />
                        <span className="truncate">{task.address}</span>
                      </div>
                    ) : (
                      <div className="flex items-center gap-2 text-muted-foreground italic">
                        <MapPin className="h-4 w-4 flex-shrink-0 opacity-40" />
                        <span className="text-xs">Без адреса</span>
                      </div>
                    )}
                    <div className="flex items-center gap-4">
                      <div className="flex items-center gap-2">
                        <Clock className="h-4 w-4" />
                        <span>{task.duration != null ? `${task.duration} мин` : 'мгновенная'}</span>
                      </div>
                      {(task.windowStartDate || task.windowStartTime || task.windowEndDate || task.windowEndTime) && (
                        <span className={cn(
                          'text-xs px-2 py-1 rounded',
                          hasWindowConflict ? 'bg-red-100 text-red-700' : 'bg-blue-100 text-blue-700',
                        )}>
                          {[task.windowStartDate, task.windowStartTime].filter(Boolean).join(' ') || '—'}
                          &nbsp;—&nbsp;
                          {[task.windowEndDate, task.windowEndTime].filter(Boolean).join(' ') || '—'}
                        </span>
                      )}
                      {hasWindowConflict && (
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <AlertTriangle className="h-4 w-4 text-red-500 flex-shrink-0 cursor-help" />
                          </TooltipTrigger>
                          <TooltipContent>
                            Задача не укладывается в временное окно. Отредактируйте окно или длительность.
                          </TooltipContent>
                        </Tooltip>
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
