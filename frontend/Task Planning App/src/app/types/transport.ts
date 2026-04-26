export type TransportMode = 'auto' | 'pedestrian' | 'masstransit';

export interface TransportInfo {
  name: string;
  type: string;
  color?: string;
}

export interface RouteSegment {
  kind: 'pedestrian' | 'transport';
  duration?: string;
  distance?: string;
  transports?: TransportInfo[];
  stopFrom?: string;
  stopTo?: string;
  stopsCount?: number;
}

export interface RouteLeg {
  fromTitle: string;
  toTitle: string;
  duration?: string;
  distance?: string;
  segments: RouteSegment[];
}

export interface TransportTypeMeta {
  icon: string;
  label: string;
  bg: string;
  text: string;
}

export const TRANSPORT_TYPE_META: Record<string, TransportTypeMeta> = {
  subway:      { icon: 'М',  label: 'Метро',       bg: 'bg-red-600',    text: 'text-white' },
  underground: { icon: 'М',  label: 'Метро',       bg: 'bg-red-600',    text: 'text-white' },
  bus:         { icon: '🚌', label: 'Автобус',     bg: 'bg-blue-500',   text: 'text-white' },
  tram:        { icon: '🚋', label: 'Трамвай',     bg: 'bg-green-600',  text: 'text-white' },
  trolleybus:  { icon: '🚎', label: 'Троллейбус',  bg: 'bg-purple-600', text: 'text-white' },
  minibus:     { icon: '🚐', label: 'Маршрутка',   bg: 'bg-yellow-500', text: 'text-white' },
  rail:        { icon: '🚆', label: 'Электричка',  bg: 'bg-gray-700',   text: 'text-white' },
};
