// Нагрузочное тестирование REST API по открытой модели (constant-arrival-rate).
//
// Проверяется поведение ОДНОЙ ручки при заданном целевом RPS:
//   ENDPOINT=tasks_get   GET  /api/tasks            (чтение)
//   ENDPOINT=tasks_post  POST /api/tasks            (запись)
//   ENDPOINT=optimize    POST /api/routes/optimize  (алгоритм + БД, кэш матрицы прогрет)
//
// Параметры через переменные окружения:
//   RATE      целевой RPS                    (по умолчанию 50)
//   DURATION  длительность замера            (по умолчанию 30s)
//   WARMUP    длительность прогрева           (по умолчанию 5s, метрики отбрасываются)
//   POOL      число пользователей в пуле     (по умолчанию 30)
//   TASKS     задач на пользователя          (по умолчанию 8)
//
// Каждая итерация делает РОВНО один запрос к измеряемой ручке: токены и задачи
// готовятся в setup(), поэтому в измерение не попадают логин и сетевые накладные.
// Результат точки пишется в results/<endpoint>_<rate>.json.

import http from 'k6/http';
import { Trend, Rate, Counter } from 'k6/metrics';
import exec from 'k6/execution';

const BASE_URL = 'http://localhost:8080/api';

const ENDPOINT = __ENV.ENDPOINT || 'tasks_get';
const RATE = __ENV.RATE ? parseInt(__ENV.RATE, 10) : 50;
const DURATION = __ENV.DURATION || '30s';
const WARMUP = __ENV.WARMUP || '5s';
const POOL = __ENV.POOL ? parseInt(__ENV.POOL, 10) : 30;
const TASKS_PER_USER = __ENV.TASKS ? parseInt(__ENV.TASKS, 10) : 8;

// Фиксированный набор координат (Санкт-Петербург). Все пользователи получают
// один и тот же набор, поэтому после прогрева матрица расстояний полностью
// лежит в кэше приложения и ручка optimize не ходит во внешний OSRM.
const COORDS = [
  [59.9386, 30.3141], [59.9311, 30.3609], [59.9606, 30.3586],
  [59.9530, 30.3050], [59.9270, 30.3380], [59.9430, 30.2860],
  [59.9190, 30.3220], [59.9645, 30.3100], [59.9075, 30.3340],
  [59.9490, 30.3760],
];

const START_TIME_UNIX = 1735732800; // фиксированное время старта маршрута

const latency = new Trend('endpoint_latency_ms', true);
const errors = new Rate('endpoint_errors');
const reqs = new Counter('endpoint_reqs');

function durationSeconds(s) {
  if (s.endsWith('ms')) return parseFloat(s) / 1000;
  if (s.endsWith('m')) return parseFloat(s) * 60;
  if (s.endsWith('s')) return parseFloat(s);
  return parseFloat(s);
}

// preAllocatedVUs/maxVUs масштабируются под RATE: при росте задержки в точке
// насыщения нужно больше VU, чтобы выдерживать целевую частоту запросов.
const preAllocatedVUs = Math.min(Math.max(RATE, 50), 2000);
const maxVUs = Math.min(RATE * 4, 8000);

export const options = {
  scenarios: {
    warmup: {
      executor: 'constant-arrival-rate',
      rate: Math.max(1, Math.round(RATE / 4)),
      timeUnit: '1s',
      duration: WARMUP,
      preAllocatedVUs,
      maxVUs,
      exec: 'measure',
      tags: { phase: 'warmup' },
    },
    measure: {
      executor: 'constant-arrival-rate',
      rate: RATE,
      timeUnit: '1s',
      duration: DURATION,
      startTime: WARMUP,
      preAllocatedVUs,
      maxVUs,
      exec: 'measure',
      tags: { phase: 'measure' },
    },
  },
  summaryTrendStats: ['avg', 'min', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'],
  thresholds: {
    'endpoint_latency_ms{phase:measure}': ['p(95)<2000'],
    'endpoint_errors{phase:measure}': ['rate<0.01'],
    'endpoint_reqs{phase:measure}': ['count>=0'],
  },
};

function jsonHeaders(token) {
  return {
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
  };
}

// setup: регистрирует пул пользователей, сеет каждому одинаковый набор задач
// и прогревает кэш матрицы расстояний (один optimize на пользователя).
export function setup() {
  const nonce = Date.now();
  const users = [];

  for (let u = 0; u < POOL; u++) {
    const email = `loadtest-${nonce}-${u}@example.com`;
    const reg = http.post(
      `${BASE_URL}/auth/register`,
      JSON.stringify({ email, password: 'LoadTest123!' }),
      { headers: { 'Content-Type': 'application/json' } },
    );
    if (reg.status !== 200) {
      throw new Error(`register failed (${reg.status}): ${reg.body}`);
    }
    const token = reg.json('accessToken');

    const tasks = [];
    for (let i = 0; i < TASKS_PER_USER; i++) {
      const [lat, lng] = COORDS[i % COORDS.length];
      tasks.push({
        title: `Seed-${u}-${i}`,
        addressText: 'Санкт-Петербург',
        latitude: lat,
        longitude: lng,
        durationMin: 15,
        sortIndex: i,
      });
    }
    const batch = http.post(`${BASE_URL}/tasks/batch`, JSON.stringify({ tasks }), jsonHeaders(token));
    if (batch.status !== 200) {
      throw new Error(`batch failed (${batch.status}): ${batch.body}`);
    }
    const taskIds = batch.json().map((t) => t.ID);

    // прогрев кэша матрицы расстояний для этого набора координат
    http.post(
      `${BASE_URL}/routes/optimize`,
      JSON.stringify({ taskIds, startTimeUnix: START_TIME_UNIX }),
      jsonHeaders(token),
    );

    users.push({ token, taskIds });
  }

  return { users };
}

export function measure(data) {
  const users = data.users;
  const user = users[exec.scenario.iterationInTest % users.length];
  let res;

  if (ENDPOINT === 'tasks_get') {
    res = http.get(`${BASE_URL}/tasks`, jsonHeaders(user.token));
  } else if (ENDPOINT === 'tasks_post') {
    const [lat, lng] = COORDS[exec.scenario.iterationInTest % COORDS.length];
    res = http.post(
      `${BASE_URL}/tasks`,
      JSON.stringify({
        title: `Load-${exec.scenario.iterationInTest}`,
        addressText: 'Санкт-Петербург',
        latitude: lat,
        longitude: lng,
        durationMin: 15,
        sortIndex: 0,
      }),
      jsonHeaders(user.token),
    );
  } else if (ENDPOINT === 'optimize') {
    res = http.post(
      `${BASE_URL}/routes/optimize`,
      JSON.stringify({ taskIds: user.taskIds, startTimeUnix: START_TIME_UNIX }),
      jsonHeaders(user.token),
    );
  } else {
    throw new Error(`unknown ENDPOINT: ${ENDPOINT}`);
  }

  const ok = res.status >= 200 && res.status < 300;
  latency.add(res.timings.duration);
  errors.add(ok ? 0 : 1);
  reqs.add(1);
}

export function handleSummary(data) {
  const lat = data.metrics['endpoint_latency_ms{phase:measure}'];
  const err = data.metrics['endpoint_errors{phase:measure}'];
  const cnt = data.metrics['endpoint_reqs{phase:measure}'];

  const count = cnt ? cnt.values.count : 0;
  const durSec = durationSeconds(DURATION);
  const lv = lat ? lat.values : {};

  const point = {
    endpoint: ENDPOINT,
    targetRate: RATE,
    achievedRate: +(count / durSec).toFixed(1),
    requests: count,
    avg: lv.avg != null ? +lv.avg.toFixed(1) : null,
    p90: lv['p(90)'] != null ? +lv['p(90)'].toFixed(1) : null,
    p95: lv['p(95)'] != null ? +lv['p(95)'].toFixed(1) : null,
    p99: lv['p(99)'] != null ? +lv['p(99)'].toFixed(1) : null,
    max: lv.max != null ? +lv.max.toFixed(1) : null,
    errorRatePct: err ? +(err.values.rate * 100).toFixed(2) : 0,
  };

  const line =
    `\n[${point.endpoint}] target=${point.targetRate} rps  ` +
    `achieved=${point.achievedRate} rps  ` +
    `avg=${point.avg}ms  p95=${point.p95}ms  p99=${point.p99}ms  ` +
    `err=${point.errorRatePct}%  (n=${point.requests})\n`;

  const out = {};
  out['stdout'] = line;
  out[`results/${ENDPOINT}_${RATE}.json`] = JSON.stringify(point, null, 2);
  return out;
}
