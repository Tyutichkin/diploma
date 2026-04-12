import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend, Rate, Counter } from 'k6/metrics';

const responseTime = new Trend('response_time_ms');
const errorRate = new Rate('error_rate');
const requestCount = new Counter('total_requests');

export const options = {
  scenarios: {
    load_10: {
      executor: 'constant-vus',
      vus: 10,
      duration: '60s',
      startTime: '0s',
      tags: { scenario: '10_users' },
    },
    load_50: {
      executor: 'constant-vus',
      vus: 50,
      duration: '60s',
      startTime: '70s',
      tags: { scenario: '50_users' },
    },
    load_100: {
      executor: 'constant-vus',
      vus: 100,
      duration: '60s',
      startTime: '140s',
      tags: { scenario: '100_users' },
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<2000'],
    error_rate: ['rate<0.01'],
  },
};

const BASE_URL = 'http://localhost:8080/api';

function login() {
  const payload = JSON.stringify({
    email: 'loadtest@example.com',
    password: 'LoadTest123!',
  });
  const params = { headers: { 'Content-Type': 'application/json' } };
  const res = http.post(`${BASE_URL}/auth/login`, payload, params);
  check(res, { 'login 200': (r) => r.status === 200 });
  if (res.status === 200) {
    return res.json('accessToken');
  }
  return null;
}

export default function () {
  // 1. Login
  const token = login();
  requestCount.add(1);
  if (!token) {
    errorRate.add(1);
    return;
  }
  errorRate.add(0);

  const authHeaders = {
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
  };

  // 2. Get tasks list
  const listRes = http.get(`${BASE_URL}/tasks`, authHeaders);
  responseTime.add(listRes.timings.duration);
  requestCount.add(1);
  const listOk = check(listRes, { 'GET /tasks 200': (r) => r.status === 200 });
  errorRate.add(listOk ? 0 : 1);

  // 3. Create a task
  const taskPayload = JSON.stringify({
    title: `Task-${__VU}-${__ITER}`,
    addressText: 'Санкт-Петербург, Невский проспект, 1',
    latitude: 59.9343,
    longitude: 30.3351,
    durationMin: 30,
    sortIndex: 0,
  });
  const createRes = http.post(`${BASE_URL}/tasks`, taskPayload, authHeaders);
  responseTime.add(createRes.timings.duration);
  requestCount.add(1);
  const createOk = check(createRes, { 'POST /tasks 200': (r) => r.status === 200 });
  errorRate.add(createOk ? 0 : 1);

  sleep(1);
}
