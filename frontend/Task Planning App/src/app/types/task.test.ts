import { describe, it, expect } from 'vitest';
import { Task, getWindowConflictIds } from './task';

function makeTask(id: string, overrides: Partial<Task> = {}): Task {
  return { id, title: id, ...overrides };
}

describe('getWindowConflictIds', () => {
  it('не помечает конфликтом узкие окна рядом с широким окном', () => {
    // Кейс: 3 задачи по 30 мин.
    //   A: 12:00–12:30  (ровно под свою длительность)
    //   B: 14:00–14:30  (ровно под свою длительность)
    //   C: 10:00–19:00  (широкое окно, не мешает A и B)
    // Все три могут быть выполнены последовательно без конфликта.
    const tasks: Task[] = [
      makeTask('a', {
        duration: 30,
        windowStartDate: '2026-04-19', windowStartTime: '12:00',
        windowEndDate: '2026-04-19', windowEndTime: '12:30',
      }),
      makeTask('b', {
        duration: 30,
        windowStartDate: '2026-04-19', windowStartTime: '14:00',
        windowEndDate: '2026-04-19', windowEndTime: '14:30',
      }),
      makeTask('c', {
        duration: 30,
        windowStartDate: '2026-04-19', windowStartTime: '10:00',
        windowEndDate: '2026-04-19', windowEndTime: '19:00',
      }),
    ];

    const conflicts = getWindowConflictIds(tasks);
    expect(conflicts.size).toBe(0);
  });

  it('помечает конфликтом пересекающиеся окна, в которые не уместить обе задачи', () => {
    // Кейс: 2 задачи по 30 мин с пересекающимися окнами.
    //   A: 12:00–12:30
    //   B: 12:15–12:45
    // A должна занять 12:00–12:30, B должна занять 12:15–12:45 — пересечение
    // 15 мин, и ни одна не может сдвинуться. Обе должны быть помечены.
    const tasks: Task[] = [
      makeTask('a', {
        duration: 30,
        windowStartDate: '2026-04-19', windowStartTime: '12:00',
        windowEndDate: '2026-04-19', windowEndTime: '12:30',
      }),
      makeTask('b', {
        duration: 30,
        windowStartDate: '2026-04-19', windowStartTime: '12:15',
        windowEndDate: '2026-04-19', windowEndTime: '12:45',
      }),
    ];

    const conflicts = getWindowConflictIds(tasks);
    expect(conflicts.has('a')).toBe(true);
    expect(conflicts.has('b')).toBe(true);
  });

  it('не помечает задачи без пересечения окон', () => {
    const tasks: Task[] = [
      makeTask('a', {
        duration: 30,
        windowStartDate: '2026-04-19', windowStartTime: '10:00',
        windowEndDate: '2026-04-19', windowEndTime: '10:30',
      }),
      makeTask('b', {
        duration: 30,
        windowStartDate: '2026-04-19', windowStartTime: '11:00',
        windowEndDate: '2026-04-19', windowEndTime: '11:30',
      }),
    ];

    const conflicts = getWindowConflictIds(tasks);
    expect(conflicts.size).toBe(0);
  });

  it('помечает задачу, чья длительность превышает окно', () => {
    const tasks: Task[] = [
      makeTask('a', {
        duration: 60,
        windowStartDate: '2026-04-19', windowStartTime: '12:00',
        windowEndDate: '2026-04-19', windowEndTime: '12:30',
      }),
    ];

    const conflicts = getWindowConflictIds(tasks);
    expect(conflicts.has('a')).toBe(true);
  });
});
