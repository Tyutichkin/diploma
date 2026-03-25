import { describe, it, expect, beforeEach, afterEach, afterAll, beforeAll, vi } from 'vitest';
import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';
import {
  readStoredSession,
  writeStoredSession,
  login,
  register,
  logoutRequest,
  listTasks,
  createTask,
  updateTask,
  deleteTask,
  reorderTasks,
  saveRoute,
  listRoutes,
  getRoute,
  deleteRoute,
  deleteAllRoutes,
  renameRoute,
  optimizeRoute,
  ApiError,
} from './client';
import type { AuthSession } from '../types/auth';

// ── MSW server ────────────────────────────────────────────────────────────────

const BASE = 'http://localhost:8080/api';

const makeTokenPair = () => ({
  accessToken: makeJwt('user@example.com'),
  refreshToken: 'refresh-token-abc',
});

function makeJwt(email: string): string {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' })).replace(/[=]/g, '');
  const payload = btoa(JSON.stringify({ email, uid: 'uid-1', exp: 9999999999 })).replace(/[=]/g, '');
  return `${header}.${payload}.signature`;
}

const apiTaskResponse = {
  ID: 'task-1',
  Title: 'Buy groceries',
  AddressText: '123 Main St',
  Latitude: 55.75,
  Longitude: 37.61,
  DurationMin: 30,
  SortIndex: 0,
  WindowStart: null,
  WindowEnd: null,
};

const apiRouteResponse = {
  ID: 'route-1',
  Status: 'optimized',
  Source: 'optimized',
  CreatedAt: '2024-01-01T00:00:00Z',
  Name: 'My Route',
};

const apiRouteFullResponse = {
  Route: apiRouteResponse,
  Stops: [
    { TaskID: 'task-1', Position: 0 },
    { TaskID: 'task-2', Position: 1 },
  ],
  Stats: {
    TotalDistanceM: 5000,
    TotalTravelSec: 1200,
    TotalServiceSec: 3600,
  },
};

const server = setupServer(
  http.post(`${BASE}/auth/login`, () => HttpResponse.json(makeTokenPair())),
  http.post(`${BASE}/auth/register`, () => HttpResponse.json(makeTokenPair())),
  http.post(`${BASE}/auth/logout`, () => HttpResponse.json({ ok: true })),
  http.get(`${BASE}/tasks`, () => HttpResponse.json([apiTaskResponse])),
  http.post(`${BASE}/tasks`, () => HttpResponse.json(apiTaskResponse)),
  http.patch(`${BASE}/tasks/task-1`, () => HttpResponse.json(apiTaskResponse)),
  http.delete(`${BASE}/tasks/task-1`, () => HttpResponse.json({ ok: true })),
  http.patch(`${BASE}/tasks/order`, () => HttpResponse.json({ ok: true })),
  http.post(`${BASE}/routes`, () => HttpResponse.json(apiRouteResponse)),
  http.post(`${BASE}/routes/optimize`, () => HttpResponse.json(apiRouteFullResponse)),
  http.get(`${BASE}/routes`, () => HttpResponse.json([apiRouteResponse])),
  http.get(`${BASE}/routes/route-1`, () => HttpResponse.json(apiRouteFullResponse)),
  http.delete(`${BASE}/routes/route-1`, () => HttpResponse.json({ ok: true })),
  http.delete(`${BASE}/routes`, () => HttpResponse.json({ ok: true })),
  http.patch(`${BASE}/routes/route-1/name`, () => HttpResponse.json({ ok: true })),
  http.post(`${BASE}/auth/refresh`, () => HttpResponse.json(makeTokenPair())),
);

beforeAll(() => server.listen());
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

// ── Test helpers ──────────────────────────────────────────────────────────────

function makeSession(overrides?: Partial<AuthSession>): AuthSession {
  return {
    accessToken: makeJwt('user@example.com'),
    refreshToken: 'refresh-token-abc',
    email: 'user@example.com',
    ...overrides,
  };
}

const authOptions = (session?: AuthSession) => ({
  session: session ?? makeSession(),
  onSessionChange: vi.fn(),
  onUnauthorized: vi.fn(),
});

// ── 7.5 Хранилище сессии ──────────────────────────────────────────────────────

describe('Session storage', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  // 7.5.1 writeStoredSession → записывает в localStorage
  it('7.5.1: writes session to localStorage', () => {
    const session: AuthSession = {
      accessToken: 'access',
      refreshToken: 'refresh',
      email: 'user@test.com',
    };
    writeStoredSession(session);
    const raw = localStorage.getItem('planner.auth.session');
    expect(raw).not.toBeNull();
    const parsed = JSON.parse(raw!);
    expect(parsed.accessToken).toBe('access');
    expect(parsed.refreshToken).toBe('refresh');
  });

  // 7.5.2 readStoredSession → читает существующую сессию
  it('7.5.2: reads existing session', () => {
    const session: AuthSession = {
      accessToken: makeJwt('user@test.com'),
      refreshToken: 'refresh',
      email: 'user@test.com',
    };
    writeStoredSession(session);
    const read = readStoredSession();
    expect(read).not.toBeNull();
    expect(read!.accessToken).toBe(session.accessToken);
    expect(read!.refreshToken).toBe(session.refreshToken);
  });

  // 7.5.3 readStoredSession — пусто → null
  it('7.5.3: returns null when localStorage is empty', () => {
    expect(readStoredSession()).toBeNull();
  });

  // 7.5.4 readStoredSession — битый JSON → null, не падает
  it('7.5.4: returns null for corrupted JSON', () => {
    localStorage.setItem('planner.auth.session', '{{broken json}}');
    expect(readStoredSession()).toBeNull();
  });

  // 7.5.5 writeStoredSession(null) → удаляет ключ
  it('7.5.5: removes key when session is null', () => {
    writeStoredSession(makeSession());
    writeStoredSession(null);
    expect(localStorage.getItem('planner.auth.session')).toBeNull();
  });
});

// ── 7.1 Авторизация ───────────────────────────────────────────────────────────

describe('login', () => {
  // 7.1.1 login() — успех
  it('7.1.1: returns AuthSession on success', async () => {
    const session = await login('user@example.com', 'password');
    expect(session.accessToken).toBeTruthy();
    expect(session.refreshToken).toBeTruthy();
    expect(session.email).toBeTruthy();
  });

  // 7.1.2 login() — 401 → ApiError
  it('7.1.2: throws ApiError with status 401 on 401 response', async () => {
    server.use(http.post(`${BASE}/auth/login`, () => HttpResponse.json({ error: 'invalid' }, { status: 401 })));
    await expect(login('user@example.com', 'wrong')).rejects.toMatchObject({
      status: 401,
    });
  });
});

describe('register', () => {
  // 7.1.3 register() — успех
  it('7.1.3: returns AuthSession on success', async () => {
    const session = await register('new@example.com', 'password');
    expect(session.accessToken).toBeTruthy();
    expect(session.refreshToken).toBeTruthy();
  });

  // 7.1.4 register() — 409 дубликат email → ApiError
  it('7.1.4: throws ApiError with status 409 on duplicate email', async () => {
    server.use(http.post(`${BASE}/auth/register`, () => HttpResponse.json({ error: 'exists' }, { status: 409 })));
    await expect(register('dup@example.com', 'password')).rejects.toMatchObject({
      status: 409,
    });
  });
});

describe('logoutRequest', () => {
  // 7.1.5 logoutRequest() — успех
  it('7.1.5: completes without error on success', async () => {
    await expect(logoutRequest('refresh-token')).resolves.toBeUndefined();
  });
});

// ── 7.2 Задачи ────────────────────────────────────────────────────────────────

describe('listTasks', () => {
  // 7.2.1 listTasks() → Task[]
  it('7.2.1: maps API tasks to Task array', async () => {
    const tasks = await listTasks(authOptions());
    expect(tasks).toHaveLength(1);
    expect(tasks[0].id).toBe('task-1');
    expect(tasks[0].title).toBe('Buy groceries');
    expect(tasks[0].duration).toBe(30);
  });

  // 7.2.2 listTasks() — пустой список
  it('7.2.2: returns empty array', async () => {
    server.use(http.get(`${BASE}/tasks`, () => HttpResponse.json([])));
    const tasks = await listTasks(authOptions());
    expect(tasks).toHaveLength(0);
  });
});

describe('createTask', () => {
  // 7.2.3 createTask() → Task
  it('7.2.3: returns created Task', async () => {
    const task = await createTask(
      {
        title: 'Buy groceries',
        address: '123 Main St',
        latitude: 55.75,
        longitude: 37.61,
        duration: 30,
        sortIndex: 0,
      },
      authOptions(),
    );
    expect(task.id).toBe('task-1');
    expect(task.title).toBe('Buy groceries');
  });
});

describe('updateTask', () => {
  // 7.2.4 updateTask() → Task
  it('7.2.4: returns updated Task', async () => {
    const task = await updateTask('task-1', { title: 'Updated' }, authOptions());
    expect(task.id).toBe('task-1');
  });
});

describe('deleteTask', () => {
  // 7.2.5 deleteTask() — успех
  it('7.2.5: completes without error on success', async () => {
    await expect(deleteTask('task-1', authOptions())).resolves.toBeUndefined();
  });

  // 7.2.6 deleteTask() — 404 → ApiError
  it('7.2.6: throws ApiError with status 404 on 404 response', async () => {
    server.use(http.delete(`${BASE}/tasks/task-999`, () => HttpResponse.json({ error: 'not found' }, { status: 404 })));
    await expect(deleteTask('task-999', authOptions())).rejects.toMatchObject({ status: 404 });
  });
});

describe('reorderTasks', () => {
  // 7.2.7 reorderTasks() → PATCH /api/tasks/order
  it('7.2.7: sends PATCH to /api/tasks/order', async () => {
    let capturedUrl = '';
    server.use(
      http.patch(`${BASE}/tasks/order`, ({ request }) => {
        capturedUrl = request.url;
        return HttpResponse.json({ ok: true });
      }),
    );
    await reorderTasks([{ id: 'task-1', sortIndex: 0 }], authOptions());
    expect(capturedUrl).toContain('/tasks/order');
  });
});

// ── 7.3 Маршруты ──────────────────────────────────────────────────────────────

describe('saveRoute', () => {
  // 7.3.1 saveRoute() → SavedRouteSummary
  it('7.3.1: returns SavedRouteSummary', async () => {
    const route = await saveRoute(['task-1'], 'manual', authOptions());
    expect(route.id).toBe('route-1');
    expect(route.status).toBe('optimized');
  });
});

describe('optimizeRoute', () => {
  // 7.3.2 optimizeRoute() → SavedRouteDetails
  it('7.3.2: returns SavedRouteDetails with taskIds and stats', async () => {
    const route = await optimizeRoute(['task-1', 'task-2'], authOptions(), 540);
    expect(route.orderedTaskIds).toHaveLength(2);
    expect(route.totalDistanceKm).toBe(5); // 5000m / 1000
    expect(route.totalTravelTimeMin).toBe(20); // 1200s / 60
  });
});

describe('listRoutes', () => {
  // 7.3.3 listRoutes() → SavedRouteSummary[]
  it('7.3.3: returns array of routes', async () => {
    const routes = await listRoutes(authOptions());
    expect(routes).toHaveLength(1);
    expect(routes[0].id).toBe('route-1');
  });
});

describe('getRoute', () => {
  // 7.3.4 getRoute() → SavedRouteDetails
  it('7.3.4: returns SavedRouteDetails', async () => {
    const route = await getRoute('route-1', authOptions());
    expect(route.id).toBe('route-1');
    expect(route.orderedTaskIds).toEqual(['task-1', 'task-2']);
  });
});

describe('deleteRoute', () => {
  // 7.3.5 deleteRoute() → без ошибки
  it('7.3.5: completes without error', async () => {
    await expect(deleteRoute('route-1', authOptions())).resolves.toBeUndefined();
  });
});

describe('deleteAllRoutes', () => {
  // 7.3.6 deleteAllRoutes() → без ошибки
  it('7.3.6: completes without error', async () => {
    await expect(deleteAllRoutes(authOptions())).resolves.toBeUndefined();
  });
});

describe('renameRoute', () => {
  // 7.3.7 renameRoute() → PATCH /api/routes/:id/name
  it('7.3.7: sends PATCH to /api/routes/:id/name', async () => {
    let capturedUrl = '';
    server.use(
      http.patch(`${BASE}/routes/route-1/name`, ({ request }) => {
        capturedUrl = request.url;
        return HttpResponse.json({ ok: true });
      }),
    );
    await renameRoute('route-1', 'New Name', authOptions());
    expect(capturedUrl).toContain('/routes/route-1/name');
  });
});

// ── 7.4 Обновление токена ─────────────────────────────────────────────────────

describe('Token refresh', () => {
  // 7.4.1 401 → refresh → повтор
  it('7.4.1: retries request after successful token refresh', async () => {
    let listCallCount = 0;
    const onSessionChange = vi.fn();

    server.use(
      http.get(`${BASE}/tasks`, () => {
        listCallCount++;
        if (listCallCount === 1) {
          return HttpResponse.json({ error: 'unauthorized' }, { status: 401 });
        }
        return HttpResponse.json([apiTaskResponse]);
      }),
      http.post(`${BASE}/auth/refresh`, () => HttpResponse.json(makeTokenPair())),
    );

    const tasks = await listTasks({
      session: makeSession(),
      onSessionChange,
      onUnauthorized: vi.fn(),
    });

    expect(tasks).toHaveLength(1);
    expect(onSessionChange).toHaveBeenCalledOnce();
  });

  // 7.4.3 refresh тоже возвращает 401 → сессия обнуляется
  it('7.4.3: calls onUnauthorized when refresh fails with 401', async () => {
    const onUnauthorized = vi.fn();

    server.use(
      http.get(`${BASE}/tasks`, () => HttpResponse.json({ error: 'unauthorized' }, { status: 401 })),
      http.post(`${BASE}/auth/refresh`, () => HttpResponse.json({ error: 'expired' }, { status: 401 })),
    );

    await expect(
      listTasks({
        session: makeSession(),
        onSessionChange: vi.fn(),
        onUnauthorized,
      }),
    ).rejects.toBeTruthy();

    expect(onUnauthorized).toHaveBeenCalled();
  });

  // 7.4.4 Сетевая ошибка
  it('7.4.4: throws on network error', async () => {
    server.use(http.get(`${BASE}/tasks`, () => HttpResponse.error()));
    await expect(
      listTasks({
        session: makeSession(),
        onSessionChange: vi.fn(),
        onUnauthorized: vi.fn(),
      }),
    ).rejects.toBeTruthy();
  });
});

// ── ApiError class ────────────────────────────────────────────────────────────

describe('ApiError', () => {
  it('has correct name and status', () => {
    const err = new ApiError('Not found', 404);
    expect(err.name).toBe('ApiError');
    expect(err.status).toBe(404);
    expect(err.message).toBe('Not found');
    expect(err instanceof Error).toBe(true);
  });
});
