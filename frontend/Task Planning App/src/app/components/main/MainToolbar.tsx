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
    <div className="sticky top-[56px] sm:top-[72px] z-30 mb-3 flex flex-wrap items-center gap-2 bg-gray-50 py-1.5 border-b border-gray-100">
      <Tooltip>
        <TooltipTrigger asChild>
          <Button onClick={onAddTask} size="sm" className="h-9">
            <Plus className="h-4 w-4 sm:mr-2" />
            <span className="hidden sm:inline">Добавить задачу</span>
            <span className="sr-only sm:hidden">Добавить задачу</span>
          </Button>
        </TooltipTrigger>
        <TooltipContent>
          <p>Создать новую задачу с адресом и временными рамками</p>
        </TooltipContent>
      </Tooltip>

      <Tooltip>
        <TooltipTrigger asChild>
          <span tabIndex={hasWindowConflicts ? 0 : undefined}>
            <Button
              onClick={onOptimize}
              disabled={isOptimizing || isPersistingOrder || taskCount < 2 || hasWindowConflicts}
              variant="default"
              size="sm"
              className="h-9 bg-blue-600 hover:bg-blue-700"
            >
              {isOptimizing ? (
                <Loader2 className="h-4 w-4 sm:mr-2 animate-spin" />
              ) : (
                <Route className="h-4 w-4 sm:mr-2" />
              )}
              <span className="hidden sm:inline">Оптимизировать маршрут</span>
              <span className="sr-only sm:hidden">Оптимизировать маршрут</span>
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

      <span className="hidden sm:inline-block w-px self-stretch bg-gray-200 mx-1" aria-hidden />

      <Tooltip>
        <TooltipTrigger asChild>
          <Button variant="outline" onClick={onOpenImport} size="sm" className="h-9">
            <Upload className="h-4 w-4 sm:mr-2" />
            <span className="hidden sm:inline">Импорт из файла</span>
            <span className="sr-only sm:hidden">Импорт из файла</span>
          </Button>
        </TooltipTrigger>
        <TooltipContent>
          <p>Загрузить задачи из CSV или Excel-файла</p>
        </TooltipContent>
      </Tooltip>

      <Tooltip>
        <TooltipTrigger asChild>
          <Button onClick={onOpenExport} disabled={taskCount === 0} variant="outline" size="sm" className="h-9">
            <Download className="h-4 w-4 sm:mr-2" />
            <span className="hidden sm:inline">Экспорт</span>
            <span className="sr-only sm:hidden">Экспорт</span>
          </Button>
        </TooltipTrigger>
        <TooltipContent>
          <p>Скачать задачи в выбранном формате</p>
        </TooltipContent>
      </Tooltip>
    </div>
  );
}
