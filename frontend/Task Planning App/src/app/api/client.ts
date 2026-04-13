import { AuthSession } from '../types/auth';
import { Task } from '../types/task';
import { SavedRouteDetails, SavedRouteSummary } from '../types/route';
import { DistanceCell } from '../utils/routeOptimizer';

const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080/api').replace(/\/$/, '');
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
  Latitude: number;
  Longitude: number;
  DurationMin: number;
  WindowStart?: string | null;
  WindowEnd?: string | null;
  SortIndex: number;
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
  return tasks.map((task) => mapTaskFromApi(task));
}

export interface SaveTaskInput {
  title: string;
  address: string;
  latitude: number;
  longitude: number;
  duration: number;
  timeWindowStart?: string;
  timeWindowEnd?: string;
  sortIndex: number;
}

export async function createTask(input: SaveTaskInput, options: AuthorizedRequestOptions) {
  const task = await requestWithAuth<ApiTask>(
    '/tasks',
    {
      method: 'POST',
      body: JSON.stringify({
        title: input.title,
        addressText: input.address,
        latitude: input.latitude,
        longitude: input.longitude,
        durationMin: input.duration,
        windowStart: serializeTime(input.timeWindowStart),
        windowEnd: serializeTime(input.timeWindowEnd),
        sortIndex: input.sortIndex,
      }),
    },
    options,
  );

  return mapTaskFromApi(task);
}

export async function updateTask(taskId: string, input: Partial<SaveTaskInput>, options: AuthorizedRequestOptions) {
  const task = await requestWithAuth<ApiTask>(
    `/tasks/${taskId}`,
    {
      method: 'PATCH',
      body: JSON.stringify({
        title: input.title,
        addressText: input.address,
        latitude: input.latitude,
        longitude: input.longitude,
        durationMin: input.duration,
        windowStart: input.timeWindowStart === undefined ? undefined : serializeTime(input.timeWindowStart),
        windowEnd: input.timeWindowEnd === undefined ? undefined : serializeTime(input.timeWindowEnd),
        sortIndex: input.sortIndex,
      }),
    },
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

// optimizeRoute sends task IDs and a pre-computed Yandex distance matrix to
// the backend.  The backend builds an in-memory graph, runs the optimisation
// algorithm with time-window constraints, and returns the ordered stop list.
//
// Passing a matrix computed via buildYandexDistanceMatrix() ensures that the
// distances used in optimisation are identical to those shown on the map.
// When distanceMatrix is omitted the backend falls back to its own routing API.
//
// startTaskId and endTaskId pin the first and last stops respectively.
// precedenceConstraints enforces that certain tasks are visited before others.
export async function optimizeRoute(
  taskIds: string[],
  options: AuthorizedRequestOptions,
  startTimeMins = 540,
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
        startTimeMins,
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
  return routes.map((route) => mapRouteSummary(route));
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
    address: task.AddressText,
    latitude: task.Latitude,
    longitude: task.Longitude,
    duration: task.DurationMin,
    timeWindowStart: normalizeTime(task.WindowStart),
    timeWindowEnd: normalizeTime(task.WindowEnd),
    order: task.SortIndex,
    completed: false,
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
  const orderedTaskIds = [...(route.Stops ?? [])]
    .sort((left, right) => left.Position - right.Position)
    .map((stop) => stop.TaskID);

  const totalDistanceKm = route.Stats?.TotalDistanceM ? route.Stats.TotalDistanceM / 1000 : undefined;
  const totalTravelTimeMin = route.Stats?.TotalTravelSec ? Math.round(route.Stats.TotalTravelSec / 60) : undefined;
  const serviceTimeMin = route.Stats?.TotalServiceSec ? Math.round(route.Stats.TotalServiceSec / 60) : undefined;

  return {
    ...mapRouteSummary(route.Route),
    orderedTaskIds,
    totalDistanceKm,
    totalTravelTimeMin,
    totalDurationMin:
      totalTravelTimeMin !== undefined || serviceTimeMin !== undefined
        ? (totalTravelTimeMin ?? 0) + (serviceTimeMin ?? 0)
        : undefined,
  };
}

function normalizeTime(value?: string | null) {
  if (!value) {
    return undefined;
  }

  return value.slice(0, 5);
}

function serializeTime(value?: string) {
  if (!value) {
    return '';
  }

  return value.length === 5 ? `${value}:00` : value;
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
