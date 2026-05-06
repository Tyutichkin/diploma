import { RouteSegment, TransportMode, TRANSPORT_TYPE_META } from '../../types/transport';

export type SegmentIconType =
  | 'pedestrian'
  | 'auto'
  | 'subway'
  | 'rail'
  | 'tram'
  | 'bus'
  | 'trolleybus'
  | 'minibus';

export interface SegmentIconResult {
  type: SegmentIconType;
  icon: string;
  label: string;
}

const TRANSIT_PRIORITY: SegmentIconType[] = [
  'subway',
  'rail',
  'tram',
  'bus',
  'trolleybus',
  'minibus',
];

const PEDESTRIAN: SegmentIconResult = { type: 'pedestrian', icon: '🚶', label: 'Пешком' };
const AUTO: SegmentIconResult = { type: 'auto', icon: '🚗', label: 'Авто' };

function normalizeTransportType(raw: string): SegmentIconType | null {
  if (raw === 'underground') return 'subway';
  if (TRANSIT_PRIORITY.includes(raw as SegmentIconType)) return raw as SegmentIconType;
  return null;
}

export function getSegmentIcon(mode: TransportMode, segments: RouteSegment[]): SegmentIconResult {
  if (mode === 'pedestrian') return PEDESTRIAN;
  if (mode === 'auto') return AUTO;

  const presentTypes = new Set<SegmentIconType>();
  for (const seg of segments) {
    if (seg.kind !== 'transport') continue;
    for (const tr of seg.transports ?? []) {
      const t = normalizeTransportType(tr.type);
      if (t) presentTypes.add(t);
    }
  }
  for (const candidate of TRANSIT_PRIORITY) {
    if (presentTypes.has(candidate)) {
      const meta = TRANSPORT_TYPE_META[candidate] ?? TRANSPORT_TYPE_META.bus;
      return { type: candidate, icon: meta.icon, label: meta.label };
    }
  }
  return PEDESTRIAN;
}
