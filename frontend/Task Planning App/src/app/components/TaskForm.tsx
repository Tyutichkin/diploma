import { useEffect, useMemo, useState, type FormEvent } from 'react';
import { Task } from '../types/task';
import { GeocodeSuggestion } from '../utils/routeOptimizer';
import { todayISO } from '../utils/formatters';
import { useAddressAutocomplete } from '../hooks/useAddressAutocomplete';
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
import { Loader2, MapPin, Flag } from 'lucide-react';
import { cn } from './ui/utils';
import { DateTimeFieldWithClear } from './form/DateTimeFieldWithClear';

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

interface PreselectedCoords {
  lat: number;
  lng: number;
  address: string;
}

function computeWindowMins(
  windowStartDate: string,
  windowStartTime: string,
  windowEndDate: string,
  windowEndTime: string,
): number | null {
  if (!windowStartDate && !windowStartTime && !windowEndDate && !windowEndTime) return null;

  // Без дат, но с обоими временами — сравниваем в рамках одного дня.
  if (!windowStartDate && !windowEndDate && windowStartTime && windowEndTime) {
    const [sh, sm] = windowStartTime.split(':').map(Number);
    const [eh, em] = windowEndTime.split(':').map(Number);
    return eh * 60 + em - (sh * 60 + sm);
  }

  const sDate = windowStartDate || windowEndDate;
  const eDate = windowEndDate || windowStartDate;
  if (!sDate || !eDate) return null;

  const sTime = windowStartTime || '00:00';
  const eTime = windowEndTime || '23:59';
  const start = new Date(`${sDate}T${sTime}`);
  const end = new Date(`${eDate}T${eTime}`);
  if (isNaN(start.getTime()) || isNaN(end.getTime())) return null;

  return (end.getTime() - start.getTime()) / 60000;
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

  // Координаты из выбранной подсказки — чтобы не геокодировать повторно.
  const [preselectedCoords, setPreselectedCoords] = useState<PreselectedCoords | null>(null);
  const autocomplete = useAddressAutocomplete();

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
      const today = todayISO();
      setTitle('');
      setAddress('');
      setDuration('');
      setWindowStartDate(today);
      setWindowStartTime('');
      setWindowEndDate(today);
      setWindowEndTime('');
    }
    setRole(initialRole);
    setGeocodeError(null);
    setWindowError(null);
    setPreselectedCoords(null);
    autocomplete.reset();
    // autocomplete.reset is stable; включать его в deps вызовет лишний прогон при re-create
    // eslint-disable-next-line react-hooks/exhaustive-deps
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
    autocomplete.query(value);
  };

  const handleSelectAutoSuggestion = (suggestion: GeocodeSuggestion) => {
    autocomplete.reset();
    setAddress(suggestion.displayName);
    setGeocodeError(null);
    setPreselectedCoords({ lat: suggestion.lat, lng: suggestion.lng, address: suggestion.displayName });
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

  const submit = async (toSave: Task) => {
    setIsGeocoding(true);
    try {
      await onSave(toSave, role);
      onClose();
    } catch {
      // toast в parent
    } finally {
      setIsGeocoding(false);
    }
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setGeocodeError(null);
    setWindowError(null);

    const windowMins = computeWindowMins(windowStartDate, windowStartTime, windowEndDate, windowEndTime);
    if (windowMins !== null && windowMins <= 0) return;

    if (windowMins !== null) {
      const durationMins = duration.trim() !== '' ? parseInt(duration, 10) : 0;
      if (durationMins > 0 && durationMins > windowMins) {
        setWindowError(
          `Длительность (${durationMins} мин) превышает размер окна (${Math.round(windowMins)} мин). Увеличьте окно или уменьшите длительность.`,
        );
        return;
      }
    }

    // Задача без адреса
    if (!address.trim()) {
      await submit({ ...buildBaseTask(), address: undefined, latitude: undefined, longitude: undefined });
      return;
    }

    // Адрес выбран из автодополнения — координаты уже есть
    if (preselectedCoords && preselectedCoords.address === address) {
      await submit({ ...buildBaseTask(), latitude: preselectedCoords.lat, longitude: preselectedCoords.lng });
      return;
    }

    // Адрес не изменился у существующей задачи — пропускаем геокодирование
    if (
      task &&
      (task.address ?? '') === address &&
      task.latitude !== undefined &&
      task.longitude !== undefined
    ) {
      await submit({
        ...buildBaseTask(),
        latitude: task.latitude,
        longitude: task.longitude,
        completed: task.completed,
      });
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
      const toSave: Task = {
        ...buildBaseTask(),
        latitude: results[0].lat,
        longitude: results[0].lng,
        address: results[0].displayName,
      };
      try {
        await onSave(toSave, role);
        onClose();
      } catch {
        // toast в parent
      }
    } finally {
      setIsGeocoding(false);
    }
  };

  const hasAnyWindowField =
    Boolean(windowStartDate) || Boolean(windowStartTime) || Boolean(windowEndDate) || Boolean(windowEndTime);
  const windowMinsLive = useMemo(
    () => computeWindowMins(windowStartDate, windowStartTime, windowEndDate, windowEndTime),
    [windowStartDate, windowStartTime, windowEndDate, windowEndTime],
  );
  const durationMinsNum = duration.trim() !== '' ? parseInt(duration, 10) : 0;

  const windowInverted = hasAnyWindowField && windowMinsLive !== null && windowMinsLive <= 0;
  const durationOverWindow =
    hasAnyWindowField &&
    windowMinsLive !== null &&
    windowMinsLive > 0 &&
    durationMinsNum > 0 &&
    durationMinsNum > windowMinsLive;
  const hasLiveWindowError = windowInverted || durationOverWindow;

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
                Адрес <span className="text-muted-foreground font-normal">(необязательно)</span>
              </Label>
              <div className="relative">
                <Input
                  id="address"
                  placeholder="Например: г. Москва, ул. Тверская, д. 1"
                  value={address}
                  onChange={(e) => handleAddressChange(e.target.value)}
                  onBlur={() => setTimeout(() => autocomplete.setOpen(false), 150)}
                  onFocus={() => autocomplete.suggestions.length > 0 && autocomplete.setOpen(true)}
                  autoComplete="off"
                />
                {autocomplete.open && autocomplete.suggestions.length > 0 && (
                  <ul className="absolute z-50 mt-1 w-full rounded-md border bg-popover shadow-md text-sm overflow-hidden">
                    {autocomplete.suggestions.map((s, i) => (
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
              {geocodeError && <p className="text-sm text-destructive">{geocodeError}</p>}
            </div>

            <div className="space-y-2">
              <Label htmlFor="duration">
                Длительность (минуты) <span className="text-muted-foreground font-normal">(необязательно)</span>
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

              <DateTimeFieldWithClear
                label="Начало окна"
                idPrefix="windowStart"
                date={windowStartDate}
                time={windowStartTime}
                onDateChange={(v) => { setWindowStartDate(v); setWindowError(null); }}
                onTimeChange={(v) => { setWindowStartTime(v); setWindowError(null); }}
                dateAriaLabel="Дата начала окна"
                timeAriaLabel="Время начала окна"
                clearDateAriaLabel="Очистить дату начала окна"
                clearTimeAriaLabel="Очистить время начала окна"
              />

              <DateTimeFieldWithClear
                label="Конец окна"
                idPrefix="windowEnd"
                date={windowEndDate}
                time={windowEndTime}
                onDateChange={(v) => { setWindowEndDate(v); setWindowError(null); }}
                onTimeChange={(v) => { setWindowEndTime(v); setWindowError(null); }}
                dateAriaLabel="Дата конца окна"
                timeAriaLabel="Время конца окна"
                clearDateAriaLabel="Очистить дату конца окна"
                clearTimeAriaLabel="Очистить время конца окна"
              />

              {windowInverted && (
                <p className="text-xs text-destructive">Конец окна должен быть позже начала.</p>
              )}
              {durationOverWindow && (
                <p className="text-xs text-destructive">
                  Длительность ({durationMinsNum} мин) превышает размер окна ({Math.round(windowMinsLive!)} мин) — задача не сможет уложиться в окно.
                </p>
              )}
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
