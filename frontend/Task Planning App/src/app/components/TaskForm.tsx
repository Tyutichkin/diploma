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
import { Loader2, MapPin } from 'lucide-react';

interface TaskFormProps {
  task: Task | null;
  isOpen: boolean;
  isSaving?: boolean;
  onClose: () => void;
  onSave: (task: Task) => Promise<void>;
  onGeocodeMultiple: (address: string) => Promise<GeocodeSuggestion[]>;
}

export function TaskForm({ task, isOpen, isSaving = false, onClose, onSave, onGeocodeMultiple }: TaskFormProps) {
  const [title, setTitle] = useState('');
  const [address, setAddress] = useState('');
  const [duration, setDuration] = useState('30');
  const [timeWindowStart, setTimeWindowStart] = useState('');
  const [timeWindowEnd, setTimeWindowEnd] = useState('');
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
    setSuggestions([]);
    setSelectedSuggestion(null);
    setGeocodeError(null);
    setWindowError(null);
    setPendingTask(null);
  }, [task, isOpen]);

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
        timeWindowStart: timeWindowStart || undefined,
        timeWindowEnd: timeWindowEnd || undefined,
        completed: task.completed,
        order: task.order,
      };
      setIsGeocoding(true);
      try {
        await onSave(updatedTask);
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
        timeWindowStart: timeWindowStart || undefined,
        timeWindowEnd: timeWindowEnd || undefined,
        completed: task?.completed || false,
        order: task?.order,
      };
      setPendingTask(built);

      if (results.length === 1) {
        // Only one result — use it directly
        const taskWithCoords: Task = { ...built, latitude: results[0].lat, longitude: results[0].lng, address: results[0].displayName };
        setIsGeocoding(true);
        try {
          await onSave(taskWithCoords);
          onClose();
        } catch {
          // error shown via toast in parent
        } finally {
          setIsGeocoding(false);
        }
        return;
      }

      // Multiple results — let user pick
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
      await onSave(taskWithCoords);
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
                    <Input
                      id="timeWindowStart"
                      type="time"
                      value={timeWindowStart}
                      onChange={(e) => { setTimeWindowStart(e.target.value); setWindowError(null); }}
                    />
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="timeWindowEnd">Конец окна</Label>
                    <Input
                      id="timeWindowEnd"
                      type="time"
                      value={timeWindowEnd}
                      onChange={(e) => { setTimeWindowEnd(e.target.value); setWindowError(null); }}
                    />
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
