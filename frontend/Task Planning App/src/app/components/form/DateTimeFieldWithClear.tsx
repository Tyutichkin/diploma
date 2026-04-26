import { Input } from '../ui/input';
import { Label } from '../ui/label';
import { X } from 'lucide-react';

interface ClearableInputProps {
  id: string;
  type: 'date' | 'time';
  ariaLabel: string;
  clearAriaLabel: string;
  value: string;
  onChange: (value: string) => void;
}

function ClearableInput({ id, type, ariaLabel, clearAriaLabel, value, onChange }: ClearableInputProps) {
  return (
    <div className="flex items-center gap-1">
      <Input
        id={id}
        type={type}
        aria-label={ariaLabel}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="flex-1"
      />
      {value && (
        <button
          type="button"
          onClick={() => onChange('')}
          className="flex-shrink-0 p-1 text-muted-foreground hover:text-destructive transition-colors rounded"
          aria-label={clearAriaLabel}
        >
          <X className="h-3.5 w-3.5" />
        </button>
      )}
    </div>
  );
}

interface DateTimeFieldWithClearProps {
  label: string;
  idPrefix: string;
  date: string;
  time: string;
  onDateChange: (value: string) => void;
  onTimeChange: (value: string) => void;
  dateAriaLabel: string;
  timeAriaLabel: string;
  clearDateAriaLabel: string;
  clearTimeAriaLabel: string;
}

export function DateTimeFieldWithClear({
  label,
  idPrefix,
  date,
  time,
  onDateChange,
  onTimeChange,
  dateAriaLabel,
  timeAriaLabel,
  clearDateAriaLabel,
  clearTimeAriaLabel,
}: DateTimeFieldWithClearProps) {
  return (
    <div className="space-y-1">
      <Label className="text-xs font-medium">{label}</Label>
      <div className="grid grid-cols-2 gap-2">
        <ClearableInput
          id={`${idPrefix}Date`}
          type="date"
          ariaLabel={dateAriaLabel}
          clearAriaLabel={clearDateAriaLabel}
          value={date}
          onChange={onDateChange}
        />
        <ClearableInput
          id={`${idPrefix}Time`}
          type="time"
          ariaLabel={timeAriaLabel}
          clearAriaLabel={clearTimeAriaLabel}
          value={time}
          onChange={onTimeChange}
        />
      </div>
    </div>
  );
}
