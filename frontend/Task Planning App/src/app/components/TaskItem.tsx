import { Task } from '../types/task';
import { useDrag, useDrop } from 'react-dnd';
import { Card, CardContent } from './ui/card';
import { Button } from './ui/button';
import { GripVertical, Clock, MapPin, Edit2, Trash2, CheckCircle2 } from 'lucide-react';
import { cn } from './ui/utils';

interface TaskItemProps {
  task: Task;
  index: number;
  onEdit: (task: Task) => void;
  onDelete: (id: string) => void | Promise<void>;
  onToggleComplete: (id: string) => void;
  onMove: (dragIndex: number, hoverIndex: number) => void;
  onMoveEnd: () => void | Promise<void>;
}

const ITEM_TYPE = 'TASK';

export function TaskItem({
  task,
  index,
  onEdit,
  onDelete,
  onToggleComplete,
  onMove,
  onMoveEnd,
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

  return (
    <div ref={(node) => preview(drop(node))} style={{ opacity: isDragging ? 0.5 : 1 }}>
      <Card className={cn(
        "mb-2 transition-all hover:shadow-md",
        task.completed && "opacity-60 bg-gray-50"
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
                <h3 className={cn(
                  "font-medium",
                  task.completed && "line-through text-gray-500"
                )}>
                  {task.title}
                </h3>
                <div className="flex gap-1 flex-shrink-0">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => onToggleComplete(task.id)}
                    className="h-8 w-8 p-0"
                  >
                    <CheckCircle2 className={cn(
                      "h-4 w-4",
                      task.completed ? "text-green-500" : "text-gray-300"
                    )} />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => onEdit(task)}
                    className="h-8 w-8 p-0"
                  >
                    <Edit2 className="h-4 w-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => onDelete(task.id)}
                    className="h-8 w-8 p-0 text-red-500 hover:text-red-700"
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
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
                  {task.timeWindowStart && task.timeWindowEnd && (
                    <span className="text-xs bg-blue-100 text-blue-700 px-2 py-1 rounded">
                      {task.timeWindowStart} - {task.timeWindowEnd}
                    </span>
                  )}
                </div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
