import { describe, it, expect } from 'vitest';
import { normalizeTime, serializeTime, getEmailFromToken, mapTaskFromApi, ApiTask } from './clientUtils';

// ── 7.6 Утилиты маппинга ──────────────────────────────────────────────────────

describe('normalizeTime', () => {
  // 7.6.1 Корректное время
  it('slices HH:MM:SS to HH:MM', () => {
    expect(normalizeTime('09:30:00')).toBe('09:30');
  });

  // 7.6.2 undefined → undefined
  it('returns undefined for undefined', () => {
    expect(normalizeTime(undefined)).toBeUndefined();
  });

  it('returns undefined for null', () => {
    expect(normalizeTime(null)).toBeUndefined();
  });

  it('returns undefined for empty string', () => {
    expect(normalizeTime('')).toBeUndefined();
  });

  it('normalizes midnight correctly', () => {
    expect(normalizeTime('00:00:00')).toBe('00:00');
  });

  it('normalizes end of day correctly', () => {
    expect(normalizeTime('23:59:59')).toBe('23:59');
  });
});

describe('serializeTime', () => {
  // 7.6.3 Добавляет :00 к HH:MM
  it('appends :00 to HH:MM format', () => {
    expect(serializeTime('09:30')).toBe('09:30:00');
  });

  it('returns empty string for undefined', () => {
    expect(serializeTime(undefined)).toBe('');
  });

  it('returns empty string for empty string', () => {
    expect(serializeTime('')).toBe('');
  });

  it('does not modify already-full HH:MM:SS format', () => {
    expect(serializeTime('09:30:00')).toBe('09:30:00');
  });

  it('serializes midnight', () => {
    expect(serializeTime('00:00')).toBe('00:00:00');
  });
});

describe('getEmailFromToken', () => {
  // Helper: build a fake JWT with email in payload
  function makeJwtWithEmail(email: string): string {
    const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' })).replace(/=/g, '');
    const payload = btoa(JSON.stringify({ email, uid: 'user-1', exp: 9999999999 })).replace(/=/g, '');
    return `${header}.${payload}.fake-signature`;
  }

  // 7.6.4 Валидный JWT → возвращает email
  it('extracts email from valid JWT payload', () => {
    const token = makeJwtWithEmail('user@example.com');
    expect(getEmailFromToken(token)).toBe('user@example.com');
  });

  // 7.6.5 Случайная строка → null
  it('returns null for random string', () => {
    expect(getEmailFromToken('notajwt')).toBeNull();
  });

  it('returns null for empty string', () => {
    expect(getEmailFromToken('')).toBeNull();
  });

  it('returns null for token without email claim', () => {
    const header = btoa(JSON.stringify({ alg: 'HS256' })).replace(/=/g, '');
    const payload = btoa(JSON.stringify({ uid: 'user-1' })).replace(/=/g, '');
    const token = `${header}.${payload}.sig`;
    expect(getEmailFromToken(token)).toBeNull();
  });

  it('returns null for malformed base64 payload', () => {
    expect(getEmailFromToken('header.!!!.sig')).toBeNull();
  });
});

describe('mapTaskFromApi', () => {
  const baseApiTask: ApiTask = {
    ID: 'task-123',
    Title: 'Buy groceries',
    AddressText: '123 Main Street',
    Latitude: 55.75,
    Longitude: 37.61,
    DurationMin: 30,
    SortIndex: 2,
    IsCompleted: false,
  };

  // 7.6.6 Маппинг PascalCase → camelCase
  it('maps API task PascalCase fields to camelCase', () => {
    const result = mapTaskFromApi(baseApiTask);
    expect(result.id).toBe('task-123');
    expect(result.title).toBe('Buy groceries');
    expect(result.address).toBe('123 Main Street');
    expect(result.latitude).toBe(55.75);
    expect(result.longitude).toBe(37.61);
    expect(result.duration).toBe(30);
    expect(result.order).toBe(2);
    expect(result.completed).toBe(false);
  });

  it('normalizes WindowStart from HH:MM:SS to HH:MM', () => {
    const task = { ...baseApiTask, WindowStart: '09:30:00', WindowEnd: '18:00:00' };
    const result = mapTaskFromApi(task);
    expect(result.timeWindowStart).toBe('09:30');
    expect(result.timeWindowEnd).toBe('18:00');
  });

  // 7.6.7 Задача без time window
  it('returns undefined for timeWindowStart when no time window', () => {
    const result = mapTaskFromApi(baseApiTask);
    expect(result.timeWindowStart).toBeUndefined();
    expect(result.timeWindowEnd).toBeUndefined();
  });

  it('handles null WindowStart', () => {
    const task = { ...baseApiTask, WindowStart: null };
    const result = mapTaskFromApi(task);
    expect(result.timeWindowStart).toBeUndefined();
  });
});
