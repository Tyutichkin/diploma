import { useEffect, useRef, useState } from 'react';
import { Info } from 'lucide-react';

const ITEMS = [
  { icon: '🚶', label: 'Пешком' },
  { icon: 'М',  label: 'Метро' },
  { icon: '🚌', label: 'Наземный транспорт' },
  { icon: '🚗', label: 'На автомобиле' },
];

const FOOTNOTE =
  'Расстояния указаны ориентировочно и могут отличаться в зависимости ' +
  'от выбранного маршрута и времени в пути.';

function useIsMobile(): boolean {
  const [isMobile, setIsMobile] = useState(false);
  useEffect(() => {
    if (typeof window === 'undefined') return;
    const mq = window.matchMedia('(max-width: 640px)');
    const update = () => setIsMobile(mq.matches);
    update();
    mq.addEventListener('change', update);
    return () => mq.removeEventListener('change', update);
  }, []);
  return isMobile;
}

export function MapLegend() {
  const isMobile = useIsMobile();
  const [open, setOpen] = useState(false);
  const popoverRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!isMobile || !open) return;
    const handler = (e: MouseEvent) => {
      if (popoverRef.current && !popoverRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [isMobile, open]);

  if (isMobile) {
    return (
      <div ref={popoverRef} className="absolute bottom-3 right-3 z-[1000]">
        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          className="flex items-center justify-center w-8 h-8 rounded-full bg-white/95 shadow ring-1 ring-gray-200 text-gray-700"
          aria-label="Легенда"
        >
          <Info className="h-4 w-4" />
        </button>
        {open && (
          <div className="absolute bottom-10 right-0 bg-white/95 shadow-lg rounded-xl ring-1 ring-gray-200 p-3 w-56 text-sm">
            <div className="font-medium text-gray-900 mb-2">Условные обозначения</div>
            <div className="space-y-1">
              {ITEMS.map((it) => (
                <div key={it.label} className="flex items-center gap-2 text-gray-700">
                  <span className="w-5 text-center">{it.icon}</span>
                  <span>{it.label}</span>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    );
  }

  return (
    <div className="absolute bottom-3 left-1/2 -translate-x-1/2 z-[1000] pointer-events-none">
      <div className="pointer-events-auto bg-white/95 shadow rounded-xl ring-1 ring-gray-200 px-3 py-2 flex items-center gap-4 text-sm text-gray-700">
        {ITEMS.map((it) => (
          <div key={it.label} className="flex items-center gap-1.5">
            <span>{it.icon}</span>
            <span>{it.label}</span>
          </div>
        ))}
      </div>
      <div className="mt-1 text-[11px] text-gray-500 max-w-md text-center px-1">{FOOTNOTE}</div>
    </div>
  );
}
