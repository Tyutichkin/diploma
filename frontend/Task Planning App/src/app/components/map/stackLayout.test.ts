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
      t('b', 55.750001, 37.620001),
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
      [-16, -16],
      [-16, -16 - 36],
    ]);
    expect(info.anchorLineHeightPx).toBe(72);
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
