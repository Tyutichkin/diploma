import { Task } from '../types/task';
import { loadYandexMaps } from './yandexMaps';

export interface GeocodeSuggestion {
  lat: number;
  lng: number;
  displayName: string;
}

export interface DistanceCell {
  distanceM: number;
  durationSec: number;
}

/**
 * Строит матрицу расстояний n×n для списка задач через Яндекс Маршруты JS API.
 *
 * Используется тот же Yandex Maps JS API 2.1 что и для визуализации маршрута
 * на карте, поэтому данные оптимизации (порядок задач) согласованы с тем,
 * что пользователь видит на экране (учёт пробок, реальная дорожная сеть,
 * выбранный режим транспорта).
 *
 * matrix[i][j] = стоимость проезда от tasks[i] до tasks[j].
 * Диагональные элементы (i === j) равны нулю.
 *
 * @param tasks        Список задач с заполненными координатами.
 * @param routingMode  Режим маршрутизации: 'auto' | 'pedestrian' | 'masstransit'.
 *                     Для masstransit используется 'auto' как приближение,
 *                     поскольку ymaps.route не возвращает надёжные данные
 *                     по времени для общественного транспорта.
 */
export async function buildYandexDistanceMatrix(
  tasks: Task[],
  routingMode: 'auto' | 'pedestrian' | 'masstransit' = 'auto',
): Promise<DistanceCell[][]> {
  const n = tasks.length;
  if (n === 0) return [];

  await loadYandexMaps();

  // masstransit через multiRouter.MultiRoute не возвращает надёжные данные
  // по времени для построения матрицы — используем 'auto' как приближение.
  const mode = routingMode === 'masstransit' ? 'auto' : routingMode;

  // Инициализируем матрицу нулями на диагонали и заглушками для остальных.
  const fallbackSec = 99_999;
  const matrix: DistanceCell[][] = Array.from({ length: n }, (_, i) =>
    Array.from({ length: n }, (__, j) =>
      i === j ? { distanceM: 0, durationSec: 0 } : { distanceM: 0, durationSec: fallbackSec },
    ),
  );

  // Запрашиваем все пары параллельно (n*(n-1) запросов).
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
          // Используем multiRouter.MultiRoute — тот же движок, что рисует
          // маршрут на карте, поэтому данные оптимизации согласованы
          // с отображением.
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
              // getActiveRoute() возвращает активный маршрут с полями:
              //   properties.get('distance').value — метры
              //   properties.get('duration').value  — секунды
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              const activeRoute: any = mr.getActiveRoute();
              if (activeRoute) {
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                const distObj: any = activeRoute.properties.get('distance');
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                const durObj: any = activeRoute.properties.get('duration');
                resolve({
                  distanceM: Math.round(typeof distObj?.value === 'number' ? distObj.value : 0),
                  durationSec: Math.round(
                    typeof durObj?.value === 'number' ? durObj.value : fallbackSec,
                  ),
                });
                return;
              }

              // Запасной путь: читаем из модельных маршрутов напрямую
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
                  durationSec: totalDurSec > 0 ? Math.round(totalDurSec) : fallbackSec,
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
        // Оставляем fallback-значение; алгоритм использует большое время,
        // чтобы избегать этой пары.
      }
    }),
  );

  return matrix;
}

/**
 * Геокодирование через Yandex Maps JS API 2.1 (ymaps.geocode).
 * Возвращает до 5 вариантов адреса с координатами.
 */
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

/**
 * Возвращает подсказки адресов через Yandex Geocoder REST API.
 * Возвращает до 5 вариантов с координатами — ymaps.suggest недоступен на бесплатном тарифе.
 */
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
