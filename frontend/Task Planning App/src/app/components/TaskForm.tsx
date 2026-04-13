import { useState, useEffect, type FormEvent } from 'react';
import { Task } from '../types/task';
import { GeocodeSuggestion } from '../utils/routeOptimizer';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Label } from './ui/label';
import { Textarea } from './ui/textarea';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from './ui/dialog';
import { Loader2, MapPin, Flag, X } from 'lucide-react';
import { cn } from './ui/utils';

export type TaskRole = 'start' | 'end' | null;

interface TaskFormProps {
  task: Task | null;
  isOpen: boolean;
  isSaving?: boolean;
  initialRole?: TaskRole;
  canSetStart?: boolean;
  canSetEnd?: boolean;
  onClose: () => void;
  onSave: (task: Task, role: TaskRole) => Promise<void>;
  onGeocodeMultiple: (address: string) => Promise<GeocodeSuggestion[]>;
}

export function TaskForm({
  task,
  isOpen,
  isSaving = false,
  initialRole = null,
  canSetStart = true,
  canSetEnd = true,
  onClose,
  onSave,
  onGeocodeMultiple,
}: TaskFormProps) {
  const [title, setTitle] = useState('');
  const [address, setAddress] = useState('');
  const [duration, setDuration] = useState('30');
  const [timeWindowStart, setTimeWindowStart] = useState('');
  const [timeWindowEnd, setTimeWindowEnd] = useState('');
  const [role, setRole] = useState<TaskRole>(null);
  const [isGeocoding, setIsGeocoding] = useState(false);
  const [geocodeError, setGeocodeError] = useState<string | null>(null);
  const [windowError, setWindowError] = useState<string | null>(null);
  const [suggestions, setSuggestions] = useState<GeocodeSuggestion[]>([]);
  const [selectedSuggestion, setSelectedSuggestion] = useState<GeocodeSuggestion | null>(null);
  const [pendingTask, setPendingTask] = useState<Task | null>(null);

  useEffect(() => {
    if (task) {
      setTitle(task.title);
      setAddress(task.address);
      setDuration(task.duration.toString());
      setTimeWindowStart(task.timeWindowStart || '');
      setTimeWindowEnd(task.timeWindowEnd || '');
    } else {
      setTitle('');
      setAddress('');
      setDuration('30');
      setTimeWindowStart('');
      setTimeWindowEnd('');
    }
    setRole(initialRole);
    setSuggestions([]);
    setSelectedSuggestion(null);
    setGeocodeError(null);
    setWindowError(null);
    setPendingTask(null);
  }, [task, isOpen, initialRole]);

  const handleToggleRole = (toggled: 'start' | 'end') => {
    setRole((current) => (current === toggled ? null : toggled));
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setGeocodeError(null);
    setSuggestions([]);
    setSelectedSuggestion(null);
    setWindowError(null);

    if (timeWindowStart && timeWindowEnd) {
      const [sh, sm] = timeWindowStart.split(':').map(Number);
      const [eh, em] = timeWindowEnd.split(':').map(Number);
      const windowMins = (eh * 60 + em) - (sh * 60 + sm);
      const durationMins = parseInt(duration, 10) || 0;
      if (windowMins <= 0) {
        setWindowError('Конец окна должен быть позже начала.');
        return;
      }
      if (durationMins > windowMins) {
        setWindowError(`Длительность (${durationMins} мин) превышает размер окна (${windowMins} мин). Увеличьте окно или уменьшите длительность.`);
        return;
      }
    }

    // If address hasn't changed for an existing task, skip geocoding
    if (task && task.address === address && task.latitude !== undefined && task.longitude !== undefined) {
      const updatedTask: Task = {
        id: task.id,
        title,
        address,
        latitude: task.latitude,
        longitude: task.longitude,
        duration: parseInt(duration, 10) || 30,
        timeWindowStart: timeWindowStart,
        timeWindowEnd: timeWindowEnd,
        completed: task.completed,
        order: task.order,
      };
      setIsGeocoding(true);
      try {
        await onSave(updatedTask, role);
        onClose();
      } catch {
        // error shown via toast in parent
      } finally {
        setIsGeocoding(false);
      }
      return;
    }

    setIsGeocoding(true);
    try {
      let results: GeocodeSuggestion[];
      try {
        results = await onGeocodeMultiple(address);
      } catch (err) {
        const msg = err instanceof Error ? err.message : 'Ошибка геокодирования';
        setGeocodeError(msg);
        return;
      }

      if (results.length === 0) {
        setGeocodeError('Адрес не найден. Попробуйте уточнить запрос.');
        return;
      }

      const built: Task = {
        id: task?.id || crypto.randomUUID(),
        title,
        address,
        duration: parseInt(duration, 10) || 30,
        timeWindowStart: timeWindowStart,
        timeWindowEnd: timeWindowEnd,
        completed: task?.completed || false,
        order: task?.order,
      };
      setPendingTask(built);

      if (results.length === 1) {
        const taskWithCoords: Task = { ...built, latitude: results[0].lat, longitude: results[0].lng, address: results[0].displayName };
        setIsGeocoding(true);
        try {
          await onSave(taskWithCoords, role);
          onClose();
        } catch {
          // error shown via toast in parent
        } finally {
          setIsGeocoding(false);
        }
        return;
      }

      setSuggestions(results);
    } finally {
      setIsGeocoding(false);
    }
  };

  const handleSelectSuggestion = async (suggestion: GeocodeSuggestion) => {
    if (!pendingTask) return;
    setSelectedSuggestion(suggestion);
    const taskWithCoords: Task = {
      ...pendingTask,
      latitude: suggestion.lat,
      longitude: suggestion.lng,
      address: suggestion.displayName,
    };
    setIsGeocoding(true);
    try {
      await onSave(taskWithCoords, role);
      onClose();
    } catch {
      // error shown via toast in parent
    } finally {
      setIsGeocoding(false);
    }
  };

  const showSuggestions = suggestions.length > 0;

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-[500px]">
        {showSuggestions ? (
          <>
            <DialogHeader>
              <DialogTitle>Выберите адрес</DialogTitle>
              <DialogDescription>
                Найдено несколько адресов. Выберите наиболее подходящий.
              </DialogDescription>
            </DialogHeader>
            <div className="flex flex-col gap-2 py-4 max-h-80 overflow-y-auto">
              {suggestions.map((s, i) => (
                <button
                  key={i}
                  type="button"
                  disabled={isGeocoding}
                  onClick={() => handleSelectSuggestion(s)}
                  className="flex items-start gap-2 rounded-lg border p-3 text-left text-sm hover:bg-accent transition-colors disabled:opacity-50"
                >
                  <MapPin className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground" />
                  <span>{s.displayName}</span>
                </button>
              ))}
            </div>
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => { setSuggestions([]); setPendingTask(null); }}
                disabled={isGeocoding}
              >
                Назад
              </Button>
            </DialogFooter>
          </>
        ) : (
          <form onSubmit={handleSubmit}>
            <DialogHeader>
              <DialogTitle>{task ? 'Редактировать задачу' : 'Добавить задачу'}</DialogTitle>
              <DialogDescription>
                Укажите детали задачи. Адрес будет использован для построения маршрута.
              </DialogDescription>
            </DialogHeader>

            <div className="grid gap-4 py-4">
              <div className="space-y-2">
                <Label htmlFor="title">Название задачи *</Label>
                <Input
                  id="title"
                  placeholder="Например: Встреча с клиентом"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  required
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="address">Адрес *</Label>
                <Textarea
                  id="address"
                  placeholder="Например: г. Москва, ул. Тверская, д. 1"
                  value={address}
                  onChange={(e) => { setAddress(e.target.value); setGeocodeError(null); }}
                  rows={2}
                  required
                />
                {geocodeError && (
                  <p className="text-sm text-destructive">{geocodeError}</p>
                )}
              </div>

              <div className="space-y-2">
                <Label htmlFor="duration">Длительность (минуты) *</Label>
                <Input
                  id="duration"
                  type="number"
                  min="5"
                  step="5"
                  placeholder="30"
                  value={duration}
                  onChange={(e) => { setDuration(e.target.value); setWindowError(null); }}
                  required
                />
              </div>

              <div className="space-y-2">
                <Label>Временное окно <span className="text-muted-foreground font-normal">(необязательно)</span></Label>
                <p className="text-xs text-muted-foreground">
                  Период, в который должно <strong>начаться</strong> обслуживание. Алгоритм будет
                  стараться прибыть не раньше начала и успеть начать до конца&nbsp;окна. Окно должно
                  быть не короче длительности задачи.
                </p>
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="timeWindowStart">Начало окна</Label>
                    <div className="flex items-center gap-1">
                      <Input
                        id="timeWindowStart"
                        type="time"
                        value={timeWindowStart}
                        onChange={(e) => { setTimeWindowStart(e.target.value); setWindowError(null); }}
                        onBlur={(e) => { if (!e.target.value) { setTimeWindowStart(''); setWindowError(null); } }}
                        className="flex-1"
                      />
                      {timeWindowStart && (
                        <button
                          type="button"
                          onClick={() => { setTimeWindowStart(''); setWindowError(null); }}
                          className="flex-shrink-0 p-1 text-muted-foreground hover:text-destructive transition-colors rounded"
                          aria-label="Очистить начало окна"
                        >
                          <X className="h-3.5 w-3.5" />
                        </button>
                      )}
                    </div>
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="timeWindowEnd">Конец окна</Label>
                    <div className="flex items-center gap-1">
                      <Input
                        id="timeWindowEnd"
                        type="time"
                        value={timeWindowEnd}
                        onChange={(e) => { setTimeWindowEnd(e.target.value); setWindowError(null); }}
                        onBlur={(e) => { if (!e.target.value) { setTimeWindowEnd(''); setWindowError(null); } }}
                        className="flex-1"
                      />
                      {timeWindowEnd && (
                        <button
                          type="button"
                          onClick={() => { setTimeWindowEnd(''); setWindowError(null); }}
                          className="flex-shrink-0 p-1 text-muted-foreground hover:text-destructive transition-colors rounded"
                          aria-label="Очистить конец окна"
                        >
                          <X className="h-3.5 w-3.5" />
                        </button>
                      )}
                    </div>
                  </div>
                </div>
                {(() => {
                  if (!timeWindowStart || !timeWindowEnd) return null;
                  const [sh, sm] = timeWindowStart.split(':').map(Number);
                  const [eh, em] = timeWindowEnd.split(':').map(Number);
                  const windowMins = (eh * 60 + em) - (sh * 60 + sm);
                  const durationMins = parseInt(duration, 10) || 0;
                  if (windowMins <= 0) {
                    return (
                      <p className="text-xs text-destructive">
                        Конец окна должен быть позже начала.
                      </p>
                    );
                  }
                  if (durationMins > windowMins) {
                    return (
                      <p className="text-xs text-destructive">
                        Длительность ({durationMins} мин) превышает размер окна ({windowMins} мин) — задача не сможет уложиться в окно.
                      </p>
                    );
                  }
                  return null;
                })()}
                {windowError && <p className="text-sm text-destructive">{windowError}</p>}
              </div>

              <div className="space-y-2">
                <Label>Роль в маршруте <span className="text-muted-foreground font-normal">(необязательно)</span></Label>
                <div className="flex gap-2">
                  <button
                    type="button"
                    disabled={!canSetStart && role !== 'start'}
                    onClick={() => handleToggleRole('start')}
                    className={cn(
                      'inline-flex items-center gap-1.5 rounded border px-3 py-1.5 text-sm font-medium transition-colors',
                      role === 'start'
                        ? 'border-green-500 bg-green-50 text-green-700'
                        : 'border-gray-200 text-gray-600 hover:border-green-300 hover:bg-green-50 hover:text-green-700',
                      (!canSetStart && role !== 'start') && 'cursor-not-allowed opacity-40',
                    )}
                    title={!canSetStart && role !== 'start' ? 'Начальная точка уже задана для другой задачи' : undefined}
                  >
                    <Flag className="h-3.5 w-3.5" />
                    Начальная точка
                  </button>
                  <button
                    type="button"
                    disabled={!canSetEnd && role !== 'end'}
                    onClick={() => handleToggleRole('end')}
                    className={cn(
                      'inline-flex items-center gap-1.5 rounded border px-3 py-1.5 text-sm font-medium transition-colors',
                      role === 'end'
                        ? 'border-orange-400 bg-orange-50 text-orange-700'
                        : 'border-gray-200 text-gray-600 hover:border-orange-300 hover:bg-orange-50 hover:text-orange-700',
                      (!canSetEnd && role !== 'end') && 'cursor-not-allowed opacity-40',
                    )}
                    title={!canSetEnd && role !== 'end' ? 'Конечная точка уже задана для другой задачи' : undefined}
                  >
                    <Flag className="h-3.5 w-3.5" />
                    Конечная точка
                  </button>
                </div>
                {!canSetStart && role !== 'start' && (
                  <p className="text-xs text-muted-foreground">
                    Начальная точка уже задана. Снимите её с другой задачи, чтобы назначить эту.
                  </p>
                )}
                {!canSetEnd && role !== 'end' && (
                  <p className="text-xs text-muted-foreground">
                    Конечная точка уже задана. Снимите её с другой задачи, чтобы назначить эту.
                  </p>
                )}
              </div>
            </div>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={onClose}>
                Отмена
              </Button>
              <Button type="submit" disabled={isGeocoding || isSaving}>
                {(isGeocoding || isSaving) && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                {task ? 'Сохранить' : 'Добавить'}
              </Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
}
