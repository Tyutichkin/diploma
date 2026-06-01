import { Task } from '../types/task';
import { loadYandexMaps } from './yandexMaps';
import { roundUpSecToMinute } from './time';
import { getEdge, setEdge } from './distanceCache';

export interface GeocodeSuggestion {
  lat: number;
  lng: number;
  displayName: string;
}

export interface DistanceCell {
  distanceM: number;
  durationSec: number;
}

export class YandexMatrixUnavailableError extends Error {
  constructor(message = 'Yandex routing unavailable') {
    super(message);
    this.name = 'YandexMatrixUnavailableError';
  }
}

// Число попыток на одну пару точек и базовая задержка между ними (линейный backoff).
const YANDEX_REQUEST_ATTEMPTS = 3;
const YANDEX_RETRY_DELAY_MS = 400;

const delay = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

function requestYandexEdge(
  from: Task,
  to: Task,
  mode: 'auto' | 'pedestrian',
  fallbackSec: number,
): Promise<DistanceCell> {
  return new Promise<DistanceCell>((resolve, reject) => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const mr = new (window.ymaps as any).multiRouter.MultiRoute(
      {
        referencePoints: [
          [from.latitude!, from.longitude!],
          [to.latitude!, to.longitude!],
        ],
        params: { routingMode: mode },
      },
      {},
    );

    mr.model.events.once('requestsuccess', () => {
      try {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const activeRoute: any = mr.getActiveRoute();
        if (activeRoute) {
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          const distObj: any = activeRoute.properties.get('distance');
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          const durObj: any = activeRoute.properties.get('duration');
          resolve({
            distanceM: Math.round(typeof distObj?.value === 'number' ? distObj.value : 0),
            durationSec: roundUpSecToMinute(
              typeof durObj?.value === 'number' ? durObj.value : fallbackSec,
            ),
          });
          return;
        }

        // fallback: читаем модельные маршруты напрямую
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const modelRoutes: any[] = mr.model.getRoutes();
        if (modelRoutes.length > 0) {
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          const legs: any[] = modelRoutes[0].getLegs();
          let totalDistM = 0;
          let totalDurSec = 0;
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          legs.forEach((leg: any) => {
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const d: any = leg.properties?.get?.('distance');
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const dur: any = leg.properties?.get?.('duration');
            totalDistM += typeof d?.value === 'number' ? d.value : 0;
            totalDurSec += typeof dur?.value === 'number' ? dur.value : 0;
          });
          resolve({
            distanceM: Math.round(totalDistM),
            durationSec: totalDurSec > 0 ? roundUpSecToMinute(totalDurSec) : fallbackSec,
          });
          return;
        }

        reject(new Error('no route data'));
      } catch (e) {
        reject(e);
      }
    });

    mr.model.events.once('requestfail', () => {
      reject(new Error('routing failed'));
    });
  });
}

async function requestYandexEdgeWithRetries(
  from: Task,
  to: Task,
  mode: 'auto' | 'pedestrian',
  fallbackSec: number,
): Promise<DistanceCell> {
  let lastError: unknown;
  for (let attempt = 1; attempt <= YANDEX_REQUEST_ATTEMPTS; attempt++) {
    try {
      return await requestYandexEdge(from, to, mode, fallbackSec);
    } catch (e) {
      lastError = e;
      if (attempt < YANDEX_REQUEST_ATTEMPTS) await delay(YANDEX_RETRY_DELAY_MS * attempt);
    }
  }
  throw lastError instanceof Error ? lastError : new Error('routing failed');
}

export async function buildYandexDistanceMatrix(
  tasks: Task[],
  routingMode: 'auto' | 'pedestrian' | 'masstransit' = 'auto',
): Promise<DistanceCell[][]> {
  const n = tasks.length;
  if (n === 0) return [];

  try {
    await loadYandexMaps();
  } catch (e) {
    throw new YandexMatrixUnavailableError(
      `Yandex Maps API failed to load: ${e instanceof Error ? e.message : String(e)}`,
    );
  }

  const mode = routingMode === 'masstransit' ? 'auto' : routingMode;

  const fallbackSec = 99_999;
  const matrix: DistanceCell[][] = Array.from({ length: n }, (_, i) =>
    Array.from({ length: n }, (__, j) =>
      i === j ? { distanceM: 0, durationSec: 0 } : { distanceM: 0, durationSec: fallbackSec },
    ),
  );

  const pairs: [number, number][] = [];
  for (let i = 0; i < n; i++) {
    for (let j = 0; j < n; j++) {
      if (i === j) continue;
      const cached = getEdge(mode, tasks[i], tasks[j]);
      if (cached) {
        matrix[i][j] = cached;
        continue;
      }
      pairs.push([i, j]);
    }
  }

  let attempted = 0;
  let failed = 0;

  await Promise.all(
    pairs.map(async ([i, j]) => {
      const from = tasks[i];
      const to = tasks[j];
      if (from.latitude == null || from.longitude == null) return;
      if (to.latitude == null || to.longitude == null) return;

      attempted++;
      try {
        const cell = await requestYandexEdgeWithRetries(from, to, mode, fallbackSec);
        matrix[i][j] = cell;
        setEdge(mode, from, to, cell);
      } catch {
        failed++;
      }
    }),
  );

  if (attempted > 0 && failed === attempted) {
    throw new YandexMatrixUnavailableError(
      `Yandex routing failed for all ${attempted} pairs after ${YANDEX_REQUEST_ATTEMPTS} attempts`,
    );
  }

  return matrix;
}

export async function geocodeAddressSuggestions(address: string): Promise<GeocodeSuggestion[]> {
  const trimmedAddress = address.trim();
  if (!trimmedAddress) return [];

  await loadYandexMaps();

  // ymaps.geocode возвращает Vow-промис, совместимый с .then(resolve, reject)
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const result = await new Promise<any>((resolve, reject) =>
    window.ymaps.geocode(trimmedAddress, { results: 5 }).then(resolve, reject),
  );

  const suggestions: GeocodeSuggestion[] = [];
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  result.geoObjects.each((obj: any) => {
    const coords: [number, number] = obj.geometry.getCoordinates(); // [lat, lon]
    const displayName: string = obj.getAddressLine();
    if (coords && displayName) {
      suggestions.push({ lat: coords[0], lng: coords[1], displayName });
    }
  });

  return suggestions;
}

export async function geocodeAddress(address: string): Promise<{ lat: number; lng: number } | null> {
  const suggestions = await geocodeAddressSuggestions(address);
  if (!suggestions[0]) return null;
  return { lat: suggestions[0].lat, lng: suggestions[0].lng };
}

// ymaps.suggest недоступен на бесплатном тарифе — используется REST.
export async function suggestAddresses(query: string): Promise<GeocodeSuggestion[]> {
  const trimmed = query.trim();
  if (!trimmed) return [];

  const apiKey =
    (import.meta.env.VITE_YANDEX_GEOCODER_KEY as string | undefined) ||
    (import.meta.env.VITE_YANDEX_MAPS_KEY as string | undefined);
  if (!apiKey) return [];

  const url = `https://geocode-maps.yandex.ru/1.x/?apikey=${encodeURIComponent(apiKey)}&geocode=${encodeURIComponent(trimmed)}&format=json&results=5&lang=ru_RU`;
  const resp = await fetch(url);
  if (!resp.ok) return [];
  const json = await resp.json();

  const members: unknown[] = json?.response?.GeoObjectCollection?.featureMember ?? [];
  const suggestions: GeocodeSuggestion[] = [];
  for (const m of members) {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const obj = (m as any).GeoObject;
    const text: string = obj?.metaDataProperty?.GeocoderMetaData?.text;
    const posStr: string = obj?.Point?.pos; // "lon lat"
    if (!text || !posStr) continue;
    const [lonStr, latStr] = posStr.split(' ');
    const lat = parseFloat(latStr);
    const lng = parseFloat(lonStr);
    if (isNaN(lat) || isNaN(lng)) continue;
    suggestions.push({ lat, lng, displayName: text });
  }
  return suggestions;
}
