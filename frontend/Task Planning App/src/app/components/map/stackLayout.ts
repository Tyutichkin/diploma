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
