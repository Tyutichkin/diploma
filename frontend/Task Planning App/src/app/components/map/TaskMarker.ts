/* eslint-disable @typescript-eslint/no-explicit-any */
import { Task } from '../../types/task';

interface CreateTaskMarkerArgs {
  task: Task;
  number: number;
  iconOffset: [number, number];
  isStack: boolean;
}

let cachedLayout: any = null;

function getMarkerLayout(): any {
  if (cachedLayout) return cachedLayout;
  cachedLayout = window.ymaps.templateLayoutFactory.createClass(
    [
      '<div class="task-marker"',
      ' style="display:flex;align-items:center;justify-content:center;',
      'width:32px;height:32px;border-radius:50%;background:#ffffff;',
      'border:2px solid #2563eb;color:#2563eb;font-weight:700;font-size:14px;',
      'box-shadow:0 1px 3px rgba(0,0,0,0.25);user-select:none;',
      'font-family:inherit;line-height:1;">',
      '$[properties.iconContent]',
      '</div>',
    ].join(''),
  );
  return cachedLayout;
}

export function createTaskMarker({ task, number, iconOffset, isStack }: CreateTaskMarkerArgs): any {
  const balloonHeader = `#${number} ${task.title}`;
  const balloonBody = [
    task.address ? `<div style="color:#555">${task.address}</div>` : '',
    task.duration != null
      ? `<div style="color:#555;margin-top:2px">Длительность: ${task.duration} мин</div>`
      : '',
    (task.windowStartDate || task.windowStartTime || task.windowEndDate || task.windowEndTime)
      ? `<div style="color:#555">Окно: ${[task.windowStartDate, task.windowStartTime].filter(Boolean).join(' ') || '—'} – ${[task.windowEndDate, task.windowEndTime].filter(Boolean).join(' ') || '—'}</div>`
      : '',
  ].filter(Boolean).join('');

  return new window.ymaps.Placemark(
    [task.latitude!, task.longitude!],
    {
      iconContent: String(number),
      balloonContentHeader: balloonHeader,
      balloonContentBody: balloonBody,
      hintContent: task.title,
    },
    {
      iconLayout: getMarkerLayout(),
      iconShape: { type: 'Circle', coordinates: [0, 0], radius: 16 },
      iconOffset,
      zIndex: isStack ? 720 : 700,
      zIndexHover: 800,
    },
  );
}

interface CreateStackAnchorArgs {
  lat: number;
  lon: number;
  lineHeightPx: number;
}

let cachedAnchorLayout: any = null;
let cachedAnchorLineHeight = -1;

function getAnchorLayout(lineHeightPx: number): any {
  if (cachedAnchorLayout && cachedAnchorLineHeight === lineHeightPx) return cachedAnchorLayout;
  cachedAnchorLayout = window.ymaps.templateLayoutFactory.createClass(
    [
      '<div style="position:relative;width:6px;height:6px;">',
      '<div style="position:absolute;left:50%;bottom:3px;transform:translateX(-50%);',
      `width:1px;height:${lineHeightPx}px;background:#9ca3af;"></div>`,
      '<div style="position:absolute;inset:0;border-radius:50%;background:#9ca3af;',
      'box-shadow:0 0 0 1px #ffffff;"></div>',
      '</div>',
    ].join(''),
  );
  cachedAnchorLineHeight = lineHeightPx;
  return cachedAnchorLayout;
}

export function createStackAnchor({ lat, lon, lineHeightPx }: CreateStackAnchorArgs): any {
  return new window.ymaps.Placemark(
    [lat, lon],
    {},
    {
      iconLayout: getAnchorLayout(lineHeightPx),
      iconShape: { type: 'Circle', coordinates: [0, 0], radius: 3 },
      iconOffset: [-3, -3],
      zIndex: 650,
      hasBalloon: false,
      hasHint: false,
    },
  );
}
