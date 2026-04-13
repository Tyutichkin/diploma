import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
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
  timeWindowStart: '09:00',
  timeWindowEnd: '18:00',
};

const taskWithStartOnly: Task = {
  id: 'task-2',
  title: 'Задача только с началом',
  address: 'ул. Арбат, 1',
  latitude: 55.75,
  longitude: 37.61,
  duration: 30,
  timeWindowStart: '10:00',
};

beforeEach(() => {
  vi.clearAllMocks();
});

describe('TaskForm — индивидуальные кнопки сброса временного окна', () => {
  it('кнопки очистки не отображаются, если поля пусты', () => {
    render(<TaskForm {...defaultProps} />);
    expect(screen.queryByRole('button', { name: /очистить начало окна/i })).toBeNull();
    expect(screen.queryByRole('button', { name: /очистить конец окна/i })).toBeNull();
  });

  it('кнопка очистки начала окна появляется при вводе значения', async () => {
    render(<TaskForm {...defaultProps} />);
    const startInput = screen.getByLabelText('Начало окна');
    await userEvent.type(startInput, '09:00');
    expect(screen.getByRole('button', { name: /очистить начало окна/i })).toBeTruthy();
    expect(screen.queryByRole('button', { name: /очистить конец окна/i })).toBeNull();
  });

  it('кнопка очистки конца окна появляется при вводе значения', async () => {
    render(<TaskForm {...defaultProps} />);
    const endInput = screen.getByLabelText('Конец окна');
    await userEvent.type(endInput, '18:00');
    expect(screen.queryByRole('button', { name: /очистить начало окна/i })).toBeNull();
    expect(screen.getByRole('button', { name: /очистить конец окна/i })).toBeTruthy();
  });

  it('обе кнопки очистки отображаются при редактировании задачи с полным окном', () => {
    render(<TaskForm {...defaultProps} task={taskWithWindow} />);
    expect(screen.getByRole('button', { name: /очистить начало окна/i })).toBeTruthy();
    expect(screen.getByRole('button', { name: /очистить конец окна/i })).toBeTruthy();
  });

  it('только кнопка начала окна отображается при задаче с одним полем', () => {
    render(<TaskForm {...defaultProps} task={taskWithStartOnly} />);
    expect(screen.getByRole('button', { name: /очистить начало окна/i })).toBeTruthy();
    expect(screen.queryByRole('button', { name: /очистить конец окна/i })).toBeNull();
  });

  it('клик по очистке начала очищает только начало окна', () => {
    render(<TaskForm {...defaultProps} task={taskWithWindow} />);

    const startInput = screen.getByLabelText('Начало окна') as HTMLInputElement;
    const endInput = screen.getByLabelText('Конец окна') as HTMLInputElement;

    fireEvent.click(screen.getByRole('button', { name: /очистить начало окна/i }));

    expect(startInput.value).toBe('');
    expect(endInput.value).toBe('18:00');
  });

  it('клик по очистке конца очищает только конец окна', () => {
    render(<TaskForm {...defaultProps} task={taskWithWindow} />);

    const startInput = screen.getByLabelText('Начало окна') as HTMLInputElement;
    const endInput = screen.getByLabelText('Конец окна') as HTMLInputElement;

    fireEvent.click(screen.getByRole('button', { name: /очистить конец окна/i }));

    expect(startInput.value).toBe('09:00');
    expect(endInput.value).toBe('');
  });

  it('после очистки начала кнопка начала исчезает, кнопка конца остаётся', () => {
    render(<TaskForm {...defaultProps} task={taskWithWindow} />);

    fireEvent.click(screen.getByRole('button', { name: /очистить начало окна/i }));

    expect(screen.queryByRole('button', { name: /очистить начало окна/i })).toBeNull();
    expect(screen.getByRole('button', { name: /очистить конец окна/i })).toBeTruthy();
  });

  it('после очистки обоих полей обе кнопки исчезают', () => {
    render(<TaskForm {...defaultProps} task={taskWithWindow} />);

    fireEvent.click(screen.getByRole('button', { name: /очистить начало окна/i }));
    fireEvent.click(screen.getByRole('button', { name: /очистить конец окна/i }));

    expect(screen.queryByRole('button', { name: /очистить начало окна/i })).toBeNull();
    expect(screen.queryByRole('button', { name: /очистить конец окна/i })).toBeNull();
  });

  it('после очистки начала и сохранения timeWindowStart передаётся как пустая строка', async () => {
    render(<TaskForm {...defaultProps} task={taskWithWindow} />);

    fireEvent.click(screen.getByRole('button', { name: /очистить начало окна/i }));

    fireEvent.submit(screen.getByRole('button', { name: /сохранить/i }).closest('form')!);

    await waitFor(() => {
      expect(mockOnSave).toHaveBeenCalledOnce();
      const savedTask = mockOnSave.mock.calls[0][0] as Task;
      expect(savedTask.timeWindowStart).toBe('');
      expect(savedTask.timeWindowEnd).toBe('18:00');
    });
  });

  it('после очистки обоих полей и сохранения оба поля передаются как пустые строки', async () => {
    render(<TaskForm {...defaultProps} task={taskWithWindow} />);

    fireEvent.click(screen.getByRole('button', { name: /очистить начало окна/i }));
    fireEvent.click(screen.getByRole('button', { name: /очистить конец окна/i }));

    fireEvent.submit(screen.getByRole('button', { name: /сохранить/i }).closest('form')!);

    await waitFor(() => {
      expect(mockOnSave).toHaveBeenCalledOnce();
      const savedTask = mockOnSave.mock.calls[0][0] as Task;
      expect(savedTask.timeWindowStart).toBe('');
      expect(savedTask.timeWindowEnd).toBe('');
    });
  });
});
