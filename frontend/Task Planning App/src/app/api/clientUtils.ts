// Exported utility functions from the API client layer for testability.
// These are pure functions extracted from client.ts.

/**
 * Normalizes "HH:MM:SS" time strings to "HH:MM".
 * Returns undefined for null/undefined/empty values.
 */
export function normalizeTime(value?: string | null): string | undefined {
  if (!value) {
    return undefined;
  }
  return value.slice(0, 5);
}

/**
 * Serializes "HH:MM" time strings to "HH:MM:SS" (appends :00).
 * Returns empty string for undefined.
 */
export function serializeTime(value?: string): string {
  if (!value) {
    return '';
  }
  return value.length === 5 ? `${value}:00` : value;
}

/**
 * Extracts the email claim from a JWT access token payload.
 * Returns null if the token is malformed or has no email claim.
 */
export function getEmailFromToken(token: string): string | null {
  try {
    const [, payload] = token.split('.');
    if (!payload) {
      return null;
    }
    const base64 = payload.replace(/-/g, '+').replace(/_/g, '/');
    const paddedBase64 = `${base64}${'='.repeat((4 - (base64.length % 4)) % 4)}`;
    const decoded = JSON.parse(atob(paddedBase64)) as { email?: string };
    return decoded.email ?? null;
  } catch {
    return null;
  }
}

export interface ApiTask {
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

export interface MappedTask {
  id: string;
  title: string;
  address: string;
  latitude: number;
  longitude: number;
  duration: number;
  timeWindowStart?: string;
  timeWindowEnd?: string;
  order: number;
  completed: boolean;
}

/**
 * Maps a PascalCase API task response to a camelCase domain Task.
 */
export function mapTaskFromApi(task: ApiTask): MappedTask {
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
