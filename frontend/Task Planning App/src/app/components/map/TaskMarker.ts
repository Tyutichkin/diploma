/* eslint-disable @typescript-eslint/no-explicit-any */
import { Task } from '../../types/task';

const MARKER_SIZE = 32;
const MARKER_RADIUS = MARKER_SIZE / 2;
const MARKER_FONT_SIZE = 14;
const MARKER_BORDER_WIDTH = 2;
const ANCHOR_SIZE = 6;
const ANCHOR_RADIUS = ANCHOR_SIZE / 2;

const Z_ANCHOR = 650;
const Z_MARKER = 700;
const Z_STACK_MARKER = 720;
const Z_MARKER_HOVER = 800;

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
    `<div class="task-marker" style="display:flex;align-items:center;justify-content:center;` +
    `width:${MARKER_SIZE}px;height:${MARKER_SIZE}px;border-radius:50%;background:#ffffff;` +
    `border:${MARKER_BORDER_WIDTH}px solid #2563eb;color:#2563eb;font-weight:700;font-size:${MARKER_FONT_SIZE}px;` +
    `box-shadow:0 1px 3px rgba(0,0,0,0.25);user-select:none;` +
    `font-family:inherit;line-height:1;">` +
    `$[properties.iconContent]` +
    `</div>`,
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

  // Кликабельная зона должна совпадать с визуальным положением кружка:
  // iconOffset сдвигает отрисовку, поэтому центр Circle тоже смещаем.
  const shapeCenter: [number, number] = [
    iconOffset[0] + MARKER_RADIUS,
    iconOffset[1] + MARKER_RADIUS,
  ];

  const placemark = new window.ymaps.Placemark(
    [task.latitude!, task.longitude!],
    {
      iconContent: String(number),
      balloonContentHeader: balloonHeader,
      balloonContentBody: balloonBody,
      hintContent: task.title,
    },
    {
      iconLayout: getMarkerLayout(),
      iconShape: { type: 'Circle', coordinates: shapeCenter, radius: MARKER_RADIUS },
      iconOffset,
      zIndex: isStack ? Z_STACK_MARKER : Z_MARKER,
      zIndexHover: Z_MARKER_HOVER,
    },
  );

  // Явно открываем балун при клике — на случай, если кастомный iconLayout
  // съедает дефолтное поведение, и чтобы клик по номеру задачи всегда показывал её данные.
  placemark.events.add('click', () => {
    placemark.balloon.open();
  });

  return placemark;
}

interface CreateStackAnchorArgs {
  lat: number;
  lon: number;
  lineHeightPx: number;
}

const anchorLayoutCache = new Map<number, any>();

function getAnchorLayout(lineHeightPx: number): any {
  let layout = anchorLayoutCache.get(lineHeightPx);
  if (layout) return layout;
  layout = window.ymaps.templateLayoutFactory.createClass(
    [
      '<div style="position:relative;width:6px;height:6px;">',
      '<div style="position:absolute;left:50%;bottom:3px;transform:translateX(-50%);',
      `width:1px;height:${lineHeightPx}px;background:#9ca3af;"></div>`,
      '<div style="position:absolute;inset:0;border-radius:50%;background:#9ca3af;',
      'box-shadow:0 0 0 1px #ffffff;"></div>',
      '</div>',
    ].join(''),
  );
  anchorLayoutCache.set(lineHeightPx, layout);
  return layout;
}

export function createStackAnchor({ lat, lon, lineHeightPx }: CreateStackAnchorArgs): any {
  return new window.ymaps.Placemark(
    [lat, lon],
    {},
    {
      iconLayout: getAnchorLayout(lineHeightPx),
      iconShape: { type: 'Circle', coordinates: [0, 0], radius: ANCHOR_RADIUS },
      iconOffset: [-ANCHOR_RADIUS, -ANCHOR_RADIUS],
      zIndex: Z_ANCHOR,
      hasBalloon: false,
      hasHint: false,
    },
  );
}
