import { SavedRouteSummary } from '../../types/route';
import { Button } from '../ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '../ui/card';
import { Check, History, Loader2, Pencil, Trash2, X } from 'lucide-react';

interface SavedRoutesPanelProps {
  routes: SavedRouteSummary[];
  activeRouteId: string | null;
  loadingRouteId: string | null;
  taskCount: number;
  editingRouteId: string | null;
  editingRouteName: string;
  onApply: (routeId: string) => void;
  onDelete: (routeId: string) => void;
  onDeleteAll: () => void;
  onStartRename: (routeId: string, currentName: string | undefined, source: string) => void;
  onChangeRename: (name: string) => void;
  onConfirmRename: () => void;
  onCancelRename: () => void;
}

function defaultRouteName(source: string) {
  return source === 'optimized' ? 'Оптимизированный маршрут' : 'Маршрут';
}

export function SavedRoutesPanel({
  routes,
  activeRouteId,
  loadingRouteId,
  taskCount,
  editingRouteId,
  editingRouteName,
  onApply,
  onDelete,
  onDeleteAll,
  onStartRename,
  onChangeRename,
  onConfirmRename,
  onCancelRename,
}: SavedRoutesPanelProps) {
  return (
    <Card className="mb-6">
      <CardHeader className="flex flex-row items-center justify-between gap-4 space-y-0">
        <CardTitle className="flex items-center gap-2">
          <History className="h-5 w-5" />
          Сохранённые маршруты
          <span className="text-sm font-normal text-gray-500">({routes.length})</span>
        </CardTitle>
        {routes.length > 0 && (
          <Button
            variant="outline"
            size="sm"
            className="text-red-600 hover:text-red-700 hover:border-red-300"
            onClick={onDeleteAll}
          >
            <Trash2 className="mr-1.5 h-3.5 w-3.5" />
            Очистить всё
          </Button>
        )}
      </CardHeader>
      <CardContent>
        {routes.length === 0 ? (
          <p className="text-sm text-gray-500">Маршруты ещё не сохранялись.</p>
        ) : (
          <div className="max-h-64 overflow-y-auto pr-1">
            <div className="space-y-3">
              {routes.map((route) => (
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
                          onChange={(e) => onChangeRename(e.target.value)}
                          onKeyDown={(e) => {
                            if (e.key === 'Enter') onConfirmRename();
                            if (e.key === 'Escape') onCancelRename();
                          }}
                        />
                        <button
                          className="text-green-600 hover:text-green-700"
                          onClick={onConfirmRename}
                        >
                          <Check className="h-4 w-4" />
                        </button>
                        <button
                          className="text-gray-400 hover:text-gray-600"
                          onClick={onCancelRename}
                        >
                          <X className="h-4 w-4" />
                        </button>
                      </div>
                    ) : (
                      <div className="flex items-center gap-1.5">
                        <div className="font-medium text-gray-900 truncate">
                          {route.name ?? defaultRouteName(route.source)}
                        </div>
                        {route.source === 'optimized' && (
                          <button
                            className="flex-shrink-0 text-gray-400 hover:text-gray-600"
                            onClick={() => onStartRename(route.id, route.name, route.source)}
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
                      disabled={loadingRouteId === route.id || taskCount === 0}
                      onClick={() => onApply(route.id)}
                    >
                      {loadingRouteId === route.id && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                      Применить
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="text-red-400 hover:text-red-600 hover:bg-red-50 px-2"
                      onClick={() => onDelete(route.id)}
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
  );
}
