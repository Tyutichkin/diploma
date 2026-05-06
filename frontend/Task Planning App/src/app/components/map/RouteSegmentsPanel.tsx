import { useState } from 'react';
import { ChevronDown, ChevronUp } from 'lucide-react';
import { RouteLeg, TransportMode, TRANSPORT_TYPE_META } from '../../types/transport';

interface RouteSegmentsPanelProps {
  legs: RouteLeg[];
  mode: TransportMode;
  onLegFocus?: (legIndex: number) => void;
}

export function RouteSegmentsPanel({ legs, mode, onLegFocus }: RouteSegmentsPanelProps) {
  const [open, setOpen] = useState(true);
  if (legs.length === 0) return null;

  return (
    <div className="flex-shrink-0 border border-blue-200 rounded-lg bg-blue-50 overflow-hidden">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="w-full flex items-center justify-between px-4 py-2 text-sm font-medium text-blue-800 hover:bg-blue-100 transition-colors"
      >
        <span>Детали маршрута</span>
        {open ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
      </button>

      {open && (
        <div className="max-h-52 overflow-y-auto px-4 pb-3 space-y-4">
          {legs.map((leg, li) => (
            <div key={li}>
              <button
                type="button"
                onClick={() => onLegFocus?.(li)}
                className="text-xs font-semibold text-blue-700 mb-1 pt-2 hover:underline w-full text-left"
              >
                {leg.fromTitle} → {leg.toTitle}
                {leg.duration && <span className="text-gray-500 font-normal"> · {leg.duration}</span>}
                {leg.distance && <span className="text-gray-500 font-normal"> · {leg.distance}</span>}
              </button>

              {mode === 'masstransit' ? (
                <MasstransitLegDetails leg={leg} />
              ) : (
                <SimpleLegDetails mode={mode} />
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function SimpleLegDetails({ mode }: { mode: TransportMode }) {
  const icon = mode === 'pedestrian' ? '🚶' : '🚗';
  const label = mode === 'pedestrian' ? 'Пешком' : 'На автомобиле';
  return (
    <div className="flex items-center gap-2 text-sm text-gray-700">
      <span className="text-base leading-none">{icon}</span>
      <span>{label}</span>
    </div>
  );
}

function MasstransitLegDetails({ leg }: { leg: RouteLeg }) {
  return (
    <div className="space-y-1">
      {leg.segments.map((seg, si) => (
        <div key={si} className="flex items-start gap-2">
          {seg.kind === 'pedestrian' ? (
            <>
              <span className="text-base leading-none mt-0.5">🚶</span>
              <div className="text-sm text-gray-700">
                <span>Пешком</span>
                {seg.distance && <span className="text-gray-500"> · {seg.distance}</span>}
                {seg.duration && <span className="text-gray-500"> · {seg.duration}</span>}
              </div>
            </>
          ) : (
            <>
              <div className="flex flex-wrap gap-1 mt-0.5">
                {(seg.transports ?? []).map((tr, ti) => {
                  const meta = TRANSPORT_TYPE_META[tr.type] ?? TRANSPORT_TYPE_META.bus;
                  const bgStyle = tr.color ? { backgroundColor: tr.color } : undefined;
                  return (
                    <span
                      key={ti}
                      className={`inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded text-xs font-bold ${meta.text} ${!tr.color ? meta.bg : ''}`}
                      style={bgStyle}
                      title={`${meta.label} ${tr.name}`}
                    >
                      {tr.type === 'subway' || tr.type === 'underground' ? 'М' : meta.icon}{' '}
                      {tr.name}
                    </span>
                  );
                })}
              </div>
              <div className="text-sm text-gray-700">
                {seg.stopFrom && (
                  <span>от <span className="font-medium">{seg.stopFrom}</span></span>
                )}
                {seg.stopTo && (
                  <span> до <span className="font-medium">{seg.stopTo}</span></span>
                )}
                {seg.duration && <span className="text-gray-500"> · {seg.duration}</span>}
              </div>
            </>
          )}
        </div>
      ))}
    </div>
  );
}
