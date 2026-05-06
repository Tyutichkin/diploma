import { Task } from '../types/task';
import { loadYandexMaps } from './yandexMaps';
import { roundUpSecToMinute } from './time';

export interface GeocodeSuggestion {
  lat: number;
  lng: number;
  displayName: string;
}

export interface DistanceCell {
  distanceM: number;
  durationSec: number;
}

// Матрица n×n через ymaps multiRouter. matrix[i][j] — стоимость i→j, диагональ = 0.
// Берём тот же движок, что рисует маршрут на карте, чтобы оптимизация и отображение совпадали.
// masstransit не даёт надёжных времён — подменяется на 'auto'.
export async function buildYandexDistanceMatrix(
  tasks: Task[],
  routingMode: 'auto' | 'pedestrian' | 'masstransit' = 'auto',
): Promise<DistanceCell[][]> {
  const n = tasks.length;
  if (n === 0) return [];

  await loadYandexMaps();

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
      if (i !== j) pairs.push([i, j]);
    }
  }

  await Promise.all(
    pairs.map(async ([i, j]) => {
      const from = tasks[i];
      const to = tasks[j];
      if (from.latitude == null || from.longitude == null) return;
      if (to.latitude == null || to.longitude == null) return;

      try {
        const { distanceM, durationSec } = await new Promise<{
          distanceM: number;
          durationSec: number;
        }>((resolve, reject) => {
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
                  // Округляем секунды вверх до минуты — внутри системы работаем только
                  // с минутно-выровненными значениями (см. utils/time.ts).
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

        matrix[i][j] = { distanceM, durationSec };
      } catch {
        // fallback-время оставляет пару «дорогой», алгоритм её избегает
      }
    }),
  );

  return matrix;
}

// Геокодирование через ymaps.geocode — до 5 вариантов адреса с координатами.
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

// Подсказки адресов через Yandex Geocoder REST API (до 5 вариантов).
// ymaps.suggest недоступен на бесплатном тарифе, поэтому используется REST.
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
