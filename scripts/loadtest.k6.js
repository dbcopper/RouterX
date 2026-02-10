import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  vus: 5,
  duration: '10s',
};

export default function () {
  const url = 'http://localhost:8080/v1/chat/completions';
  const payload = JSON.stringify({
    model: 'gpt-4.1-mini',
    messages: [
      { role: 'user', content: [{ type: 'text', text: 'Hello' }] }
    ]
  });
  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': 'Bearer demo_key_fake_123456'
    }
  };
  const res = http.post(url, payload, params);
  check(res, { 'status is 200': (r) => r.status === 200 });
  sleep(1);
}
