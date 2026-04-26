import { AuthSession } from '../types/auth';
import { Task } from '../types/task';
import { SavedRouteDetails, SavedRouteSummary } from '../types/route';
import { DistanceCell } from '../utils/routeOptimizer';

const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL ?? '/api').replace(/\/$/, '');
const SESSION_STORAGE_KEY = 'planner.auth.session';
let refreshSessionPromise: Promise<AuthSession> | null = null;

interface TokenPairResponse {
  accessToken: string;
  refreshToken: string;
}

interface ApiErrorPayload {
  error?: string;
}

interface ApiTask {
  ID: string;
  Title: string;
  AddressText: string;
  Latitude: number | null;
  Longitude: number | null;
  DurationMin: number | null;
  WindowStartDate?: string | null;
  WindowStartTime?: string | null;
  WindowEndDate?: string | null;
  WindowEndTime?: string | null;
  SortIndex: number;
  IsCompleted: boolean;
}

interface ApiRoute {
  ID: string;
  Status: string;
  Source: string;
  CreatedAt: string;
  Name?: string | null;
}

interface ApiRouteStop {
  TaskID: string;
  Position: number;
  TravelFromPrevSec?: number | null;
  ArriveTime?: string | null;
  ServiceStartTime?: string | null;
  ServiceEndTime?: string | null;
  WaitSec?: number | null;
}

interface ApiRouteStats {
  TotalDistanceM?: number | null;
  TotalTravelSec?: number | null;
  TotalServiceSec?: number | null;
}

interface ApiRouteFull {
  Route: ApiRoute;
  Stops: ApiRouteStop[];
  Stats?: ApiRouteStats | null;
}

export class ApiError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
  }
}

interface AuthorizedRequestOptions {
  session: AuthSession;
  onSessionChange: (session: AuthSession) => void;
  onUnauthorized: () => void | Promise<void>;
}

export function readStoredSession(): AuthSession | null {
  if (typeof window === 'undefined') {
    return null;
  }

  const raw = window.localStorage.getItem(SESSION_STORAGE_KEY);
  if (!raw) {
    return null;
  }

  try {
    const parsed = JSON.parse(raw) as Partial<AuthSession>;
    if (!parsed.accessToken || !parsed.refreshToken) {
      return null;
    }

    return {
      accessToken: parsed.accessToken,
      refreshToken: parsed.refreshToken,
      email: parsed.email ?? getEmailFromToken(parsed.accessToken) ?? '',
    };
  } catch {
    return null;
  }
}

export function writeStoredSession(session: AuthSession | null) {
  if (typeof window === 'undefined') {
    return;
  }

  if (!session) {
    window.localStorage.removeItem(SESSION_STORAGE_KEY);
    return;
  }

  window.localStorage.setItem(SESSION_STORAGE_KEY, JSON.stringify(session));
}

export function getCompletedTaskIdsStorageKey(email: string) {
  return `planner.completed.${email}`;
}

export async function login(email: string, password: string) {
  const pair = await request<TokenPairResponse>('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  });

  return buildSession(pair, email);
}

export async function register(email: string, password: string) {
  const pair = await request<TokenPairResponse>('/auth/register', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  });

  return buildSession(pair, email);
}

export async function logoutRequest(refreshToken: string) {
  await request('/auth/logout', {
    method: 'POST',
    body: JSON.stringify({ refreshToken }),
  });
}

export async function listTasks(options: AuthorizedRequestOptions) {
  const tasks = await requestWithAuth<ApiTask[]>('/tasks', { method: 'GET' }, options);
  return tasks.map(mapTaskFromApi);
}

export interface SaveTaskInput {
  title: string;
  address?: string;
  latitude?: number;
  longitude?: number;
  duration?: number;
  windowStartDate?: string; // "YYYY-MM-DD"
  windowStartTime?: string; // "HH:mm"
  windowEndDate?: string;
  windowEndTime?: string;
  sortIndex: number;
  completed?: boolean;
}

// Тело POST /tasks и /tasks/batch: пустые поля → null (бэк ждёт явный сброс).
function taskCreateBody(input: SaveTaskInput) {
  return {
    title: input.title,
    addressText: input.address ?? null,
    latitude: input.latitude ?? null,
    longitude: input.longitude ?? null,
    durationMin: input.duration,
    windowStartDate: input.windowStartDate || null,
    windowStartTime: input.windowStartTime || null,
    windowEndDate: input.windowEndDate || null,
    windowEndTime: input.windowEndTime || null,
    sortIndex: input.sortIndex,
  };
}

// Тело PATCH /tasks/:id: undefined-поля бэк не трогает (partial update).
function taskUpdateBody(input: Partial<SaveTaskInput>) {
  return {
    title: input.title,
    addressText: input.address,
    latitude: input.latitude,
    longitude: input.longitude,
    durationMin: input.duration,
    windowStartDate: input.windowStartDate,
    windowStartTime: input.windowStartTime,
    windowEndDate: input.windowEndDate,
    windowEndTime: input.windowEndTime,
    sortIndex: input.sortIndex,
    isCompleted: input.completed,
  };
}

export async function createTask(input: SaveTaskInput, options: AuthorizedRequestOptions) {
  const task = await requestWithAuth<ApiTask>(
    '/tasks',
    { method: 'POST', body: JSON.stringify(taskCreateBody(input)) },
    options,
  );
  return mapTaskFromApi(task);
}

export async function createTasksBatch(inputs: SaveTaskInput[], options: AuthorizedRequestOptions) {
  const tasks = await requestWithAuth<ApiTask[]>(
    '/tasks/batch',
    { method: 'POST', body: JSON.stringify({ tasks: inputs.map(taskCreateBody) }) },
    options,
  );
  return tasks.map(mapTaskFromApi);
}

export async function updateTask(taskId: string, input: Partial<SaveTaskInput>, options: AuthorizedRequestOptions) {
  const task = await requestWithAuth<ApiTask>(
    `/tasks/${taskId}`,
    { method: 'PATCH', body: JSON.stringify(taskUpdateBody(input)) },
    options,
  );
  return mapTaskFromApi(task);
}

export async function deleteTask(taskId: string, options: AuthorizedRequestOptions) {
  await requestWithAuth(`/tasks/${taskId}`, { method: 'DELETE' }, options);
}

export async function reorderTasks(
  order: { id: string; sortIndex: number }[],
  options: AuthorizedRequestOptions,
) {
  await requestWithAuth('/tasks/order', { method: 'PATCH', body: JSON.stringify({ order }) }, options);
}

export async function saveRoute(taskIds: string[], source: string, options: AuthorizedRequestOptions) {
  const route = await requestWithAuth<ApiRoute>(
    '/routes',
    {
      method: 'POST',
      body: JSON.stringify({
        source,
        orderedTaskIds: taskIds,
      }),
    },
    options,
  );

  return mapRouteSummary(route);
}

export interface PrecedenceConstraint {
  beforeTaskId: string;
  afterTaskId: string;
}

// startTimeUnix: 0 — бэкенд подставит сегодня 09:00 UTC.
// distanceMatrix, посчитанная через buildYandexDistanceMatrix, синхронизирует
// расстояния оптимизатора с теми, что показываются на карте.
export async function optimizeRoute(
  taskIds: string[],
  options: AuthorizedRequestOptions,
  startTimeUnix = 0,
  distanceMatrix?: DistanceCell[][],
  startTaskId?: string,
  endTaskId?: string,
  precedenceConstraints?: PrecedenceConstraint[],
): Promise<SavedRouteDetails> {
  const route = await requestWithAuth<ApiRouteFull>(
    '/routes/optimize',
    {
      method: 'POST',
      body: JSON.stringify({
        taskIds,
        startTimeUnix,
        distanceMatrix,
        startTaskId: startTaskId ?? undefined,
        endTaskId: endTaskId ?? undefined,
        precedenceConstraints: precedenceConstraints?.length ? precedenceConstraints : undefined,
      }),
    },
    options,
  );
  return mapRouteDetails(route);
}

export async function deleteRoute(routeId: string, options: AuthorizedRequestOptions) {
  await requestWithAuth(`/routes/${routeId}`, { method: 'DELETE' }, options);
}

export async function deleteAllRoutes(options: AuthorizedRequestOptions) {
  await requestWithAuth('/routes', { method: 'DELETE' }, options);
}

export async function renameRoute(routeId: string, name: string, options: AuthorizedRequestOptions) {
  await requestWithAuth(`/routes/${routeId}/name`, { method: 'PATCH', body: JSON.stringify({ name }) }, options);
}

export async function listRoutes(options: AuthorizedRequestOptions) {
  const routes = await requestWithAuth<ApiRoute[]>('/routes', { method: 'GET' }, options);
  return routes.map(mapRouteSummary);
}

export async function getRoute(routeId: string, options: AuthorizedRequestOptions) {
  const route = await requestWithAuth<ApiRouteFull>(`/routes/${routeId}`, { method: 'GET' }, options);
  return mapRouteDetails(route);
}

async function requestWithAuth<T>(path: string, init: RequestInit, options: AuthorizedRequestOptions): Promise<T> {
  try {
    return await request<T>(path, init, options.session.accessToken);
  } catch (error) {
    if (!(error instanceof ApiError) || error.status !== 401) {
      throw error;
    }
  }

  try {
    const nextSession = await refreshSession(options);
    return request<T>(path, init, nextSession.accessToken);
  } catch (error) {
    await options.onUnauthorized();
    throw error;
  }
}

async function refreshSession(options: AuthorizedRequestOptions) {
  if (!refreshSessionPromise) {
    refreshSessionPromise = request<TokenPairResponse>('/auth/refresh', {
      method: 'POST',
      body: JSON.stringify({ refreshToken: options.session.refreshToken }),
    })
      .then((pair) => {
        const nextSession = buildSession(pair, getEmailFromToken(pair.accessToken) ?? options.session.email);
        options.onSessionChange(nextSession);
        return nextSession;
      })
      .finally(() => {
        refreshSessionPromise = null;
      });
  }

  return refreshSessionPromise;
}

async function request<T = void>(path: string, init: RequestInit, accessToken?: string): Promise<T> {
  const headers = new Headers(init.headers);

  if (!headers.has('Content-Type') && init.body) {
    headers.set('Content-Type', 'application/json');
  }

  if (!headers.has('Accept')) {
    headers.set('Accept', 'application/json');
  }

  if (accessToken) {
    headers.set('Authorization', `Bearer ${accessToken}`);
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    headers,
  });

  if (!response.ok) {
    const payload = await parseErrorPayload(response);
    throw new ApiError(payload?.error ?? 'Request failed', response.status);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  const text = await response.text();
  if (!text) {
    return undefined as T;
  }

  return JSON.parse(text) as T;
}

async function parseErrorPayload(response: Response) {
  try {
    return (await response.json()) as ApiErrorPayload;
  } catch {
    return null;
  }
}

function buildSession(pair: TokenPairResponse, fallbackEmail: string): AuthSession {
  return {
    accessToken: pair.accessToken,
    refreshToken: pair.refreshToken,
    email: getEmailFromToken(pair.accessToken) ?? fallbackEmail,
  };
}

function mapTaskFromApi(task: ApiTask): Task {
  return {
    id: task.ID,
    title: task.Title,
    address: task.AddressText || undefined,
    latitude: task.Latitude ?? undefined,
    longitude: task.Longitude ?? undefined,
    duration: task.DurationMin ?? undefined,
    windowStartDate: task.WindowStartDate || undefined,
    windowStartTime: task.WindowStartTime || undefined,
    windowEndDate: task.WindowEndDate || undefined,
    windowEndTime: task.WindowEndTime || undefined,
    order: task.SortIndex,
    completed: task.IsCompleted,
  };
}

function mapRouteSummary(route: ApiRoute): SavedRouteSummary {
  return {
    id: route.ID,
    status: route.Status,
    source: route.Source,
    createdAt: route.CreatedAt,
    name: route.Name ?? undefined,
  };
}

function mapRouteDetails(route: ApiRouteFull): SavedRouteDetails {
  const sortedStops = [...(route.Stops ?? [])].sort((a, b) => a.Position - b.Position);

  const orderedTaskIds = sortedStops.map((s) => s.TaskID);

  const stops = sortedStops.map((s) => ({
    taskId: s.TaskID,
    position: s.Position,
    travelFromPrevSec: s.TravelFromPrevSec ?? undefined,
    arriveTime: s.ArriveTime ?? undefined,
    serviceStartTime: s.ServiceStartTime ?? undefined,
    serviceEndTime: s.ServiceEndTime ?? undefined,
    waitSec: s.WaitSec ?? undefined,
  }));

  const totalDistanceKm = route.Stats?.TotalDistanceM != null ? route.Stats.TotalDistanceM / 1000 : undefined;
  const totalTravelTimeMin = route.Stats?.TotalTravelSec != null ? Math.round(route.Stats.TotalTravelSec / 60) : undefined;
  const serviceTimeMin = route.Stats?.TotalServiceSec != null ? Math.round(route.Stats.TotalServiceSec / 60) : undefined;

  return {
    ...mapRouteSummary(route.Route),
    orderedTaskIds,
    stops,
    totalDistanceKm,
    totalTravelTimeMin,
    totalDurationMin:
      totalTravelTimeMin !== undefined || serviceTimeMin !== undefined
        ? (totalTravelTimeMin ?? 0) + (serviceTimeMin ?? 0)
        : undefined,
  };
}


function getEmailFromToken(token: string) {
  try {
    const [, payload] = token.split('.');
    if (!payload) {
      return null;
    }

    const base64 = payload.replace(/-/g, '+').replace(/_/g, '/');
    const paddedBase64 = `${base64}${'='.repeat((4 - (base64.length % 4)) % 4)}`;
    const decoded = JSON.parse(window.atob(paddedBase64)) as { email?: string };
    return decoded.email ?? null;
  } catch {
    return null;
  }
}
