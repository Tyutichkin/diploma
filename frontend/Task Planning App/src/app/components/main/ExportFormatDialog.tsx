import { Button } from '../ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '../ui/dialog';
import { Download } from 'lucide-react';

interface ExportFormatDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCSV: () => void;
  onXLSX: () => void;
  onJSON: () => void;
}

export function ExportFormatDialog({
  open,
  onOpenChange,
  onCSV,
  onXLSX,
  onJSON,
}: ExportFormatDialogProps) {
  const pick = (action: () => void) => () => {
    action();
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>Экспорт задач</DialogTitle>
          <DialogDescription>Выберите формат файла для скачивания</DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-3 pt-2">
          <Button variant="outline" className="justify-start" onClick={pick(onCSV)}>
            <Download className="mr-2 h-4 w-4" />
            CSV
            <span className="ml-auto text-xs text-gray-400">совместим с импортом</span>
          </Button>
          <Button variant="outline" className="justify-start" onClick={pick(onXLSX)}>
            <Download className="mr-2 h-4 w-4" />
            Excel (.xlsx)
            <span className="ml-auto text-xs text-gray-400">совместим с импортом</span>
          </Button>
          <Button variant="outline" className="justify-start" onClick={pick(onJSON)}>
            <Download className="mr-2 h-4 w-4" />
            JSON
            <span className="ml-auto text-xs text-gray-400">полные данные маршрута</span>
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
