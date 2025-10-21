import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Custom metrics
export const blockedRequests = new Rate('blocked_requests');

export let options = {
  stages: [
    { duration: '30s', target: 100 }, // Ramp up
    { duration: '1m', target: 100 },  // Stay at 100 users
    { duration: '30s', target: 0 },   // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95% das requests < 500ms
    blocked_requests: ['rate>0.3'],   // >30% das requests bloqueadas
  },
};

export default function() {
  // Teste com requests normais (sem token)
  let response = http.get('http://localhost:8080/');
  
  // Verifica se foi bloqueada (429) ou permitida (200)
  let wasBlocked = response.status === 429;
  blockedRequests.add(wasBlocked);
  
  check(response, {
    'status is 200 or 429': (r) => r.status === 200 || r.status === 429,
    'response time < 500ms': (r) => r.timings.duration < 500,
  });
  
  // Teste ocasional com token (menos frequente)
  if (Math.random() < 0.1) { // 10% das vezes
    let tokenResponse = http.get('http://localhost:8080/', {
      headers: { 'API_KEY': 'abc123' }
    });
    
    blockedRequests.add(tokenResponse.status === 429);
    
    check(tokenResponse, {
      'token request status is 200 or 429': (r) => r.status === 200 || r.status === 429,
    });
  }
  
  sleep(0.1); // 100ms entre requests
}
