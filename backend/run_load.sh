#!/usr/bin/env bash
# Прогон нагрузочного теста: каждая основная ручка прогоняется при наборе
# целевых RPS (открытая модель). Каждая точка пишется в results/<ep>_<rate>.json.
# Итог сводится в results/summary.csv.
set -uo pipefail
cd "$(dirname "$0")"
mkdir -p results

DURATION="${DURATION:-30s}"
WARMUP="${WARMUP:-5s}"
POOL="${POOL:-30}"
TASKS="${TASKS:-8}"

run_point() {
  local ep="$1" rate="$2"
  echo ">>> ${ep} @ ${rate} RPS (duration=${DURATION}, warmup=${WARMUP})"
  k6 run --quiet \
    -e ENDPOINT="$ep" -e RATE="$rate" \
    -e DURATION="$DURATION" -e WARMUP="$WARMUP" \
    -e POOL="$POOL" -e TASKS="$TASKS" \
    load_test.js
}

# endpoint : целевые RPS
run_sweep() {
  local ep="$1"; shift
  for rate in "$@"; do
    run_point "$ep" "$rate"
  done
}

# Диапазоны доведены до точки насыщения каждой ручки (см. раздел диплома).
run_sweep tasks_get  50 100 200 400 800 1600 3200 6400 12800
run_sweep tasks_post 25 50 100 200 400 800 1600 3200 6400
run_sweep optimize   5 10 20 40 80 160 320 640 1280 2560 5120

echo ""
echo "=== Сводка (results/summary.csv) ==="
python3 - <<'PY'
import json, glob, csv, os

rows = []
for f in glob.glob("results/*.json"):
    with open(f) as fh:
        rows.append(json.load(fh))

order = {"tasks_get": 0, "tasks_post": 1, "optimize": 2}
rows.sort(key=lambda r: (order.get(r["endpoint"], 9), r["targetRate"]))

cols = ["endpoint", "targetRate", "achievedRate", "requests",
        "avg", "p90", "p95", "p99", "max", "errorRatePct"]
with open("results/summary.csv", "w", newline="") as fh:
    w = csv.DictWriter(fh, fieldnames=cols)
    w.writeheader()
    for r in rows:
        w.writerow({c: r.get(c) for c in cols})

hdr = f'{"endpoint":12} {"target":>7} {"achiev":>7} {"avg":>7} {"p95":>7} {"p99":>7} {"err%":>6}'
print(hdr)
print("-" * len(hdr))
for r in rows:
    print(f'{r["endpoint"]:12} {r["targetRate"]:>7} {r["achievedRate"]:>7.1f} '
          f'{r["avg"]:>7.1f} {r["p95"]:>7.1f} {r["p99"]:>7.1f} {r["errorRatePct"]:>6.2f}')
PY
