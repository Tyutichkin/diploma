import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  vus: __ENV.VUS ? parseInt(__ENV.VUS) : 10,
  duration: '60s',
  thresholds: {
    http_req_duration: ['p(95)<2000'],
    http_req_failed: ['rate<0.01'],
  },
};

const BASE_URL = 'http://localhost:8080/api';

function login() {
  const payload = JSON.stringify({
    email: 'loadtest@example.com',
    password: 'LoadTest123!',
  });
  const res = http.post(`${BASE_URL}/auth/login`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });
  check(res, { 'login 200': (r) => r.status === 200 });
  return res.status === 200 ? res.json('accessToken') : null;
}

export default function () {
  const token = login();
  if (!token) return;

  const authHeaders = {
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
  };

  const listRes = http.get(`${BASE_URL}/tasks`, authHeaders);
  check(listRes, { 'GET /tasks 200': (r) => r.status === 200 });

  const createRes = http.post(
    `${BASE_URL}/tasks`,
    JSON.stringify({
      title: `Task-${__VU}-${__ITER}`,
      addressText: 'Санкт-Петербург, Невский проспект, 1',
      latitude: 59.9343,
      longitude: 30.3351,
      durationMin: 30,
      sortIndex: 0,
    }),
    authHeaders
  );
  check(createRes, { 'POST /tasks 200': (r) => r.status === 200 });

  sleep(1);
}
