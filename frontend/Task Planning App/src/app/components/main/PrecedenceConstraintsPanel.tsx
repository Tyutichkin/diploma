import { Task } from '../../types/task';
import { Button } from '../ui/button';
import { Card, CardContent } from '../ui/card';
import { Tooltip, TooltipContent, TooltipTrigger } from '../ui/tooltip';
import { Info, Plus, X } from 'lucide-react';

export interface PrecedencePair {
  beforeId: string;
  afterId: string;
}

interface PrecedenceConstraintsPanelProps {
  tasks: Task[];
  precedences: PrecedencePair[];
  onChange: (next: PrecedencePair[]) => void;
}

export function PrecedenceConstraintsPanel({
  tasks,
  precedences,
  onChange,
}: PrecedenceConstraintsPanelProps) {
  const updatePair = (idx: number, patch: Partial<PrecedencePair>) => {
    onChange(precedences.map((p, i) => (i === idx ? { ...p, ...patch } : p)));
  };

  return (
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
                  <p>
                    Первая задача выполняется раньше второй.
                    <br />
                    Ограничения учитываются при оптимизации.
                  </p>
                </TooltipContent>
              </Tooltip>
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => onChange([...precedences, { beforeId: '', afterId: '' }])}
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
                  onChange={(e) => updatePair(idx, { beforeId: e.target.value })}
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
                  onChange={(e) => updatePair(idx, { afterId: e.target.value })}
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
                  onClick={() => onChange(precedences.filter((_, i) => i !== idx))}
                >
                  <X className="h-4 w-4" />
                </button>
              </div>
            ))}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
