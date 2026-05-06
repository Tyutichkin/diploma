// Проектное правило: длительности всегда округляются вверх до целой минуты.
// Применяй на границах ввода (Yandex matrix, REST-данные с бэка) и в отображении
// (защитно — на случай, если где-то в цепочке секунды просочились).

export function roundUpSecToMinute(sec: number): number {
  if (!Number.isFinite(sec) || sec <= 0) return 0;
  return Math.ceil(sec / 60) * 60;
}

export function secToMinCeil(sec: number): number {
  if (!Number.isFinite(sec) || sec <= 0) return 0;
  return Math.ceil(sec / 60);
}
