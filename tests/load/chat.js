import http from 'k6/http';
import { check } from 'k6';

const base = __ENV.BASE_URL || 'http://127.0.0.1:8080';
const key = __ENV.API_KEY;
const model = __ENV.MODEL || 'gpt-4.1';
const rps = Number(__ENV.RPS || 100);
const duration = __ENV.DURATION || '30s';

export const options = {
  scenarios: {
    chat: {
      executor: 'constant-arrival-rate',
      rate: rps,
      timeUnit: '1s',
      duration,
      preAllocatedVUs: Math.max(20, Math.ceil(rps / 2)),
      maxVUs: Math.max(100, rps * 3),
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.005'],
    http_req_duration: ['p(95)<2000'],
  },
};

export default function () {
  const response = http.post(
    `${base}/v1/chat/completions`,
    JSON.stringify({
      model,
      messages: [{ role: 'user', content: 'Responda apenas OK.' }],
      max_tokens: 8,
    }),
    { headers: { Authorization: `Bearer ${key}`, 'Content-Type': 'application/json' } },
  );
  check(response, {
    'status is 200': (r) => r.status === 200,
    'has completion': (r) => r.body.includes('chat.completion'),
  });
}
