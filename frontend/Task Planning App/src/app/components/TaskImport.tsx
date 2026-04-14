import { useRef, useState } from 'react';
import Papa from 'papaparse';
import * as XLSX from 'xlsx';
import { AlertCircle, CheckCircle2, FileUp, Upload, X } from 'lucide-react';
import { Button } from './ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from './ui/dialog';

export interface ImportedTaskRow {
  title: string;
  address?: string;
  duration?: number;
  windowDate?: string;       // "YYYY-MM-DD"; если не указано — сегодня
  windowStartTime?: string;
  windowEndTime?: string;
}

interface ParsedRow {
  raw: ImportedTaskRow;
  errors: string[];
}

interface TaskImportProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onImport: (rows: ImportedTaskRow[]) => Promise<void>;
}

const COLUMN_INFO = [
  { name: 'title', label: 'Название', required: true, hint: 'Текст, например: "Встреча с клиентом"' },
  { name: 'address', label: 'Адрес', required: false, hint: 'Полный адрес для геокодирования. Если пусто — задача без карты.' },
  { name: 'duration', label: 'Длительность (мин)', required: false, hint: 'Целое число ≥ 1, например: 30. Если пусто — без длительности.' },
  { name: 'date', label: 'Дата', required: false, hint: 'ДД.ММ.ГГГГ или ГГГГ-ММ-ДД, например: 25.12.2025. Если пусто — задача на сегодня.' },
  { name: 'window_start', label: 'Начало окна', required: false, hint: 'Формат ЧЧ:ММ, например: 09:00' },
  { name: 'window_end', label: 'Конец окна', required: false, hint: 'Формат ЧЧ:ММ, например: 17:30' },
];

const SAMPLE_CSV = `title,address,duration,date,window_start,window_end
Встреча с клиентом,"Москва, ул. Арбат, 1",45,25.12.2025,10:00,12:00
Доставка документов,"Москва, Тверская, 10",20,25.12.2025,14:00,
Подготовка отчёта,,60,,,
`;

function normalizeHeader(h: string) {
  return h.trim().toLowerCase().replace(/\s+/g, '_');
}

function isValidTime(value: string): boolean {
  return /^\d{1,2}:\d{2}$/.test(value);
}

function todayISO(): string {
  const d = new Date();
  const yyyy = d.getFullYear();
  const mm = String(d.getMonth() + 1).padStart(2, '0');
  const dd = String(d.getDate()).padStart(2, '0');
  return `${yyyy}-${mm}-${dd}`;
}

// принимает ДД.ММ.ГГГГ или ГГГГ-ММ-ДД, возвращает ISO или null
function parseDateValue(value: string): string | null {
  if (/^\d{4}-\d{2}-\d{2}$/.test(value)) {
    const d = new Date(value);
    return isNaN(d.getTime()) ? null : value;
  }
  if (/^\d{2}\.\d{2}\.\d{4}$/.test(value)) {
    const [day, month, year] = value.split('.');
    const iso = `${year}-${month}-${day}`;
    const d = new Date(iso);
    return isNaN(d.getTime()) ? null : iso;
  }
  return null;
}

function parseRows(rawRows: Record<string, string>[]): ParsedRow[] {
  return rawRows.map((row) => {
    const errors: string[] = [];

    const normalized: Record<string, string> = {};
    for (const [k, v] of Object.entries(row)) {
      normalized[normalizeHeader(k)] = (v ?? '').toString().trim();
    }

    const title = normalized['title'] ?? '';
    if (!title) errors.push('Поле title обязательно');

    const durationRaw = normalized['duration'] ?? '';
    const duration = durationRaw ? parseInt(durationRaw, 10) : undefined;
    if (durationRaw && (isNaN(duration!) || duration! < 1)) {
      errors.push('duration должно быть целым числом ≥ 1');
    }

    const dateRaw = normalized['date'] ?? '';
    let windowDate: string;
    if (!dateRaw) {
      windowDate = todayISO();
    } else {
      const parsed = parseDateValue(dateRaw);
      if (!parsed) {
        errors.push(`date не соответствует формату ДД.ММ.ГГГГ или ГГГГ-ММ-ДД (получено: "${dateRaw}")`);
        windowDate = todayISO();
      } else {
        windowDate = parsed;
      }
    }

    const windowStart = normalized['window_start'] ?? '';
    const windowEnd = normalized['window_end'] ?? '';

    if (windowStart && !isValidTime(windowStart)) {
      errors.push(`window_start не соответствует формату ЧЧ:ММ (получено: "${windowStart}")`);
    }
    if (windowEnd && !isValidTime(windowEnd)) {
      errors.push(`window_end не соответствует формату ЧЧ:ММ (получено: "${windowEnd}")`);
    }
    if (windowStart && windowEnd && isValidTime(windowStart) && isValidTime(windowEnd)) {
      const [sh, sm] = windowStart.split(':').map(Number);
      const [eh, em] = windowEnd.split(':').map(Number);
      if (sh * 60 + sm >= eh * 60 + em) {
        errors.push('window_end должен быть позже window_start');
      }
    }

    return {
      raw: {
        title,
        address: normalized['address'] || undefined,
        duration,
        windowDate,
        windowStartTime: windowStart || undefined,
        windowEndTime: windowEnd || undefined,
      },
      errors,
    };
  });
}

function readCSV(text: string): Record<string, string>[] {
  const result = Papa.parse<Record<string, string>>(text, {
    header: true,
    skipEmptyLines: true,
    transformHeader: (h) => h.trim(),
  });
  return result.data;
}

function readXLSX(buffer: ArrayBuffer): Record<string, string>[] {
  const wb = XLSX.read(buffer, { type: 'array' });
  const ws = wb.Sheets[wb.SheetNames[0]];
  if (!ws) return [];
  const data = XLSX.utils.sheet_to_json<Record<string, string>>(ws, {
    defval: '',
    raw: false,
  });
  return data;
}

function downloadSampleCSV() {
  const blob = new Blob([SAMPLE_CSV], { type: 'text/csv;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = 'tasks_template.csv';
  link.click();
  URL.revokeObjectURL(url);
}

function downloadSampleXLSX() {
  const rows = [
    { title: 'Встреча с клиентом', address: 'Москва, ул. Арбат, 1', duration: 45, date: '25.12.2025', window_start: '10:00', window_end: '12:00' },
    { title: 'Доставка документов', address: 'Москва, Тверская, 10', duration: 20, date: '25.12.2025', window_start: '14:00', window_end: '' },
    { title: 'Подготовка отчёта', address: '', duration: 60, date: '', window_start: '', window_end: '' },
  ];

  const ws = XLSX.utils.json_to_sheet(rows, {
    header: ['title', 'address', 'duration', 'date', 'window_start', 'window_end'],
  });

  ws['!cols'] = [
    { wch: 30 }, // title
    { wch: 35 }, // address
    { wch: 18 }, // duration
    { wch: 14 }, // date
    { wch: 15 }, // window_start
    { wch: 15 }, // window_end
  ];

  const wb = XLSX.utils.book_new();
  XLSX.utils.book_append_sheet(wb, ws, 'Задачи');

  XLSX.writeFile(wb, 'tasks_template.xlsx');
}

export function TaskImport({ open, onOpenChange, onImport }: TaskImportProps) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [fileName, setFileName] = useState<string | null>(null);
  const [parsedRows, setParsedRows] = useState<ParsedRow[]>([]);
  const [isImporting, setIsImporting] = useState(false);

  const validRows = parsedRows.filter((r) => r.errors.length === 0).map((r) => r.raw);
  const errorCount = parsedRows.filter((r) => r.errors.length > 0).length;

  function reset() {
    setFileName(null);
    setParsedRows([]);
    if (fileInputRef.current) fileInputRef.current.value = '';
  }

  function handleClose() {
    if (isImporting) return;
    reset();
    onOpenChange(false);
  }

  async function handleFile(file: File) {
    setFileName(file.name);
    setParsedRows([]);

    try {
      const ext = file.name.split('.').pop()?.toLowerCase() ?? '';
      let rawRows: Record<string, string>[];

      if (ext === 'csv') {
        const text = await file.text();
        rawRows = readCSV(text);
      } else if (ext === 'xlsx' || ext === 'xls') {
        const buffer = await file.arrayBuffer();
        rawRows = readXLSX(buffer);
      } else {
        setParsedRows([{ raw: { title: '', duration: 0 }, errors: ['Неподдерживаемый формат файла. Используйте CSV или XLSX.'] }]);
        return;
      }

      if (rawRows.length === 0) {
        setParsedRows([{ raw: { title: '', duration: 0 }, errors: ['Файл пуст или не содержит данных.'] }]);
        return;
      }

      setParsedRows(parseRows(rawRows));
    } catch (err) {
      setParsedRows([{ raw: { title: '', duration: 0 }, errors: [`Ошибка чтения файла: ${err instanceof Error ? err.message : String(err)}`] }]);
    }
  }

  function handleInputChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (file) void handleFile(file);
  }

  function handleDrop(e: React.DragEvent<HTMLDivElement>) {
    e.preventDefault();
    const file = e.dataTransfer.files?.[0];
    if (file) void handleFile(file);
  }

  async function handleImport() {
    if (validRows.length === 0 || isImporting) return;
    setIsImporting(true);
    try {
      await onImport(validRows);
      reset();
      onOpenChange(false);
    } finally {
      setIsImporting(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-5xl w-[96vw] flex flex-col max-h-[90vh] overflow-hidden">
        <DialogHeader className="shrink-0">
          <DialogTitle>Импорт задач из файла</DialogTitle>
          <DialogDescription>
            Загрузите CSV или Excel-файл. Задачи с адресом будут геокодированы автоматически.
          </DialogDescription>
        </DialogHeader>

        <div className="flex-1 min-h-0 overflow-y-auto">
          <div className="space-y-5 pr-1">
            <div className="rounded-lg border bg-gray-50 p-4 text-sm">
              <p className="mb-3 font-semibold text-gray-800">Формат файла</p>
              <p className="mb-3 text-gray-600">
                Первая строка — заголовки столбцов. Порядок столбцов может быть любым.
                Поддерживаются форматы: <strong>.csv</strong> (разделитель — запятая) и{' '}
                <strong>.xlsx / .xls</strong>.
              </p>
              <table className="w-full text-xs">
                <thead>
                  <tr className="border-b text-left text-gray-500">
                    <th className="py-1 pr-3 font-medium">Столбец</th>
                    <th className="py-1 pr-3 font-medium">Обязательный</th>
                    <th className="py-1 font-medium">Описание</th>
                  </tr>
                </thead>
                <tbody>
                  {COLUMN_INFO.map((col) => (
                    <tr key={col.name} className="border-b last:border-0">
                      <td className="py-1.5 pr-3 font-mono text-blue-700">{col.name}</td>
                      <td className="py-1.5 pr-3 text-gray-600">
                        {col.required ? (
                          <span className="rounded bg-red-100 px-1.5 py-0.5 text-red-700">Да</span>
                        ) : (
                          <span className="rounded bg-gray-100 px-1.5 py-0.5 text-gray-600">Нет</span>
                        )}
                      </td>
                      <td className="py-1.5 text-gray-600">{col.hint}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
              <div className="mt-3 flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={downloadSampleCSV}
                >
                  <FileUp className="mr-1.5 h-3.5 w-3.5" />
                  Скачать шаблон CSV
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={downloadSampleXLSX}
                >
                  <FileUp className="mr-1.5 h-3.5 w-3.5" />
                  Скачать шаблон Excel
                </Button>
              </div>
            </div>

            <div
              className="flex cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed border-gray-300 bg-white p-8 text-center transition-colors hover:border-blue-400 hover:bg-blue-50"
              onDragOver={(e) => e.preventDefault()}
              onDrop={handleDrop}
              onClick={() => fileInputRef.current?.click()}
              role="button"
              tabIndex={0}
              onKeyDown={(e) => e.key === 'Enter' && fileInputRef.current?.click()}
            >
              <Upload className="mb-3 h-8 w-8 text-gray-400" />
              {fileName ? (
                <p className="text-sm font-medium text-gray-700">
                  Выбран файл: <span className="text-blue-700">{fileName}</span>
                </p>
              ) : (
                <>
                  <p className="text-sm font-medium text-gray-700">
                    Перетащите файл или нажмите для выбора
                  </p>
                  <p className="mt-1 text-xs text-gray-500">CSV, XLSX или XLS</p>
                </>
              )}
              <input
                ref={fileInputRef}
                type="file"
                accept=".csv,.xlsx,.xls"
                className="hidden"
                onChange={handleInputChange}
              />
            </div>

            {parsedRows.length > 0 && (
              <div>
                <div className="mb-2 flex items-center justify-between">
                  <p className="text-sm font-semibold text-gray-800">
                    Предпросмотр ({parsedRows.length} строк)
                  </p>
                  <div className="flex gap-3 text-xs">
                    <span className="flex items-center gap-1 text-green-700">
                      <CheckCircle2 className="h-3.5 w-3.5" />
                      Корректных: {validRows.length}
                    </span>
                    {errorCount > 0 && (
                      <span className="flex items-center gap-1 text-red-600">
                        <AlertCircle className="h-3.5 w-3.5" />
                        С ошибками: {errorCount}
                      </span>
                    )}
                  </div>
                </div>
                <div className="rounded-lg border text-xs overflow-auto max-h-64">
                  <table className="w-full min-w-[560px]">
                    <thead className="bg-gray-50 text-left text-gray-500">
                      <tr>
                        <th className="px-3 py-2">#</th>
                        <th className="px-3 py-2">Название</th>
                        <th className="px-3 py-2">Адрес</th>
                        <th className="px-3 py-2">Длит., мин</th>
                        <th className="px-3 py-2">Дата</th>
                        <th className="px-3 py-2">Окно</th>
                        <th className="px-3 py-2">Статус</th>
                      </tr>
                    </thead>
                    <tbody>
                      {parsedRows.map((row, i) => {
                        const ok = row.errors.length === 0;
                        return (
                          <tr
                            key={i}
                            className={ok ? 'border-t' : 'border-t bg-red-50'}
                          >
                            <td className="px-3 py-1.5 text-gray-400">{i + 1}</td>
                            <td className="px-3 py-1.5">
                              {row.raw.title || <span className="italic text-gray-400">—</span>}
                            </td>
                            <td className="max-w-[160px] truncate px-3 py-1.5 text-gray-600">
                              {row.raw.address || <span className="text-gray-400">—</span>}
                            </td>
                            <td className="px-3 py-1.5">
                              {row.raw.duration != null ? row.raw.duration : <span className="text-gray-400">—</span>}
                            </td>
                            <td className="whitespace-nowrap px-3 py-1.5 text-gray-600">
                              {row.raw.windowDate ?? <span className="text-gray-400">—</span>}
                            </td>
                            <td className="whitespace-nowrap px-3 py-1.5 text-gray-600">
                              {row.raw.windowStartTime || row.raw.windowEndTime
                                ? `${row.raw.windowStartTime ?? '—'} – ${row.raw.windowEndTime ?? '—'}`
                                : <span className="text-gray-400">—</span>}
                            </td>
                            <td className="px-3 py-1.5">
                              {ok ? (
                                <CheckCircle2 className="h-4 w-4 text-green-600" />
                              ) : (
                                <div className="flex items-start gap-1">
                                  <X className="mt-0.5 h-3.5 w-3.5 flex-shrink-0 text-red-500" />
                                  <span className="text-red-600">{row.errors.join('; ')}</span>
                                </div>
                              )}
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
                {errorCount > 0 && (
                  <p className="mt-2 text-xs text-gray-500">
                    Строки с ошибками будут пропущены. Исправьте файл и загрузите повторно.
                  </p>
                )}
              </div>
            )}
          </div>
        </div>

        <DialogFooter className="shrink-0 mt-2 pt-3 border-t">
          <Button variant="outline" onClick={handleClose} disabled={isImporting}>
            Отмена
          </Button>
          <Button
            onClick={() => void handleImport()}
            disabled={validRows.length === 0 || isImporting}
          >
            {isImporting ? (
              <>
                <Upload className="mr-2 h-4 w-4 animate-pulse" />
                Импортируется...
              </>
            ) : (
              <>
                <Upload className="mr-2 h-4 w-4" />
                Импортировать {validRows.length > 0 ? `(${validRows.length})` : ''}
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
