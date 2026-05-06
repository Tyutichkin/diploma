# План реализации: UX-редизайн карты маршрута

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Сделать numbered-маркеры задач, плашки сегментов и легенду карты понятными и не перекрывающимися; добавить стек для совпадающих координат; различать стили линий маршрута по режиму перемещения.

**Architecture:** Выносим из `MapView.tsx` (446 строк) подсистемы в `src/app/components/map/*` — чистые функции (stack, route style, segment icon) тестируются юнитами; ymaps-фабрики (`TaskMarker`, `SegmentLabel`) и React-компоненты (`MapLegend`, `RouteSegmentsPanel`) тестируем визуально. Карта остаётся на Yandex Maps JS API 2.1, кастомизация через `templateLayoutFactory.createClass`.

**Tech Stack:** React 18, TypeScript strict, Vitest (jsdom), Yandex Maps JS API 2.1, Tailwind CSS 4, lucide-react.

**Спека:** `docs/superpowers/specs/2026-05-06-map-ux-redesign-design.md`

**База путей:** все пути ниже — относительно `frontend/Task Planning App/`. Команды pnpm/git выполнять из `frontend/Task Planning App/`.

---

## Структура файлов

| Файл | Что делает |
|---|---|
| `src/app/components/map/stackLayout.ts` (новый) | Группирует задачи по координатам, считает offset для стека |
| `src/app/components/map/routeStyle.ts` (новый) | Резолвит ymaps-опции линии маршрута по `TransportMode` |
| `src/app/components/map/segmentTransportIcon.ts` (новый) | Резолвит иконку транспорта для плашки сегмента |
| `src/app/components/map/geo.ts` (новый) | `haversineMeters(lat1, lon1, lat2, lon2)` — для anti-overlap фильтра |
| `src/app/components/map/TaskMarker.ts` (новый) | Фабрика numbered Placemark с кастомным HTML-layout |
| `src/app/components/map/SegmentLabel.ts` (новый) | Фабрика плашки сегмента с кастомным HTML-layout |
| `src/app/components/map/MapLegend.tsx` (новый) | HTML-оверлей легенды поверх карты (desktop ряд / mobile кнопка) |
| `src/app/components/map/RouteSegmentsPanel.tsx` (новый) | Панель деталей сегментов, расширенная на все режимы |
| `src/app/components/MapView.tsx` (изменяется) | Использует новые модули, оставляет оркестрацию |

**Тесты:**
- `src/app/components/map/stackLayout.test.ts`
- `src/app/components/map/routeStyle.test.ts`
- `src/app/components/map/segmentTransportIcon.test.ts`
- `src/app/components/map/geo.test.ts`

---

## Замечание про коммиты

Каждая задача завершается коммитом. Если ваша политика — squash в конце, можете не коммитить пошагово, а собрать всё одним коммитом после Task 12. Команды коммита приведены для дисциплины TDD.

---

## Task 1: Хелпер haversine

**Files:**
- Create: `src/app/components/map/geo.ts`
- Test: `src/app/components/map/geo.test.ts`

- [ ] **Step 1: Написать падающий тест**

Создать `src/app/components/map/geo.test.ts`:

```ts
import { describe, expect, it } from 'vitest';
import { haversineMeters } from './geo';

describe('haversineMeters', () => {
  it('возвращает 0 для одной и той же точки', () => {
    expect(haversineMeters(55.75, 37.62, 55.75, 37.62)).toBe(0);
  });

  it('считает короткое расстояние в Москве с погрешностью <2 м', () => {
    // Тверская 1 → Тверская 3, ~70 м реально
    const d = haversineMeters(55.7558, 37.6173, 55.7565, 37.6173);
    expect(d).toBeGreaterThan(60);
    expect(d).toBeLessThan(90);
  });

  it('считает длинное расстояние Москва-СПб ~635 км', () => {
    const d = haversineMeters(55.7558, 37.6173, 59.9343, 30.3351);
    expect(d).toBeGreaterThan(630_000);
    expect(d).toBeLessThan(640_000);
  });

  it('возвращает число для антиподов', () => {
    const d = haversineMeters(0, 0, 0, 180);
    expect(d).toBeGreaterThan(20_000_000);
  });
});
```

- [ ] **Step 2: Запустить тест и убедиться, что падает**

```bash
pnpm test -- src/app/components/map/geo.test.ts
```

Ожидание: FAIL — модуль `./geo` не существует.

- [ ] **Step 3: Реализовать функцию**

Создать `src/app/components/map/geo.ts`:

```ts
const EARTH_RADIUS_M = 6_371_000;

export function haversineMeters(lat1: number, lon1: number, lat2: number, lon2: number): number {
  const toRad = (deg: number) => (deg * Math.PI) / 180;
  const dLat = toRad(lat2 - lat1);
  const dLon = toRad(lon2 - lon1);
  const a =
    Math.sin(dLat / 2) ** 2 +
    Math.cos(toRad(lat1)) * Math.cos(toRad(lat2)) * Math.sin(dLon / 2) ** 2;
  return 2 * EARTH_RADIUS_M * Math.asin(Math.min(1, Math.sqrt(a)));
}
```

- [ ] **Step 4: Запустить тест и убедиться, что прошёл**

```bash
pnpm test -- src/app/components/map/geo.test.ts
```

Ожидание: PASS, 4 теста.

- [ ] **Step 5: Коммит**

```bash
git add "frontend/Task Planning App/src/app/components/map/geo.ts" "frontend/Task Planning App/src/app/components/map/geo.test.ts"
git commit -m "feat(map): добавлен haversine-хелпер для anti-overlap фильтра"
```

---

## Task 2: stackLayout — группировка совпадающих координат

**Files:**
- Create: `src/app/components/map/stackLayout.ts`
- Test: `src/app/components/map/stackLayout.test.ts`

- [ ] **Step 1: Написать падающий тест**

Создать `src/app/components/map/stackLayout.test.ts`:

```ts
import { describe, expect, it } from 'vitest';
import { Task } from '../../types/task';
import { computeStackPlacements, groupTasksByCoord, TaskGroup } from './stackLayout';

const t = (id: string, lat: number, lon: number): Task => ({
  id,
  title: id,
  address: `addr ${id}`,
  latitude: lat,
  longitude: lon,
});

describe('groupTasksByCoord', () => {
  it('возвращает пустой массив для пустого входа', () => {
    expect(groupTasksByCoord([])).toEqual([]);
  });

  it('одна задача — одна группа размера 1', () => {
    const groups = groupTasksByCoord([t('a', 55.75, 37.62)]);
    expect(groups).toHaveLength(1);
    expect(groups[0].entries).toHaveLength(1);
    expect(groups[0].entries[0].index).toBe(0);
    expect(groups[0].lat).toBeCloseTo(55.75);
    expect(groups[0].lon).toBeCloseTo(37.62);
  });

  it('задачи с разницей координат < 1e-5 попадают в одну группу', () => {
    const groups = groupTasksByCoord([
      t('a', 55.75000, 37.62000),
      t('b', 55.75001, 37.62001), // 5-й знак отличается, но округляется до того же
    ]);
    expect(groups).toHaveLength(1);
    expect(groups[0].entries.map((e) => e.task.id)).toEqual(['a', 'b']);
  });

  it('сохраняет исходные индексы задач', () => {
    const groups = groupTasksByCoord([
      t('a', 55.75, 37.62),
      t('b', 55.80, 37.65),
      t('c', 55.75, 37.62),
    ]);
    const same = groups.find((g) => g.entries.length === 2)!;
    expect(same.entries.map((e) => e.index)).toEqual([0, 2]);
  });
});

describe('computeStackPlacements', () => {
  it('одна задача — нулевой offset, без anchor', () => {
    const group: TaskGroup = {
      key: 'k',
      lat: 0, lon: 0,
      entries: [{ task: t('a', 0, 0), index: 0 }],
    };
    const info = computeStackPlacements(group);
    expect(info.placements).toHaveLength(1);
    expect(info.placements[0].iconOffset).toEqual([-16, -16]);
    expect(info.anchorLineHeightPx).toBeNull();
  });

  it('две задачи — стек с anchor высотой 72', () => {
    const group: TaskGroup = {
      key: 'k', lat: 0, lon: 0,
      entries: [
        { task: t('a', 0, 0), index: 0 },
        { task: t('b', 0, 0), index: 1 },
      ],
    };
    const info = computeStackPlacements(group);
    expect(info.placements.map((p) => p.iconOffset)).toEqual([
      [-16, -16],       // нижний (i=0)
      [-16, -16 - 36],  // верхний (i=1)
    ]);
    expect(info.anchorLineHeightPx).toBe(72); // 36 * 2
  });

  it('три задачи — стек', () => {
    const group: TaskGroup = {
      key: 'k', lat: 0, lon: 0,
      entries: [
        { task: t('a', 0, 0), index: 5 },
        { task: t('b', 0, 0), index: 6 },
        { task: t('c', 0, 0), index: 7 },
      ],
    };
    const info = computeStackPlacements(group);
    expect(info.placements).toHaveLength(3);
    expect(info.placements[0].iconOffset[1]).toBe(-16);
    expect(info.placements[1].iconOffset[1]).toBe(-52);
    expect(info.placements[2].iconOffset[1]).toBe(-88);
    expect(info.placements[0].taskIndex).toBe(5);
    expect(info.placements[2].taskIndex).toBe(7);
    expect(info.anchorLineHeightPx).toBe(108);
  });
});
```

- [ ] **Step 2: Запустить тест и убедиться, что падает**

```bash
pnpm test -- src/app/components/map/stackLayout.test.ts
```

Ожидание: FAIL — модуль не существует.

- [ ] **Step 3: Реализовать модуль**

Создать `src/app/components/map/stackLayout.ts`:

```ts
import { Task } from '../../types/task';

export interface TaskWithIndex {
  task: Task;
  index: number;
}

export interface TaskGroup {
  key: string;
  lat: number;
  lon: number;
  entries: TaskWithIndex[];
}

export interface StackPlacement {
  taskIndex: number;
  iconOffset: [number, number];
}

export interface StackRenderInfo {
  placements: StackPlacement[];
  anchorLineHeightPx: number | null;
}

const MARKER_SIZE = 32;
const MARKER_GAP = 4;
const STACK_STEP = MARKER_SIZE + MARKER_GAP; // 36

export function groupTasksByCoord(validTasks: Task[]): TaskGroup[] {
  const map = new Map<string, TaskGroup>();
  validTasks.forEach((task, index) => {
    if (task.latitude === undefined || task.longitude === undefined) return;
    const key = `${task.latitude.toFixed(5)}|${task.longitude.toFixed(5)}`;
    let group = map.get(key);
    if (!group) {
      group = { key, lat: task.latitude, lon: task.longitude, entries: [] };
      map.set(key, group);
    }
    group.entries.push({ task, index });
  });
  return Array.from(map.values());
}

export function computeStackPlacements(group: TaskGroup): StackRenderInfo {
  const placements: StackPlacement[] = group.entries.map((entry, i) => ({
    taskIndex: entry.index,
    iconOffset: [-MARKER_SIZE / 2, -MARKER_SIZE / 2 - STACK_STEP * i],
  }));
  const anchorLineHeightPx =
    group.entries.length > 1 ? STACK_STEP * group.entries.length : null;
  return { placements, anchorLineHeightPx };
}
```

- [ ] **Step 4: Запустить тест и убедиться, что прошёл**

```bash
pnpm test -- src/app/components/map/stackLayout.test.ts
```

Ожидание: PASS — 6 тестов.

- [ ] **Step 5: Коммит**

```bash
git add "frontend/Task Planning App/src/app/components/map/stackLayout.ts" "frontend/Task Planning App/src/app/components/map/stackLayout.test.ts"
git commit -m "feat(map): группировка совпадающих координат и расчёт stack offsets"
```

---

## Task 3: routeStyle — стили линии по режиму

**Files:**
- Create: `src/app/components/map/routeStyle.ts`
- Test: `src/app/components/map/routeStyle.test.ts`

- [ ] **Step 1: Написать падающий тест**

Создать `src/app/components/map/routeStyle.test.ts`:

```ts
import { describe, expect, it } from 'vitest';
import { getRouteStyleOptions } from './routeStyle';

describe('getRouteStyleOptions', () => {
  it('pedestrian — пунктирная зелёная', () => {
    const o = getRouteStyleOptions('pedestrian');
    expect(o.routeActiveStrokeColor).toBe('#16a34a');
    expect(o.routeActiveStrokeWidth).toBe(4);
    expect(o.routeActiveStrokeStyle).toBe('dash');
  });

  it('auto — сплошная синяя', () => {
    const o = getRouteStyleOptions('auto');
    expect(o.routeActiveStrokeColor).toBe('#3b82f6');
    expect(o.routeActiveStrokeWidth).toBe(4);
    expect(o.routeActiveStrokeStyle).toBe('solid');
  });

  it('masstransit — без переопределений (дефолты ymaps)', () => {
    const o = getRouteStyleOptions('masstransit');
    expect(o.routeActiveStrokeColor).toBeUndefined();
    expect(o.routeActiveStrokeStyle).toBeUndefined();
  });

  it('boundsAutoApply и wayPointVisible выставлены везде', () => {
    (['pedestrian', 'auto', 'masstransit'] as const).forEach((mode) => {
      const o = getRouteStyleOptions(mode);
      expect(o.boundsAutoApply).toBe(true);
      expect(o.wayPointVisible).toBe(false);
    });
  });
});
```

- [ ] **Step 2: Запустить тест и убедиться, что падает**

```bash
pnpm test -- src/app/components/map/routeStyle.test.ts
```

Ожидание: FAIL — модуль не существует.

- [ ] **Step 3: Реализовать модуль**

Создать `src/app/components/map/routeStyle.ts`:

```ts
import { TransportMode } from '../../types/transport';

export interface RouteStyleOptions {
  boundsAutoApply: true;
  wayPointVisible: false;
  routeActiveStrokeWidth?: number;
  routeActiveStrokeColor?: string;
  routeActiveStrokeStyle?: 'solid' | 'dash';
  routeStrokeWidth?: number;
  routeStrokeColor?: string;
}

export function getRouteStyleOptions(mode: TransportMode): RouteStyleOptions {
  if (mode === 'pedestrian') {
    return {
      boundsAutoApply: true,
      wayPointVisible: false,
      routeActiveStrokeWidth: 4,
      routeActiveStrokeColor: '#16a34a',
      routeActiveStrokeStyle: 'dash',
      routeStrokeWidth: 2,
      routeStrokeColor: '#86efac',
    };
  }
  if (mode === 'auto') {
    return {
      boundsAutoApply: true,
      wayPointVisible: false,
      routeActiveStrokeWidth: 4,
      routeActiveStrokeColor: '#3b82f6',
      routeActiveStrokeStyle: 'solid',
      routeStrokeWidth: 2,
      routeStrokeColor: '#93c5fd',
    };
  }
  // masstransit — оставляем дефолты ymaps (фирменные цвета линий метро/автобусов)
  return { boundsAutoApply: true, wayPointVisible: false };
}
```

- [ ] **Step 4: Запустить тест и убедиться, что прошёл**

```bash
pnpm test -- src/app/components/map/routeStyle.test.ts
```

Ожидание: PASS — 4 теста.

- [ ] **Step 5: Коммит**

```bash
git add "frontend/Task Planning App/src/app/components/map/routeStyle.ts" "frontend/Task Planning App/src/app/components/map/routeStyle.test.ts"
git commit -m "feat(map): резолвер стилей линии маршрута по режиму перемещения"
```

---

## Task 4: segmentTransportIcon — иконка для плашки сегмента

**Files:**
- Create: `src/app/components/map/segmentTransportIcon.ts`
- Test: `src/app/components/map/segmentTransportIcon.test.ts`

- [ ] **Step 1: Написать падающий тест**

Создать `src/app/components/map/segmentTransportIcon.test.ts`:

```ts
import { describe, expect, it } from 'vitest';
import { RouteSegment } from '../../types/transport';
import { getSegmentIcon } from './segmentTransportIcon';

const segPedestrian = (duration = '5 мин'): RouteSegment => ({ kind: 'pedestrian', duration });
const segTransport = (
  type: string,
  duration = '5 мин',
): RouteSegment => ({
  kind: 'transport',
  duration,
  transports: [{ name: '1', type }],
});

describe('getSegmentIcon', () => {
  it('pedestrian-режим всегда отдаёт иконку пешехода', () => {
    expect(getSegmentIcon('pedestrian', []).icon).toBe('🚶');
    expect(getSegmentIcon('pedestrian', [segTransport('subway')]).icon).toBe('🚶');
  });

  it('auto-режим всегда отдаёт иконку машины', () => {
    expect(getSegmentIcon('auto', []).icon).toBe('🚗');
  });

  it('masstransit без сегментов транспорта — пешеход', () => {
    expect(getSegmentIcon('masstransit', [segPedestrian()]).icon).toBe('🚶');
  });

  it('masstransit: subway имеет наивысший приоритет над автобусом', () => {
    const r = getSegmentIcon('masstransit', [
      segTransport('bus', '20 мин'),
      segTransport('subway', '5 мин'),
    ]);
    expect(r.icon).toBe('М');
    expect(r.type).toBe('subway');
  });

  it('masstransit: при отсутствии метро доминирует наземный транспорт по типу', () => {
    const r = getSegmentIcon('masstransit', [
      segTransport('bus', '5 мин'),
      segTransport('tram', '10 мин'),
    ]);
    // tram приоритетнее bus
    expect(r.type).toBe('tram');
    expect(r.icon).toBe('🚋');
  });

  it('masstransit: rail приоритетнее tram', () => {
    const r = getSegmentIcon('masstransit', [
      segTransport('tram', '10 мин'),
      segTransport('rail', '5 мин'),
    ]);
    expect(r.type).toBe('rail');
  });
});
```

- [ ] **Step 2: Запустить тест и убедиться, что падает**

```bash
pnpm test -- src/app/components/map/segmentTransportIcon.test.ts
```

Ожидание: FAIL.

- [ ] **Step 3: Реализовать модуль**

Создать `src/app/components/map/segmentTransportIcon.ts`:

```ts
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

  // masstransit — выбираем тип с максимальным приоритетом среди присутствующих
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
```

- [ ] **Step 4: Запустить тест и убедиться, что прошёл**

```bash
pnpm test -- src/app/components/map/segmentTransportIcon.test.ts
```

Ожидание: PASS — 6 тестов.

- [ ] **Step 5: Коммит**

```bash
git add "frontend/Task Planning App/src/app/components/map/segmentTransportIcon.ts" "frontend/Task Planning App/src/app/components/map/segmentTransportIcon.test.ts"
git commit -m "feat(map): резолвер иконки транспорта для плашки сегмента"
```

---

## Task 5: TaskMarker — фабрика numbered Placemark

Эта задача без unit-тестов: фабрика возвращает ymaps Placemark, его надо рендерить визуально. Делаем «implement → ручная сборка → коммит».

**Files:**
- Create: `src/app/components/map/TaskMarker.ts`

- [ ] **Step 1: Написать модуль**

Создать `src/app/components/map/TaskMarker.ts`:

```ts
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
```

> Замечание: `cachedAnchorLayout` пересоздаётся при изменении высоты — это редкий случай (разные стеки), но мы кэшируем последний.

- [ ] **Step 2: Проверить TypeScript**

```bash
pnpm tsc --noEmit
```

Ожидание: ноль ошибок в новом файле. Если что — поправить.

- [ ] **Step 3: Коммит**

```bash
git add "frontend/Task Planning App/src/app/components/map/TaskMarker.ts"
git commit -m "feat(map): фабрика numbered task marker и якоря стека"
```

---

## Task 6: SegmentLabel — фабрика плашки сегмента

**Files:**
- Create: `src/app/components/map/SegmentLabel.ts`

- [ ] **Step 1: Написать модуль**

Создать `src/app/components/map/SegmentLabel.ts`:

```ts
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

  // чередование: чётный leg — над линией, нечётный — под
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
```

- [ ] **Step 2: Проверить TypeScript**

```bash
pnpm tsc --noEmit
```

Ожидание: ноль ошибок в новом файле.

- [ ] **Step 3: Коммит**

```bash
git add "frontend/Task Planning App/src/app/components/map/SegmentLabel.ts"
git commit -m "feat(map): фабрика плашки сегмента маршрута с чередованием сторон"
```

---

## Task 7: MapLegend — компонент легенды

**Files:**
- Create: `src/app/components/map/MapLegend.tsx`

Юнит-тесты не пишем (визуальный компонент, jsdom не покрывает Tailwind).

- [ ] **Step 1: Написать компонент**

Создать `src/app/components/map/MapLegend.tsx`:

```tsx
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

  // закрытие по клику вне popover на мобильном
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
```

- [ ] **Step 2: Проверить TypeScript**

```bash
pnpm tsc --noEmit
```

Ожидание: ноль ошибок.

- [ ] **Step 3: Коммит**

```bash
git add "frontend/Task Planning App/src/app/components/map/MapLegend.tsx"
git commit -m "feat(map): компонент легенды карты с адаптивностью"
```

---

## Task 8: RouteSegmentsPanel — панель деталей сегментов

Выносим существующую панель из `MapView.tsx` и расширяем её на все режимы.

**Files:**
- Create: `src/app/components/map/RouteSegmentsPanel.tsx`

- [ ] **Step 1: Написать компонент**

Создать `src/app/components/map/RouteSegmentsPanel.tsx`:

```tsx
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
```

- [ ] **Step 2: Проверить TypeScript**

```bash
pnpm tsc --noEmit
```

Ожидание: ноль ошибок.

- [ ] **Step 3: Коммит**

```bash
git add "frontend/Task Planning App/src/app/components/map/RouteSegmentsPanel.tsx"
git commit -m "feat(map): вынесена панель деталей сегментов, расширена на все режимы"
```

---

## Task 9: Интегрировать новые модули в MapView

Это самый объёмный шаг. Делаем разом, потому что половинная замена не работает (старые маркеры конфликтуют с новыми).

**Files:**
- Modify: `src/app/components/MapView.tsx`

- [ ] **Step 1: Сохранить координаты сегментов для onLegFocus**

В состоянии `MapView` нужны координаты пар задач, чтобы по клику в `RouteSegmentsPanel` сделать `setBounds`. Расширим `RouteLeg` локальным интерфейсом без правки `types/transport.ts`.

В `MapView.tsx` добавить рядом с импортами:

```ts
import { groupTasksByCoord, computeStackPlacements } from './map/stackLayout';
import { getRouteStyleOptions } from './map/routeStyle';
import { createTaskMarker, createStackAnchor } from './map/TaskMarker';
import { createSegmentLabel } from './map/SegmentLabel';
import { MapLegend } from './map/MapLegend';
import { RouteSegmentsPanel } from './map/RouteSegmentsPanel';

interface LegBounds {
  from: [number, number];
  to: [number, number];
}
```

И добавить новое состояние:

```ts
const [legBounds, setLegBounds] = useState<LegBounds[]>([]);
```

- [ ] **Step 2: Заменить блок построения маркеров задач**

В `MapView.tsx` найти текущий блок (строки ~106–147 в исходном файле):

```ts
    // группируем по координатам, чтобы маркеры не перекрывались
    const tasksByCoord = new Map<string, { task: Task; index: number }[]>();
    validTasks.forEach((task, index) => {
      const key = `${task.latitude}|${task.longitude}`;
      if (!tasksByCoord.has(key)) tasksByCoord.set(key, []);
      tasksByCoord.get(key)!.push({ task, index });
    });

    tasksByCoord.forEach((entries) => {
      const isSingle = entries.length === 1;
      const numbers = entries.map((e) => e.index + 1);
      const iconContent = numbers.length > 2 ? '…' : numbers.join(',');

      const balloonHeader = isSingle
        ? `#${numbers[0]} ${entries[0].task.title}`
        : `#${iconContent} — ${entries.length} задачи по одному адресу`;

      const balloonBody = entries
        .map(({ task, index }) =>
          [
            isSingle ? '' : `<div style="font-weight:600;margin-top:8px">#${index + 1} ${task.title}</div>`,
            `<div style="color:#555">${task.address}</div>`,
            task.duration != null ? `<div style="color:#555;margin-top:2px">Длительность: ${task.duration} мин</div>` : '',
            (task.windowStartDate || task.windowStartTime || task.windowEndDate || task.windowEndTime)
              ? `<div style="color:#555">Окно: ${[task.windowStartDate, task.windowStartTime].filter(Boolean).join(' ') || '—'} – ${[task.windowEndDate, task.windowEndTime].filter(Boolean).join(' ') || '—'}</div>`
              : '',
          ]
            .filter(Boolean)
            .join(''),
        )
        .join('<hr style="margin:6px 0;border-color:#ddd">');

      const hintContent = isSingle
        ? entries[0].task.title
        : `Задачи ${iconContent}: ${entries.map((e) => e.task.title).join(', ')}`;

      const placemark = new window.ymaps.Placemark(
        [entries[0].task.latitude!, entries[0].task.longitude!],
        { iconContent, balloonContentHeader: balloonHeader, balloonContentBody: balloonBody, hintContent },
        { preset: isSingle ? 'islands#blueCircleIcon' : 'islands#orangeCircleIcon' },
      );
      map.geoObjects.add(placemark);
    });
```

Заменить целиком на:

```ts
    const groups = groupTasksByCoord(validTasks);
    groups.forEach((group) => {
      const info = computeStackPlacements(group);
      if (info.anchorLineHeightPx !== null) {
        map.geoObjects.add(
          createStackAnchor({
            lat: group.lat,
            lon: group.lon,
            lineHeightPx: info.anchorLineHeightPx,
          }),
        );
      }
      info.placements.forEach((p) => {
        const entry = group.entries.find((e) => e.index === p.taskIndex)!;
        map.geoObjects.add(
          createTaskMarker({
            task: entry.task,
            number: entry.index + 1,
            iconOffset: p.iconOffset,
            isStack: info.anchorLineHeightPx !== null,
          }),
        );
      });
    });
```

- [ ] **Step 3: Заменить блок построения мультироута**

В `MapView.tsx` найти блок:

```ts
    if (routeOptimized && validTasks.length > 1) {
      const referencePoints = validTasks.map((t) => [t.latitude!, t.longitude!]);

      // в masstransit цвета линий задаются ymaps (метро, автобус и т.д.) — не переопределяем
      const routeOptions =
        transportMode === 'masstransit'
          ? { boundsAutoApply: true, wayPointVisible: false }
          : {
              boundsAutoApply: true,
              wayPointVisible: false,
              routeActiveStrokeWidth: 4,
              routeActiveStrokeColor: transportMode === 'auto' ? '#3b82f6' : '#16a34a',
              routeActiveStrokeStyle: 'solid',
              routeStrokeWidth: 2,
              routeStrokeColor: transportMode === 'auto' ? '#93c5fd' : '#86efac',
            };
```

Заменить на:

```ts
    if (routeOptimized && validTasks.length > 1) {
      const referencePoints = validTasks.map((t) => [t.latitude!, t.longitude!]);
      const routeOptions = getRouteStyleOptions(transportMode);
```

- [ ] **Step 4: Заменить создание midPlacemark**

Внутри `multiRoute.model.events.add('requestsuccess', () => {...})` найти блок:

```ts
            // метка на середине отрезка — показывает время в пути
            if (legIndex < validTasks.length - 1) {
              const lat1 = validTasks[legIndex].latitude!;
              const lon1 = validTasks[legIndex].longitude!;
              const lat2 = validTasks[legIndex + 1].latitude!;
              const lon2 = validTasks[legIndex + 1].longitude!;

              const midPlacemark = new window.ymaps.Placemark(
                [(lat1 + lat2) / 2, (lon1 + lon2) / 2],
                {
                  iconContent: legDuration || '',
                  balloonContentHeader: `${fromTitle} &rarr; ${toTitle}`,
                  balloonContentBody:
                    `<div style="font-size:13px">` +
                    `<div>Время в пути: <b>${legDuration || '—'}</b></div>` +
                    `<div>Расстояние: <b>${legDistance || '—'}</b></div>` +
                    `</div>`,
                  hintContent: `${fromTitle} → ${toTitle}: ${legDuration || '—'}`,
                },
                { preset: 'islands#grayStretchyIcon' },
              );
              map.geoObjects.add(midPlacemark);
            }
```

(Сохраняем как раньше: код фактически уже завязан на `legSegments`, ниже по коду.)

Заменить на использование новой фабрики. Чтобы не дублировать сбор `segments`, делаем создание плашки **после** сбора `segments`. Перенесите этот блок в самый конец итерации leg, после `legs.push(...)`. Полный новый код итерации leg выглядит так — заменяем содержимое forEach `routes[0].getLegs().forEach(...)`:

```ts
          routes[0].getLegs().forEach((leg: any, legIndex: number) => {
            const fromTask = validTasks[legIndex];
            const toTask = validTasks[legIndex + 1];
            const fromTitle = fromTask?.title ?? `Точка ${legIndex + 1}`;
            const toTitle = toTask?.title ?? `Точка ${legIndex + 2}`;

            const legProps = leg.properties as any;
            const legDurationObj = legProps.get('duration');
            const legDistanceObj = legProps.get('distance');
            const legDuration: string | undefined =
              typeof legDurationObj === 'object' ? legDurationObj?.text : legDurationObj;
            const legDistance: string | undefined =
              typeof legDistanceObj === 'object' ? legDistanceObj?.text : legDistanceObj;

            const segments: RouteSegment[] = [];

            if (transportMode === 'masstransit') {
              leg.getSegments().forEach((seg: any) => {
                const props = seg.properties as any;
                const kind: 'pedestrian' | 'transport' =
                  props.get('type') === 'transport' ? 'transport' : 'pedestrian';
                const durationObj = props.get('duration');
                const distanceObj = props.get('distance');
                const duration: string | undefined =
                  typeof durationObj === 'object' ? durationObj?.text : durationObj;
                const distance: string | undefined =
                  typeof distanceObj === 'object' ? distanceObj?.text : distanceObj;

                if (kind === 'transport') {
                  const rawTransports: any[] = props.get('transports') ?? [];
                  const stopFrom: string | undefined =
                    props.get('departureStop')?.name ??
                    props.get('departureStop')?.properties?.get?.('name');
                  const stopTo: string | undefined =
                    props.get('arrivalStop')?.name ??
                    props.get('arrivalStop')?.properties?.get?.('name');

                  const transports: TransportInfo[] = rawTransports.map((t) => ({
                    name: t.name ?? t.number ?? '',
                    type: t.type ?? t.transportType ?? 'bus',
                    color: t.style?.color ?? t.color,
                  }));

                  const stopsCount: number | undefined = props.get('stopsCount') ?? undefined;
                  segments.push({ kind, duration, distance, transports, stopFrom, stopTo, stopsCount });
                } else {
                  segments.push({ kind, duration, distance });
                }
              });
            }

            // плашка-метка сегмента (после сбора segments — нужна для иконки в masstransit)
            if (fromTask && toTask) {
              const label = createSegmentLabel({
                fromTask, toTask, legIndex,
                duration: legDuration, distance: legDistance,
                mode: transportMode, segments,
              });
              if (label) map.geoObjects.add(label);
            }

            legs.push({ fromTitle, toTitle, duration: legDuration, distance: legDistance, segments });
          });
```

- [ ] **Step 5: Сохранить bounds сегментов и удалить старую панель**

Сразу после `setRouteLegs(legs); onRouteLegsChange?.(legs);` добавить:

```ts
          const bounds: LegBounds[] = [];
          for (let i = 0; i < validTasks.length - 1; i++) {
            const a = validTasks[i];
            const b = validTasks[i + 1];
            bounds.push({ from: [a.latitude!, a.longitude!], to: [b.latitude!, b.longitude!] });
          }
          setLegBounds(bounds);
```

Удалить из render-блока (внизу `MapView`) старый код панели `{showTransitPanel && (...)}` целиком — большой блок `<div className="flex-shrink-0 border border-blue-200 rounded-lg ...">`.

Также удалить state `panelOpen` и константу `showTransitPanel`, импорт `ChevronDown, ChevronUp` (теперь не нужен в `MapView.tsx`).

Заменить вместо удалённой панели на:

```tsx
            {routeOptimized && routeLegs.length > 0 && (
              <RouteSegmentsPanel
                legs={routeLegs}
                mode={transportMode}
                onLegFocus={(li) => {
                  if (!mapRef.current) return;
                  const lb = legBounds[li];
                  if (!lb) return;
                  const minLat = Math.min(lb.from[0], lb.to[0]);
                  const maxLat = Math.max(lb.from[0], lb.to[0]);
                  const minLon = Math.min(lb.from[1], lb.to[1]);
                  const maxLon = Math.max(lb.from[1], lb.to[1]);
                  mapRef.current.setBounds(
                    [[minLat, minLon], [maxLat, maxLon]],
                    { checkZoomRange: true, zoomMargin: 60 },
                  );
                }}
              />
            )}
```

- [ ] **Step 6: Добавить MapLegend в render**

Внутри `<div className="relative flex-1 rounded-lg overflow-hidden min-h-0">` после `<div ref={containerRef} ... />` и блоков loading/empty, добавить:

```tsx
              {!isLoading && !loadError && validTasks.length > 0 && <MapLegend />}
```

- [ ] **Step 7: Запустить TypeScript и тесты**

```bash
pnpm tsc --noEmit
pnpm test
```

Ожидание: ноль ошибок TS, все тесты зелёные.

- [ ] **Step 8: Запустить dev-сервер и провести смоук**

```bash
pnpm dev
```

Открыть `http://localhost:5173`. Войти, создать 4–5 задач с разными адресами:
- Проверить, что numbered-маркеры — белые круги с синей обводкой и номером
- Запустить оптимизацию — увидеть линию маршрута и плашки времени между точками
- Переключить режимы pedestrian/auto/masstransit — стиль линии меняется (пунктир/сплошная), на плашке меняется иконка
- Открыть панель «Детали маршрута» — она открывается во всех режимах
- Кликнуть по элементу панели — карта зумится на сегмент
- Создать 2 задачи с одним и тем же адресом — увидеть стек двух маркеров с серой ножкой к точке
- Открыть DevTools, переключить на ширину 375px — легенда становится круглой кнопкой «i», при клике открывается popover
- Загрузить сохранённый маршрут — всё рендерится

Если что-то ломается — исправить, прежде чем коммитить.

- [ ] **Step 9: Коммит**

```bash
git add "frontend/Task Planning App/src/app/components/MapView.tsx"
git commit -m "feat(map): интегрированы новые модули — stack, segment labels, легенда, расширенная панель"
```

---

## Task 10: Финальная верификация

- [ ] **Step 1: Полная проверка**

```bash
pnpm tsc --noEmit && pnpm test && pnpm build
```

Ожидание: всё зелёное.

- [ ] **Step 2: Проверить, что нет неиспользуемых импортов в MapView.tsx**

Открыть `src/app/components/MapView.tsx` и проверить:
- удалён импорт `ChevronDown, ChevronUp` если не используется
- удалён импорт `TRANSPORT_TYPE_META` если не используется
- удалены ре-экспорты, на которые больше никто не ссылается (но `RouteLeg, RouteSegment, TransportInfo, TransportMode, TRANSPORT_TYPE_META` оставить — они уже re-exported для обратной совместимости с `MainPage` и др., см. строки 14–16 исходного файла)

`pnpm tsc --noEmit` укажет на неиспользуемые, если в `tsconfig.json` включён `noUnusedLocals`. Проверить:

```bash
grep noUnusedLocals tsconfig*.json || echo "no flag, проверяем глазами"
```

- [ ] **Step 3: Финальный коммит при необходимости**

Если на шагах 1-2 что-то поправили:

```bash
git add -u "frontend/Task Planning App/src/app/components/MapView.tsx"
git commit -m "chore(map): чистка неиспользуемых импортов после рефакторинга"
```

Если изменений нет — ничего не коммитить.

---

## Self-review (выполнено автором плана)

**Покрытие спека:**

| Требование спека | Задача |
|---|---|
| 4. Архитектура (разбиение MapView) | Tasks 1–8 создают модули, Task 9 интегрирует |
| 5.1 TaskMarker (HTML-layout 32×32, обводка, тень, z-index 700) | Task 5 |
| 5.2 Stack (группировка, offsets, anchor с ножкой) | Task 2 + Task 5 (`createStackAnchor`) + Task 9 (Step 2) |
| 5.3 SegmentLabel (середина, чередование, фильтр <50м) | Tasks 1, 4, 6 |
| 5.4 routeStyle | Task 3 + Task 9 (Step 3) |
| 5.5 RouteSegmentsPanel (расширение на все режимы) | Task 8 + Task 9 (Steps 5) |
| 5.6 MapLegend (desktop ряд / mobile кнопка) | Task 7 + Task 9 (Step 6) |
| 5.7 z-index стек | Tasks 5, 6 (zIndex: 700/600/650) |
| 7. Тесты | Tasks 1, 2, 3, 4 |

**Placeholder scan:** проверено — нет TBD/TODO, все code-блоки полные.

**Type consistency:**
- `TaskGroup`, `StackPlacement`, `StackRenderInfo` — определены в Task 2, использованы в Task 9 Step 2 через `groupTasksByCoord`/`computeStackPlacements`
- `getRouteStyleOptions` возвращает `RouteStyleOptions` — структура совместима с тем, что MultiRoute принимал раньше (boundsAutoApply, wayPointVisible, routeActive*)
- `createTaskMarker` принимает `{ task, number, iconOffset, isStack }` — ровно так зовётся в Task 9 Step 2
- `createSegmentLabel` принимает `{ fromTask, toTask, legIndex, duration, distance, mode, segments }` — ровно так зовётся в Task 9 Step 4
- `RouteSegmentsPanel` — props `{ legs, mode, onLegFocus }`, используются в Task 9 Step 5
- `getSegmentIcon` возвращает `SegmentIconResult` с полями `{ type, icon, label }` — все используются

**Гэпы:** не выявлены.
