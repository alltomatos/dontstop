import http from 'k6/http';
import { check } from 'k6';

const base = __ENV.BASE_URL || 'http://127.0.0.1:8080';
const key = __ENV.API_KEY;
const model = __ENV.MODEL || 'gpt-4.1';

export const options = {
  scenarios: {
    sse: {
      executor: 'constant-vus',
      vus: Number(__ENV.VUS || 100),
      duration: __ENV.DURATION || '30s',
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.005'],
  },
};

export default function () {
  const response = http.post(
    `${base}/v1/chat/completions`,
    JSON.stringify({
      model,
      stream: true,
      messages: [{ role: 'user', content: 'Responda apenas OK.' }],
      max_tokens: 8,
    }),
    { headers: { Authorization: `Bearer ${key}`, 'Content-Type': 'application/json' } },
  );
  check(response, {
    'status is 200': (r) => r.status === 200,
    'has done marker': (r) => r.body.includes('[DONE]'),
  });
}
