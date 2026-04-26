import { useCallback } from 'react';
import Papa from 'papaparse';
import * as XLSX from 'xlsx';
import { toast } from 'sonner';
import { Task, OptimizedRoute } from '../types/task';
import { downloadBlob } from '../utils/download';

const COLUMN_HEADERS = ['title', 'address', 'duration', 'date', 'window_start', 'window_end'] as const;

const COLUMN_WIDTHS: { wch: number }[] = [
  { wch: 30 }, // title
  { wch: 35 }, // address
  { wch: 18 }, // duration
  { wch: 14 }, // date
  { wch: 15 }, // window_start
  { wch: 15 }, // window_end
];

function todayStamp() {
  return new Date().toISOString().split('T')[0];
}

function buildExportRows(tasks: Task[]) {
  return tasks.map((task) => ({
    title: task.title,
    address: task.address ?? '',
    duration: task.duration ?? '',
    date: task.windowStartDate ?? '',
    window_start: task.windowStartTime ?? '',
    window_end: task.windowEndTime ?? '',
  }));
}

export function useTaskExport(
  tasks: Task[],
  optimizedInfo: OptimizedRoute | null,
  routeOptimized: boolean,
  activeRouteId: string | null,
) {
  const exportCSV = useCallback(() => {
    if (tasks.length === 0) return;
    const csv = Papa.unparse(buildExportRows(tasks), { columns: [...COLUMN_HEADERS] });
    downloadBlob(new Blob([csv], { type: 'text/csv;charset=utf-8;' }), `tasks-${todayStamp()}.csv`);
    toast.success('Задачи экспортированы в CSV');
  }, [tasks]);

  const exportXLSX = useCallback(() => {
    if (tasks.length === 0) return;
    const ws = XLSX.utils.json_to_sheet(buildExportRows(tasks), { header: [...COLUMN_HEADERS] });
    ws['!cols'] = COLUMN_WIDTHS;
    const wb = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(wb, ws, 'Задачи');
    XLSX.writeFile(wb, `tasks-${todayStamp()}.xlsx`);
    toast.success('Задачи экспортированы в Excel');
  }, [tasks]);

  const exportRouteJSON = useCallback(() => {
    const exportData = {
      exportDate: new Date().toISOString(),
      optimized: routeOptimized,
      routeId: activeRouteId,
      tasks: tasks.map((task, index) => ({
        order: index + 1,
        id: task.id,
        title: task.title,
        address: task.address,
        coordinates: { latitude: task.latitude, longitude: task.longitude },
        duration: task.duration,
        timeWindow: {
          startDate: task.windowStartDate,
          startTime: task.windowStartTime,
          endDate: task.windowEndDate,
          endTime: task.windowEndTime,
        },
        completed: task.completed,
      })),
      summary: optimizedInfo
        ? {
            totalDistance: optimizedInfo.totalDistance,
            totalDuration: optimizedInfo.totalDuration,
            totalTravelTime: optimizedInfo.totalTravelTime,
          }
        : null,
    };

    const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: 'application/json' });
    downloadBlob(blob, `route-${todayStamp()}.json`);
    toast.success('Маршрут экспортирован');
  }, [tasks, optimizedInfo, routeOptimized, activeRouteId]);

  return { exportCSV, exportXLSX, exportRouteJSON };
}
