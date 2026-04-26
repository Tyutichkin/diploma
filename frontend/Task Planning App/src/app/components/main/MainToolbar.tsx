import { Button } from '../ui/button';
import { Tooltip, TooltipContent, TooltipTrigger } from '../ui/tooltip';
import { Download, Loader2, Plus, Route, Upload } from 'lucide-react';

interface MainToolbarProps {
  taskCount: number;
  isOptimizing: boolean;
  isPersistingOrder: boolean;
  hasWindowConflicts: boolean;
  onAddTask: () => void;
  onOpenImport: () => void;
  onOptimize: () => void;
  onOpenExport: () => void;
}

export function MainToolbar({
  taskCount,
  isOptimizing,
  isPersistingOrder,
  hasWindowConflicts,
  onAddTask,
  onOpenImport,
  onOptimize,
  onOpenExport,
}: MainToolbarProps) {
  return (
    <div className="sticky top-[80px] z-30 mb-6 flex flex-wrap gap-3 bg-gray-50 py-2 border-b border-gray-100">
      <Tooltip>
        <TooltipTrigger asChild>
          <Button onClick={onAddTask}>
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
          <Button variant="outline" onClick={onOpenImport}>
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
          <span tabIndex={hasWindowConflicts ? 0 : undefined}>
            <Button
              onClick={onOptimize}
              disabled={isOptimizing || isPersistingOrder || taskCount < 2 || hasWindowConflicts}
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
          </span>
        </TooltipTrigger>
        <TooltipContent>
          <p>
            {hasWindowConflicts
              ? 'Исправьте конфликты временных окон перед оптимизацией'
              : 'Построить маршрут и сохранить его в бэкенде'}
          </p>
        </TooltipContent>
      </Tooltip>

      <Tooltip>
        <TooltipTrigger asChild>
          <Button onClick={onOpenExport} disabled={taskCount === 0} variant="outline">
            <Download className="mr-2 h-4 w-4" />
            Экспорт
          </Button>
        </TooltipTrigger>
        <TooltipContent>
          <p>Скачать задачи в выбранном формате</p>
        </TooltipContent>
      </Tooltip>
    </div>
  );
}
