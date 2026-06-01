// Быстрая проверка одной точки нагрузки по открытой модели (constant-arrival-rate).
// Прогоняет одну ручку при фиксированном целевом RPS — удобно для отладки.
//
//   k6 run -e ENDPOINT=tasks_get -e RATE=200 load_test_single.js
//
// Полная методика (прогрев + замер + запись результата) — в load_test.js,
// который вызывается из run_load.sh для свипа по RPS.

import http from 'k6/http';
import { check } from 'k6';
import exec from 'k6/execution';

const BASE_URL = 'http://localhost:8080/api';

const ENDPOINT = __ENV.ENDPOINT || 'tasks_get';
const RATE = __ENV.RATE ? parseInt(__ENV.RATE, 10) : 50;
const DURATION = __ENV.DURATION || '30s';
const POOL = __ENV.POOL ? parseInt(__ENV.POOL, 10) : 20;
const TASKS_PER_USER = __ENV.TASKS ? parseInt(__ENV.TASKS, 10) : 8;

const COORDS = [
  [59.9386, 30.3141], [59.9311, 30.3609], [59.9606, 30.3586],
  [59.9530, 30.3050], [59.9270, 30.3380], [59.9430, 30.2860],
  [59.9190, 30.3220], [59.9645, 30.3100],
];
const START_TIME_UNIX = 1735732800;

export const options = {
  scenarios: {
    measure: {
      executor: 'constant-arrival-rate',
      rate: RATE,
      timeUnit: '1s',
      duration: DURATION,
      preAllocatedVUs: Math.min(Math.max(RATE, 20), 600),
      maxVUs: Math.min(RATE * 4, 2500),
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<2000'],
    http_req_failed: ['rate<0.01'],
  },
};

function jsonHeaders(token) {
  return { headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` } };
}

export function setup() {
  const nonce = Date.now();
  const users = [];
  for (let u = 0; u < POOL; u++) {
    const email = `loadtest-single-${nonce}-${u}@example.com`;
    const reg = http.post(
      `${BASE_URL}/auth/register`,
      JSON.stringify({ email, password: 'LoadTest123!' }),
      { headers: { 'Content-Type': 'application/json' } },
    );
    const token = reg.json('accessToken');
    const tasks = [];
    for (let i = 0; i < TASKS_PER_USER; i++) {
      const [lat, lng] = COORDS[i % COORDS.length];
      tasks.push({ title: `Seed-${u}-${i}`, addressText: 'Санкт-Петербург', latitude: lat, longitude: lng, durationMin: 15, sortIndex: i });
    }
    const batch = http.post(`${BASE_URL}/tasks/batch`, JSON.stringify({ tasks }), jsonHeaders(token));
    const taskIds = batch.json().map((t) => t.ID);
    http.post(`${BASE_URL}/routes/optimize`, JSON.stringify({ taskIds, startTimeUnix: START_TIME_UNIX }), jsonHeaders(token));
    users.push({ token, taskIds });
  }
  return { users };
}

export default function (data) {
  const user = data.users[exec.scenario.iterationInTest % data.users.length];
  let res;
  if (ENDPOINT === 'tasks_get') {
    res = http.get(`${BASE_URL}/tasks`, jsonHeaders(user.token));
  } else if (ENDPOINT === 'tasks_post') {
    const [lat, lng] = COORDS[exec.scenario.iterationInTest % COORDS.length];
    res = http.post(`${BASE_URL}/tasks`, JSON.stringify({ title: `Load-${exec.scenario.iterationInTest}`, addressText: 'Санкт-Петербург', latitude: lat, longitude: lng, durationMin: 15, sortIndex: 0 }), jsonHeaders(user.token));
  } else {
    res = http.post(`${BASE_URL}/routes/optimize`, JSON.stringify({ taskIds: user.taskIds, startTimeUnix: START_TIME_UNIX }), jsonHeaders(user.token));
  }
  check(res, { 'status 2xx': (r) => r.status >= 200 && r.status < 300 });
}
