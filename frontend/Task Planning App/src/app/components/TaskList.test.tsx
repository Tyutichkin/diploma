import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import { TaskList } from './TaskList';
import { Task, taskHasAddress, isTaskWindowConflict, getWindowConflictIds } from '../types/task';

function makeTask(overrides: Partial<Task> & Pick<Task, 'id' | 'title'>): Task {
  return {
    duration: 30,
    ...overrides,
  };
}

function makeAddressTask(id: string, title: string): Task {
  return makeTask({
    id,
    title,
    address: `ул. Тестовая, ${id}`,
    latitude: 55.75,
    longitude: 37.61,
  });
}

function makeNoAddressTask(id: string, title: string): Task {
  return makeTask({ id, title });
}

const noop = vi.fn();

function renderTaskList(tasks: Task[]) {
  return render(
    <DndProvider backend={HTML5Backend}>
      <TaskList
        tasks={tasks}
        onEdit={noop}
        onDelete={noop}
        onToggleComplete={noop}
        onReorder={noop}
        onReorderEnd={noop}
        onSetRole={noop}
      />
    </DndProvider>,
  );
}

describe('taskHasAddress', () => {
  it('returns true when task has address, lat and lon', () => {
    expect(taskHasAddress({ id: '1', title: 't', address: 'ул. А', latitude: 55, longitude: 37, duration: 10 })).toBe(true);
  });

  it('returns false when address is empty string', () => {
    expect(taskHasAddress({ id: '1', title: 't', address: '', latitude: 55, longitude: 37, duration: 10 })).toBe(false);
  });

  it('returns false when address is undefined', () => {
    expect(taskHasAddress({ id: '1', title: 't', duration: 10 })).toBe(false);
  });

  it('returns false when latitude is undefined', () => {
    expect(taskHasAddress({ id: '1', title: 't', address: 'ул. А', longitude: 37, duration: 10 })).toBe(false);
  });

  it('returns false when longitude is undefined', () => {
    expect(taskHasAddress({ id: '1', title: 't', address: 'ул. А', latitude: 55, duration: 10 })).toBe(false);
  });
});

describe('TaskList — пустой список', () => {
  it('показывает заглушку при отсутствии задач', () => {
    renderTaskList([]);
    expect(screen.getByText(/Нет задач/)).toBeInTheDocument();
  });
});

describe('TaskList — группировка', () => {
  it('отображает заголовок группы «С адресом» при наличии таких задач', () => {
    const tasks = [makeAddressTask('a1', 'Задача А')];
    renderTaskList(tasks);
    expect(screen.getByText(/С адресом/i)).toBeInTheDocument();
  });

  it('отображает заголовок группы «Без адреса» при наличии таких задач', () => {
    const tasks = [makeNoAddressTask('n1', 'Звонок клиенту')];
    renderTaskList(tasks);
    expect(screen.getByText(/Без адреса \(1\)/i)).toBeInTheDocument();
  });

  it('отображает обе группы одновременно', () => {
    const tasks = [
      makeAddressTask('a1', 'Встреча'),
      makeNoAddressTask('n1', 'Звонок'),
    ];
    renderTaskList(tasks);
    expect(screen.getByText(/С адресом \(1\)/i)).toBeInTheDocument();
    expect(screen.getByText(/Без адреса \(1\)/i)).toBeInTheDocument();
    expect(screen.getByText('Встреча')).toBeInTheDocument();
    expect(screen.getByText('Звонок')).toBeInTheDocument();
  });

  it('не показывает заголовок «Без адреса», если все задачи имеют адрес', () => {
    const tasks = [makeAddressTask('a1', 'Задача А'), makeAddressTask('a2', 'Задача Б')];
    renderTaskList(tasks);
    expect(screen.queryByText(/Без адреса/i)).not.toBeInTheDocument();
  });

  it('не показывает заголовок «С адресом», если ни одна задача не имеет адреса', () => {
    const tasks = [makeNoAddressTask('n1', 'Звонок'), makeNoAddressTask('n2', 'Email')];
    renderTaskList(tasks);
    expect(screen.queryByText(/С адресом/i)).not.toBeInTheDocument();
  });

  it('показывает количество задач в каждой группе', () => {
    const tasks = [
      makeAddressTask('a1', 'A1'),
      makeAddressTask('a2', 'A2'),
      makeNoAddressTask('n1', 'N1'),
    ];
    renderTaskList(tasks);
    expect(screen.getByText(/С адресом \(2\)/i)).toBeInTheDocument();
    expect(screen.getByText(/Без адреса \(1\)/i)).toBeInTheDocument();
  });

  it('задачи без адреса отображают метку «Без адреса» вместо адреса', () => {
    const tasks = [makeNoAddressTask('n1', 'Звонок клиенту')];
    renderTaskList(tasks);
    expect(screen.getByText('Без адреса')).toBeInTheDocument();
  });

  it('итоговый счётчик в заголовке отражает все задачи', () => {
    const tasks = [makeAddressTask('a1', 'A'), makeNoAddressTask('n1', 'N')];
    renderTaskList(tasks);
    expect(screen.getByText(/Список задач \(2\)/i)).toBeInTheDocument();
  });
});

describe('isTaskWindowConflict — индивидуальная проверка', () => {
  it('возвращает false если нет временного окна', () => {
    expect(isTaskWindowConflict(makeTask({ id: '1', title: 't' }))).toBe(false);
  });

  it('возвращает false если только начало окна', () => {
    expect(isTaskWindowConflict(makeTask({
      id: '1', title: 't',
      windowStartDate: '2025-01-01', windowStartTime: '09:00',
    }))).toBe(false);
  });

  it('возвращает false если только конец окна', () => {
    expect(isTaskWindowConflict(makeTask({
      id: '1', title: 't',
      windowEndDate: '2025-01-01', windowEndTime: '18:00',
    }))).toBe(false);
  });

  it('возвращает false если задача укладывается в окно', () => {
    expect(isTaskWindowConflict(makeTask({
      id: '1', title: 't', duration: 60,
      windowStartDate: '2025-01-01', windowStartTime: '09:00',
      windowEndDate: '2025-01-01', windowEndTime: '11:00',
    }))).toBe(false);
  });

  it('возвращает true если длительность задачи больше окна', () => {
    expect(isTaskWindowConflict(makeTask({
      id: '1', title: 't', duration: 120,
      windowStartDate: '2025-01-01', windowStartTime: '09:00',
      windowEndDate: '2025-01-01', windowEndTime: '10:00',
    }))).toBe(true);
  });

  it('возвращает true если конец окна раньше начала', () => {
    expect(isTaskWindowConflict(makeTask({
      id: '1', title: 't', duration: 10,
      windowStartDate: '2025-01-02', windowStartTime: '09:00',
      windowEndDate: '2025-01-01', windowEndTime: '09:00',
    }))).toBe(true);
  });

  it('возвращает true если окно равно нулю (начало == конец)', () => {
    expect(isTaskWindowConflict(makeTask({
      id: '1', title: 't', duration: 10,
      windowStartDate: '2025-01-01', windowStartTime: '09:00',
      windowEndDate: '2025-01-01', windowEndTime: '09:00',
    }))).toBe(true);
  });

  it('возвращает false для мгновенной задачи если окно > 0', () => {
    expect(isTaskWindowConflict(makeTask({
      id: '1', title: 't', duration: undefined,
      windowStartDate: '2025-01-01', windowStartTime: '09:00',
      windowEndDate: '2025-01-01', windowEndTime: '10:00',
    }))).toBe(false);
  });

  it('возвращает false если длительность точно равна окну', () => {
    expect(isTaskWindowConflict(makeTask({
      id: '1', title: 't', duration: 60,
      windowStartDate: '2025-01-01', windowStartTime: '09:00',
      windowEndDate: '2025-01-01', windowEndTime: '10:00',
    }))).toBe(false);
  });
});

describe('getWindowConflictIds — межзадачные конфликты', () => {
  it('нет конфликтов у единственной задачи с валидным окном', () => {
    const tasks = [makeTask({
      id: '1', title: 't', duration: 30,
      windowStartDate: '2025-01-01', windowStartTime: '09:00',
      windowEndDate: '2025-01-01', windowEndTime: '10:00',
    })];
    expect(getWindowConflictIds(tasks).size).toBe(0);
  });

  it('конфликт у двух задач в одном окне (суммарная длительность > окно)', () => {
    const tasks = [
      makeTask({ id: '1', title: 'A', duration: 30,
        windowStartDate: '2025-01-01', windowStartTime: '12:00',
        windowEndDate: '2025-01-01', windowEndTime: '12:30' }),
      makeTask({ id: '2', title: 'B', duration: 30,
        windowStartDate: '2025-01-01', windowStartTime: '12:00',
        windowEndDate: '2025-01-01', windowEndTime: '12:30' }),
    ];
    const ids = getWindowConflictIds(tasks);
    expect(ids.has('1')).toBe(true);
    expect(ids.has('2')).toBe(true);
  });

  it('задача без пересечения окон не конфликтует', () => {
    const tasks = [
      makeTask({ id: '1', title: 'A', duration: 30,
        windowStartDate: '2025-01-01', windowStartTime: '14:00',
        windowEndDate: '2025-01-01', windowEndTime: '14:30' }),
      makeTask({ id: '2', title: 'B', duration: 30,
        windowStartDate: '2025-01-01', windowStartTime: '12:00',
        windowEndDate: '2025-01-01', windowEndTime: '12:30' }),
      makeTask({ id: '3', title: 'C', duration: 30,
        windowStartDate: '2025-01-01', windowStartTime: '12:00',
        windowEndDate: '2025-01-01', windowEndTime: '12:30' }),
    ];
    const ids = getWindowConflictIds(tasks);
    expect(ids.has('1')).toBe(false);
    expect(ids.has('2')).toBe(true);
    expect(ids.has('3')).toBe(true);
  });

  it('задачи без временных окон не конфликтуют', () => {
    const tasks = [
      makeTask({ id: '1', title: 'A', duration: 30 }),
      makeTask({ id: '2', title: 'B', duration: 30 }),
    ];
    expect(getWindowConflictIds(tasks).size).toBe(0);
  });

  it('индивидуальный конфликт (длительность > окно) тоже попадает', () => {
    const tasks = [makeTask({
      id: '1', title: 't', duration: 120,
      windowStartDate: '2025-01-01', windowStartTime: '09:00',
      windowEndDate: '2025-01-01', windowEndTime: '10:00',
    })];
    expect(getWindowConflictIds(tasks).has('1')).toBe(true);
  });

  it('три задачи в одном окне — все конфликтуют', () => {
    const tasks = [
      makeTask({ id: '1', title: 'A', duration: 30,
        windowStartDate: '2025-01-01', windowStartTime: '12:00',
        windowEndDate: '2025-01-01', windowEndTime: '12:30' }),
      makeTask({ id: '2', title: 'B', duration: 30,
        windowStartDate: '2025-01-01', windowStartTime: '12:00',
        windowEndDate: '2025-01-01', windowEndTime: '12:30' }),
      makeTask({ id: '3', title: 'C', duration: 30,
        windowStartDate: '2025-01-01', windowStartTime: '12:00',
        windowEndDate: '2025-01-01', windowEndTime: '12:30' }),
    ];
    const ids = getWindowConflictIds(tasks);
    expect(ids.size).toBe(3);
  });

  it('две задачи с достаточным окном — нет конфликта', () => {
    const tasks = [
      makeTask({ id: '1', title: 'A', duration: 30,
        windowStartDate: '2025-01-01', windowStartTime: '09:00',
        windowEndDate: '2025-01-01', windowEndTime: '11:00' }),
      makeTask({ id: '2', title: 'B', duration: 30,
        windowStartDate: '2025-01-01', windowStartTime: '09:00',
        windowEndDate: '2025-01-01', windowEndTime: '11:00' }),
    ];
    expect(getWindowConflictIds(tasks).size).toBe(0);
  });
});

describe('TaskList — конфликт временного окна', () => {
  it('отображает иконку ошибки для задачи с индивидуальным конфликтом', () => {
    const tasks = [makeTask({
      id: 'c1', title: 'Конфликтная задача',
      address: 'ул. Тестовая', latitude: 55.75, longitude: 37.61,
      duration: 120,
      windowStartDate: '2025-01-01', windowStartTime: '09:00',
      windowEndDate: '2025-01-01', windowEndTime: '10:00',
    })];
    renderTaskList(tasks);
    const container = screen.getByText('Конфликтная задача').closest('[class*="card"]')!;
    expect(container.querySelector('.lucide-triangle-alert')).toBeInTheDocument();
  });

  it('красная обводка карточки при межзадачном конфликте', () => {
    const tasks = [
      makeTask({ id: 'c1', title: 'Задача 1',
        address: 'ул. А', latitude: 55.75, longitude: 37.61, duration: 30,
        windowStartDate: '2025-01-01', windowStartTime: '12:00',
        windowEndDate: '2025-01-01', windowEndTime: '12:30' }),
      makeTask({ id: 'c2', title: 'Задача 2',
        address: 'ул. Б', latitude: 55.76, longitude: 37.62, duration: 30,
        windowStartDate: '2025-01-01', windowStartTime: '12:00',
        windowEndDate: '2025-01-01', windowEndTime: '12:30' }),
    ];
    renderTaskList(tasks);
    const card1 = screen.getByText('Задача 1').closest('[class*="card"]')!;
    const card2 = screen.getByText('Задача 2').closest('[class*="card"]')!;
    expect(card1.className).toContain('border-red-400');
    expect(card2.className).toContain('border-red-400');
  });

  it('не отображает иконку ошибки для задачи без конфликта', () => {
    const tasks = [makeTask({
      id: 'ok1', title: 'Нормальная задача',
      address: 'ул. Тестовая', latitude: 55.75, longitude: 37.61,
      duration: 30,
      windowStartDate: '2025-01-01', windowStartTime: '09:00',
      windowEndDate: '2025-01-01', windowEndTime: '12:00',
    })];
    renderTaskList(tasks);
    const container = screen.getByText('Нормальная задача').closest('[class*="card"]')!;
    expect(container.querySelector('.lucide-triangle-alert')).not.toBeInTheDocument();
  });

  it('задача с непересекающимся окном не помечается при конфликте других', () => {
    const tasks = [
      makeTask({ id: '1', title: 'Свободная',
        address: 'ул. А', latitude: 55.75, longitude: 37.61, duration: 30,
        windowStartDate: '2025-01-01', windowStartTime: '14:00',
        windowEndDate: '2025-01-01', windowEndTime: '14:30' }),
      makeTask({ id: '2', title: 'Конфликт А',
        address: 'ул. Б', latitude: 55.76, longitude: 37.62, duration: 30,
        windowStartDate: '2025-01-01', windowStartTime: '12:00',
        windowEndDate: '2025-01-01', windowEndTime: '12:30' }),
      makeTask({ id: '3', title: 'Конфликт Б',
        address: 'ул. В', latitude: 55.77, longitude: 37.63, duration: 30,
        windowStartDate: '2025-01-01', windowStartTime: '12:00',
        windowEndDate: '2025-01-01', windowEndTime: '12:30' }),
    ];
    renderTaskList(tasks);
    const free = screen.getByText('Свободная').closest('[class*="card"]')!;
    const confA = screen.getByText('Конфликт А').closest('[class*="card"]')!;
    const confB = screen.getByText('Конфликт Б').closest('[class*="card"]')!;
    expect(free.className).not.toContain('border-red-400');
    expect(confA.className).toContain('border-red-400');
    expect(confB.className).toContain('border-red-400');
  });
});
