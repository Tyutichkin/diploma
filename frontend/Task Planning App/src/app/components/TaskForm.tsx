import { useState, useEffect, useRef, type FormEvent } from 'react';
import { Task } from '../types/task';
import { GeocodeSuggestion, suggestAddresses } from '../utils/routeOptimizer';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Label } from './ui/label';
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
  const [duration, setDuration] = useState('');
  const [windowStartDate, setWindowStartDate] = useState('');
  const [windowStartTime, setWindowStartTime] = useState('');
  const [windowEndDate, setWindowEndDate] = useState('');
  const [windowEndTime, setWindowEndTime] = useState('');
  const [role, setRole] = useState<TaskRole>(null);
  const [isGeocoding, setIsGeocoding] = useState(false);
  const [geocodeError, setGeocodeError] = useState<string | null>(null);
  const [windowError, setWindowError] = useState<string | null>(null);

  // Автодополнение адреса
  const [autoSuggestions, setAutoSuggestions] = useState<GeocodeSuggestion[]>([]);
  const [autoSuggestOpen, setAutoSuggestOpen] = useState(false);
  // Координаты, полученные при выборе из дропдауна — не нужно геокодировать повторно
  const [preselectedCoords, setPreselectedCoords] = useState<{ lat: number; lng: number; address: string } | null>(null);
  const suggestTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (task) {
      setTitle(task.title);
      setAddress(task.address ?? '');
      setDuration(task.duration != null ? task.duration.toString() : '');
      setWindowStartDate(task.windowStartDate || '');
      setWindowStartTime(task.windowStartTime || '');
      setWindowEndDate(task.windowEndDate || '');
      setWindowEndTime(task.windowEndTime || '');
    } else {
      setTitle('');
      setAddress('');
      setDuration('');
      setWindowStartDate('');
      setWindowStartTime('');
      setWindowEndDate('');
      setWindowEndTime('');
    }
    setRole(initialRole);
    setGeocodeError(null);
    setWindowError(null);
    setAutoSuggestions([]);
    setAutoSuggestOpen(false);
    setPreselectedCoords(null);
    if (suggestTimerRef.current) clearTimeout(suggestTimerRef.current);
  }, [task, isOpen, initialRole]);

  const handleToggleRole = (toggled: 'start' | 'end') => {
    setRole((current) => (current === toggled ? null : toggled));
  };

  const handleAddressChange = (value: string) => {
    setAddress(value);
    setGeocodeError(null);
    if (preselectedCoords && preselectedCoords.address !== value) {
      setPreselectedCoords(null);
    }
    if (suggestTimerRef.current) clearTimeout(suggestTimerRef.current);
    if (!value.trim()) {
      setAutoSuggestions([]);
      setAutoSuggestOpen(false);
      return;
    }
    suggestTimerRef.current = setTimeout(async () => {
      try {
        const results = await suggestAddresses(value);
        setAutoSuggestions(results);
        setAutoSuggestOpen(results.length > 0);
      } catch {
        // тихо игнорируем — автодополнение не критично
      }
    }, 500);
  };

  const handleSelectAutoSuggestion = (suggestion: GeocodeSuggestion) => {
    setAutoSuggestOpen(false);
    setAutoSuggestions([]);
    setAddress(suggestion.displayName);
    setGeocodeError(null);
    setPreselectedCoords({ lat: suggestion.lat, lng: suggestion.lng, address: suggestion.displayName });
  };

  /** Вычисляет размер окна в минутах, если обе границы заданы. */
  const computeWindowMins = (): number | null => {
    if (!windowStartDate && !windowStartTime && !windowEndDate && !windowEndTime) return null;
    // Нужна хотя бы одна дата для сравнения
    const sDate = windowStartDate || windowEndDate;
    const eDate = windowEndDate || windowStartDate;
    if (!sDate || !eDate) return null;
    const sTime = windowStartTime || '00:00';
    const eTime = windowEndTime || '23:59';
    const start = new Date(`${sDate}T${sTime}`);
    const end = new Date(`${eDate}T${eTime}`);
    if (isNaN(start.getTime()) || isNaN(end.getTime())) return null;
    return (end.getTime() - start.getTime()) / 60000;
  };

  const buildBaseTask = (): Task => ({
    id: task?.id || crypto.randomUUID(),
    title,
    address,
    duration: duration.trim() !== '' ? parseInt(duration, 10) : undefined,
    windowStartDate: windowStartDate || undefined,
    windowStartTime: windowStartTime || undefined,
    windowEndDate: windowEndDate || undefined,
    windowEndTime: windowEndTime || undefined,
    completed: task?.completed || false,
    order: task?.order,
  });

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setGeocodeError(null);
    setWindowError(null);

    const windowMins = computeWindowMins();
    if (windowMins !== null && windowMins <= 0) {
      setWindowError('Конец окна должен быть позже начала.');
      return;
    }
    if (windowMins !== null) {
      const durationMins = duration.trim() !== '' ? parseInt(duration, 10) : 0;
      if (durationMins > 0 && durationMins > windowMins) {
        setWindowError(`Длительность (${durationMins} мин) превышает размер окна (${Math.round(windowMins)} мин). Увеличьте окно или уменьшите длительность.`);
        return;
      }
    }

    // Задача без адреса
    if (!address.trim()) {
      const t: Task = { ...buildBaseTask(), address: undefined, latitude: undefined, longitude: undefined };
      setIsGeocoding(true);
      try { await onSave(t, role); onClose(); } catch { /* toast в parent */ } finally { setIsGeocoding(false); }
      return;
    }

    // Адрес выбран из автодополнения — координаты уже есть
    if (preselectedCoords && preselectedCoords.address === address) {
      const t: Task = { ...buildBaseTask(), latitude: preselectedCoords.lat, longitude: preselectedCoords.lng };
      setIsGeocoding(true);
      try { await onSave(t, role); onClose(); } catch { /* toast в parent */ } finally { setIsGeocoding(false); }
      return;
    }

    // Адрес не изменился у существующей задачи — пропускаем геокодирование
    if (task && (task.address ?? '') === address && task.latitude !== undefined && task.longitude !== undefined) {
      const t: Task = { ...buildBaseTask(), latitude: task.latitude, longitude: task.longitude, completed: task.completed };
      setIsGeocoding(true);
      try { await onSave(t, role); onClose(); } catch { /* toast в parent */ } finally { setIsGeocoding(false); }
      return;
    }

    // Геокодируем вручную введённый адрес
    setIsGeocoding(true);
    try {
      let results: GeocodeSuggestion[];
      try {
        results = await onGeocodeMultiple(address);
      } catch (err) {
        setGeocodeError(err instanceof Error ? err.message : 'Ошибка геокодирования');
        return;
      }
      if (results.length === 0) {
        setGeocodeError('Адрес не найден. Выберите вариант из подсказок или уточните запрос.');
        return;
      }
      const t: Task = { ...buildBaseTask(), latitude: results[0].lat, longitude: results[0].lng, address: results[0].displayName };
      try { await onSave(t, role); onClose(); } catch { /* toast в parent */ }
    } finally {
      setIsGeocoding(false);
    }
  };

  const hasAnyWindowField = windowStartDate || windowStartTime || windowEndDate || windowEndTime;
  const windowMinsLive = computeWindowMins();
  const durationMinsNum = duration.trim() !== '' ? parseInt(duration, 10) : 0;
  const hasLiveWindowError =
    (hasAnyWindowField && windowMinsLive !== null && windowMinsLive <= 0) ||
    (hasAnyWindowField && windowMinsLive !== null && windowMinsLive > 0 &&
      durationMinsNum > 0 && durationMinsNum > windowMinsLive);

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-[500px]">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>{task ? 'Редактировать задачу' : 'Добавить задачу'}</DialogTitle>
            <DialogDescription>
              Укажите детали задачи. Адрес необязателен — задачи без адреса отображаются отдельно и не включаются в маршрут.
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
              <Label htmlFor="address">
                Адрес{' '}
                <span className="text-muted-foreground font-normal">(необязательно)</span>
              </Label>
              <div className="relative">
                <Input
                  id="address"
                  placeholder="Например: г. Москва, ул. Тверская, д. 1"
                  value={address}
                  onChange={(e) => handleAddressChange(e.target.value)}
                  onBlur={() => setTimeout(() => setAutoSuggestOpen(false), 150)}
                  onFocus={() => autoSuggestions.length > 0 && setAutoSuggestOpen(true)}
                  autoComplete="off"
                />
                {autoSuggestOpen && autoSuggestions.length > 0 && (
                  <ul className="absolute z-50 mt-1 w-full rounded-md border bg-popover shadow-md text-sm overflow-hidden">
                    {autoSuggestions.map((s, i) => (
                      <li key={i}>
                        <button
                          type="button"
                          onMouseDown={(e) => { e.preventDefault(); handleSelectAutoSuggestion(s); }}
                          className="flex w-full items-center gap-2 px-3 py-2 text-left hover:bg-accent transition-colors"
                        >
                          <MapPin className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                          <span className="truncate">{s.displayName}</span>
                        </button>
                      </li>
                    ))}
                  </ul>
                )}
              </div>
              <p className="text-xs text-muted-foreground">
                Начните вводить адрес — появятся подсказки. Задачи без адреса не включаются в маршрут.
              </p>
              {geocodeError && (
                <p className="text-sm text-destructive">{geocodeError}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="duration">
                Длительность (минуты){' '}
                <span className="text-muted-foreground font-normal">(необязательно)</span>
              </Label>
              <Input
                id="duration"
                type="number"
                min="0"
                step="5"
                placeholder="Не задана (мгновенная)"
                value={duration}
                onChange={(e) => { setDuration(e.target.value); setWindowError(null); }}
              />
            </div>

            <div className="space-y-2">
              <Label>Временное окно <span className="text-muted-foreground font-normal">(необязательно)</span></Label>
              <p className="text-xs text-muted-foreground">
                Дата и время, в которые должно <strong>начаться</strong> выполнение задачи. Если указана
                только дата без времени — задача может быть выполнена в течение всего дня.
                Если дата не указана — используется сегодняшняя.
              </p>

              {/* Начало окна */}
              <div className="space-y-1">
                <Label className="text-xs font-medium">Начало окна</Label>
                <div className="grid grid-cols-2 gap-2">
                  <div className="flex items-center gap-1">
                    <Input
                      id="windowStartDate"
                      type="date"
                      aria-label="Дата начала окна"
                      value={windowStartDate}
                      onChange={(e) => { setWindowStartDate(e.target.value); setWindowError(null); }}
                      className="flex-1"
                    />
                    {windowStartDate && (
                      <button
                        type="button"
                        onClick={() => { setWindowStartDate(''); setWindowError(null); }}
                        className="flex-shrink-0 p-1 text-muted-foreground hover:text-destructive transition-colors rounded"
                        aria-label="Очистить дату начала окна"
                      >
                        <X className="h-3.5 w-3.5" />
                      </button>
                    )}
                  </div>
                  <div className="flex items-center gap-1">
                    <Input
                      id="windowStartTime"
                      type="time"
                      aria-label="Время начала окна"
                      value={windowStartTime}
                      onChange={(e) => { setWindowStartTime(e.target.value); setWindowError(null); }}
                      className="flex-1"
                    />
                    {windowStartTime && (
                      <button
                        type="button"
                        onClick={() => { setWindowStartTime(''); setWindowError(null); }}
                        className="flex-shrink-0 p-1 text-muted-foreground hover:text-destructive transition-colors rounded"
                        aria-label="Очистить время начала окна"
                      >
                        <X className="h-3.5 w-3.5" />
                      </button>
                    )}
                  </div>
                </div>
              </div>

              {/* Конец окна */}
              <div className="space-y-1">
                <Label className="text-xs font-medium">Конец окна</Label>
                <div className="grid grid-cols-2 gap-2">
                  <div className="flex items-center gap-1">
                    <Input
                      id="windowEndDate"
                      type="date"
                      aria-label="Дата конца окна"
                      value={windowEndDate}
                      onChange={(e) => { setWindowEndDate(e.target.value); setWindowError(null); }}
                      className="flex-1"
                    />
                    {windowEndDate && (
                      <button
                        type="button"
                        onClick={() => { setWindowEndDate(''); setWindowError(null); }}
                        className="flex-shrink-0 p-1 text-muted-foreground hover:text-destructive transition-colors rounded"
                        aria-label="Очистить дату конца окна"
                      >
                        <X className="h-3.5 w-3.5" />
                      </button>
                    )}
                  </div>
                  <div className="flex items-center gap-1">
                    <Input
                      id="windowEndTime"
                      type="time"
                      aria-label="Время конца окна"
                      value={windowEndTime}
                      onChange={(e) => { setWindowEndTime(e.target.value); setWindowError(null); }}
                      className="flex-1"
                    />
                    {windowEndTime && (
                      <button
                        type="button"
                        onClick={() => { setWindowEndTime(''); setWindowError(null); }}
                        className="flex-shrink-0 p-1 text-muted-foreground hover:text-destructive transition-colors rounded"
                        aria-label="Очистить время конца окна"
                      >
                        <X className="h-3.5 w-3.5" />
                      </button>
                    )}
                  </div>
                </div>
              </div>

              {hasAnyWindowField && windowMinsLive !== null && windowMinsLive <= 0 && (
                <p className="text-xs text-destructive">Конец окна должен быть позже начала.</p>
              )}
              {hasAnyWindowField && windowMinsLive !== null && windowMinsLive > 0 && (() => {
                const durationMins = duration.trim() !== '' ? parseInt(duration, 10) : 0;
                if (durationMins > 0 && durationMins > windowMinsLive) {
                  return (
                    <p className="text-xs text-destructive">
                      Длительность ({durationMins} мин) превышает размер окна ({Math.round(windowMinsLive)} мин) — задача не сможет уложиться в окно.
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
            <Button type="submit" disabled={isGeocoding || isSaving || hasLiveWindowError}>
              {(isGeocoding || isSaving) && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {task ? 'Сохранить' : 'Добавить'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
