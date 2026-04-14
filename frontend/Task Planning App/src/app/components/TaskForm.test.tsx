import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { TaskForm } from './TaskForm';
import { Task } from '../types/task';

const mockOnClose = vi.fn();
const mockOnSave = vi.fn().mockResolvedValue(undefined);
const mockOnGeocodeMultiple = vi.fn().mockResolvedValue([
  { lat: 55.75, lng: 37.61, displayName: 'ул. Тверская, 1, Москва' },
]);

const defaultProps = {
  task: null,
  isOpen: true,
  isSaving: false,
  onClose: mockOnClose,
  onSave: mockOnSave,
  onGeocodeMultiple: mockOnGeocodeMultiple,
};

const taskWithWindow: Task = {
  id: 'task-1',
  title: 'Тестовая задача',
  address: 'ул. Тверская, 1',
  latitude: 55.75,
  longitude: 37.61,
  duration: 30,
  windowStartDate: '2024-01-15',
  windowStartTime: '09:00',
  windowEndDate: '2024-01-15',
  windowEndTime: '18:00',
};

const taskWithStartDateOnly: Task = {
  id: 'task-2',
  title: 'Задача только с датой начала',
  address: 'ул. Арбат, 1',
  latitude: 55.75,
  longitude: 37.61,
  duration: 30,
  windowStartDate: '2024-01-15',
};

beforeEach(() => {
  vi.clearAllMocks();
});

describe('TaskForm — раздельные поля дата/время для временного окна', () => {
  it('кнопки очистки не отображаются, если поля пусты', () => {
    render(<TaskForm {...defaultProps} />);
    expect(screen.queryByRole('button', { name: /очистить дату начала окна/i })).toBeNull();
    expect(screen.queryByRole('button', { name: /очистить время начала окна/i })).toBeNull();
    expect(screen.queryByRole('button', { name: /очистить дату конца окна/i })).toBeNull();
    expect(screen.queryByRole('button', { name: /очистить время конца окна/i })).toBeNull();
  });

  it('кнопки очистки появляются при вводе даты начала', () => {
    render(<TaskForm {...defaultProps} />);
    const startDateInput = screen.getByLabelText('Дата начала окна');
    fireEvent.change(startDateInput, { target: { value: '2024-01-15' } });
    expect(screen.getByRole('button', { name: /очистить дату начала окна/i })).toBeTruthy();
  });

  it('все кнопки очистки отображаются при редактировании задачи с полным окном', () => {
    render(<TaskForm {...defaultProps} task={taskWithWindow} />);
    expect(screen.getByRole('button', { name: /очистить дату начала окна/i })).toBeTruthy();
    expect(screen.getByRole('button', { name: /очистить время начала окна/i })).toBeTruthy();
    expect(screen.getByRole('button', { name: /очистить дату конца окна/i })).toBeTruthy();
    expect(screen.getByRole('button', { name: /очистить время конца окна/i })).toBeTruthy();
  });

  it('только кнопка даты начала окна отображается при задаче только с датой начала', () => {
    render(<TaskForm {...defaultProps} task={taskWithStartDateOnly} />);
    expect(screen.getByRole('button', { name: /очистить дату начала окна/i })).toBeTruthy();
    expect(screen.queryByRole('button', { name: /очистить время начала окна/i })).toBeNull();
    expect(screen.queryByRole('button', { name: /очистить дату конца окна/i })).toBeNull();
    expect(screen.queryByRole('button', { name: /очистить время конца окна/i })).toBeNull();
  });

  it('клик по очистке даты начала очищает только дату начала', () => {
    render(<TaskForm {...defaultProps} task={taskWithWindow} />);

    const startDateInput = screen.getByLabelText('Дата начала окна') as HTMLInputElement;
    const startTimeInput = screen.getByLabelText('Время начала окна') as HTMLInputElement;

    fireEvent.click(screen.getByRole('button', { name: /очистить дату начала окна/i }));

    expect(startDateInput.value).toBe('');
    expect(startTimeInput.value).toBe('09:00');
  });

  it('клик по очистке времени конца очищает только время конца', () => {
    render(<TaskForm {...defaultProps} task={taskWithWindow} />);

    const endDateInput = screen.getByLabelText('Дата конца окна') as HTMLInputElement;
    const endTimeInput = screen.getByLabelText('Время конца окна') as HTMLInputElement;

    fireEvent.click(screen.getByRole('button', { name: /очистить время конца окна/i }));

    expect(endDateInput.value).toBe('2024-01-15');
    expect(endTimeInput.value).toBe('');
  });

  it('после очистки всех полей кнопки исчезают', () => {
    render(<TaskForm {...defaultProps} task={taskWithWindow} />);

    fireEvent.click(screen.getByRole('button', { name: /очистить дату начала окна/i }));
    fireEvent.click(screen.getByRole('button', { name: /очистить время начала окна/i }));
    fireEvent.click(screen.getByRole('button', { name: /очистить дату конца окна/i }));
    fireEvent.click(screen.getByRole('button', { name: /очистить время конца окна/i }));

    expect(screen.queryByRole('button', { name: /очистить дату начала окна/i })).toBeNull();
    expect(screen.queryByRole('button', { name: /очистить время начала окна/i })).toBeNull();
    expect(screen.queryByRole('button', { name: /очистить дату конца окна/i })).toBeNull();
    expect(screen.queryByRole('button', { name: /очистить время конца окна/i })).toBeNull();
  });

  it('при сохранении задачи передаются раздельные поля окна', async () => {
    render(<TaskForm {...defaultProps} task={taskWithWindow} />);

    fireEvent.submit(screen.getByRole('button', { name: /сохранить/i }).closest('form')!);

    await waitFor(() => {
      expect(mockOnSave).toHaveBeenCalledOnce();
      const savedTask = mockOnSave.mock.calls[0][0] as Task;
      expect(savedTask.windowStartDate).toBe('2024-01-15');
      expect(savedTask.windowStartTime).toBe('09:00');
      expect(savedTask.windowEndDate).toBe('2024-01-15');
      expect(savedTask.windowEndTime).toBe('18:00');
    });
  });

  it('после очистки даты начала и сохранения windowStartDate пустая', async () => {
    render(<TaskForm {...defaultProps} task={taskWithWindow} />);

    fireEvent.click(screen.getByRole('button', { name: /очистить дату начала окна/i }));
    fireEvent.submit(screen.getByRole('button', { name: /сохранить/i }).closest('form')!);

    await waitFor(() => {
      expect(mockOnSave).toHaveBeenCalledOnce();
      const savedTask = mockOnSave.mock.calls[0][0] as Task;
      expect(savedTask.windowStartDate).toBeUndefined();
      expect(savedTask.windowStartTime).toBe('09:00');
      expect(savedTask.windowEndDate).toBe('2024-01-15');
      expect(savedTask.windowEndTime).toBe('18:00');
    });
  });
});
