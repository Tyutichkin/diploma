export function roundUpSecToMinute(sec: number): number {
  if (!Number.isFinite(sec) || sec <= 0) return 0;
  return Math.ceil(sec / 60) * 60;
}

export function secToMinCeil(sec: number): number {
  if (!Number.isFinite(sec) || sec <= 0) return 0;
  return Math.ceil(sec / 60);
}
