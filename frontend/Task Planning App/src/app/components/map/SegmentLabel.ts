/* eslint-disable @typescript-eslint/no-explicit-any */
import { Task } from '../../types/task';
import { RouteSegment, TransportMode } from '../../types/transport';
import { haversineMeters } from './geo';
import { getSegmentIcon, SegmentIconResult } from './segmentTransportIcon';

const MIN_SEGMENT_DISTANCE_M = 50;

interface CreateSegmentLabelArgs {
  fromTask: Task;
  toTask: Task;
  legIndex: number;
  duration?: string;
  distance?: string;
  mode: TransportMode;
  segments: RouteSegment[];
}

let cachedLayout: any = null;

function getLabelLayout(): any {
  if (cachedLayout) return cachedLayout;
  cachedLayout = window.ymaps.templateLayoutFactory.createClass(
    [
      '<div style="display:inline-flex;align-items:center;gap:4px;',
      'padding:4px 8px;border-radius:12px;background:#ffffff;',
      'border:1px solid #e5e7eb;box-shadow:0 1px 2px rgba(0,0,0,0.15);',
      'font-size:12px;font-weight:500;color:#111827;line-height:1.2;',
      'white-space:nowrap;font-family:inherit;">',
      '$[properties.iconContent]',
      '</div>',
    ].join(''),
  );
  return cachedLayout;
}

export function shouldShowSegmentLabel(fromTask: Task, toTask: Task): boolean {
  if (
    fromTask.latitude === undefined || fromTask.longitude === undefined ||
    toTask.latitude === undefined || toTask.longitude === undefined
  ) return false;
  const d = haversineMeters(fromTask.latitude, fromTask.longitude, toTask.latitude, toTask.longitude);
  return d >= MIN_SEGMENT_DISTANCE_M;
}

function buildLabelContent(icon: SegmentIconResult, duration?: string, distance?: string): string {
  const primary = duration || distance || '';
  const secondary = duration && distance ? distance : '';
  const escIcon = icon.icon.replace(/[<>&]/g, '');
  const primarySpan = `<span>${primary}</span>`;
  const secondarySpan = secondary ? `<span style="color:#6b7280;font-weight:400"> · ${secondary}</span>` : '';
  return `<span style="margin-right:2px">${escIcon}</span>${primarySpan}${secondarySpan}`;
}

export function createSegmentLabel({
  fromTask, toTask, legIndex, duration, distance, mode, segments,
}: CreateSegmentLabelArgs): any | null {
  if (!shouldShowSegmentLabel(fromTask, toTask)) return null;

  const lat = (fromTask.latitude! + toTask.latitude!) / 2;
  const lon = (fromTask.longitude! + toTask.longitude!) / 2;
  const icon = getSegmentIcon(mode, segments);
  const iconContent = buildLabelContent(icon, duration, distance);

  const verticalOffset = legIndex % 2 === 0 ? -14 : 14;

  return new window.ymaps.Placemark(
    [lat, lon],
    {
      iconContent,
      balloonContentHeader: `${fromTask.title} → ${toTask.title}`,
      balloonContentBody:
        '<div style="font-size:13px">' +
        `<div>Тип: <b>${icon.label}</b></div>` +
        `<div>Время: <b>${duration || '—'}</b></div>` +
        `<div>Расстояние: <b>${distance || '—'}</b></div>` +
        '</div>',
      hintContent: `${fromTask.title} → ${toTask.title}: ${duration || distance || '—'}`,
    },
    {
      iconLayout: getLabelLayout(),
      iconShape: { type: 'Rectangle', coordinates: [[-40, -12], [40, 12]] },
      iconOffset: [0, verticalOffset],
      zIndex: 600,
      zIndexHover: 610,
    },
  );
}
