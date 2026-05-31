import { LRUCache } from 'lru-cache';
import type { DistanceCell } from './routeOptimizer';

// Клиентский in-memory кэш рёбер матрицы расстояний Yandex.
// Живёт на фронте, потому что бесплатная маршрутизация Яндекса (JS API 2.1
// MultiRoute) доступна только из браузера; серверный кэш (OSRM) — отдельный.
//
// Ключ — строка из режима транспорта и округлённых координат пары точек.
// Координаты сравниваются как строки (toFixed), а не как float, поэтому
// погрешность плавающей точки и джиттер геокодера не дают ложных промахов.

export interface CachePoint {
  latitude: number | null;
  longitude: number | null;
}

const COORD_PRECISION = 5; // 5 знаков ≈ сетка ~1 м
const cache = new LRUCache<string, DistanceCell>({
  max: 50_000,
  ttl: 1000 * 60 * 60 * 24, // 24 ч, как у серверного CachedProvider
});

function coordKey(lat: number, lng: number): string {
  return `${lat.toFixed(COORD_PRECISION)},${lng.toFixed(COORD_PRECISION)}`;
}

function edgeKey(mode: string, from: CachePoint, to: CachePoint): string | null {
  if (from.latitude == null || from.longitude == null) return null;
  if (to.latitude == null || to.longitude == null) return null;
  return `${mode}|${coordKey(from.latitude, from.longitude)}->${coordKey(to.latitude, to.longitude)}`;
}

export function getEdge(mode: string, from: CachePoint, to: CachePoint): DistanceCell | undefined {
  const key = edgeKey(mode, from, to);
  return key ? cache.get(key) : undefined;
}

export function setEdge(mode: string, from: CachePoint, to: CachePoint, cell: DistanceCell): void {
  const key = edgeKey(mode, from, to);
  if (key) cache.set(key, cell);
}

// Для тестов: очистка между кейсами.
export function clearDistanceCache(): void {
  cache.clear();
}
