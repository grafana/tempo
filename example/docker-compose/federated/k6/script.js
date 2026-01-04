import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Counter, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const requestCount = new Counter('requests');
const requestDuration = new Trend('request_duration');

// Configuration from environment variables
const TARGET_URL = __ENV.TARGET_URL || 'http://sample-app-1:8080/generate-trace';
const RPS = parseInt(__ENV.REQUESTS_PER_SECOND) || 1;

export const options = {
  scenarios: {
    constant_load: {
      executor: 'constant-arrival-rate',
      rate: RPS,
      timeUnit: '1s',
      duration: __ENV.DURATION || '9h',
      preAllocatedVUs: 10,
      maxVUs: 50,
    },
  },
  thresholds: {
    errors: ['rate<0.1'], // Error rate should be less than 10%
    http_req_duration: ['p(95)<2000'], // 95% of requests should be under 2s
  },
};

export default function () {
  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
    timeout: '10s',
  };

  // Make request to the sample app which triggers the trace chain
  const response = http.get(TARGET_URL, params);

  // Track metrics
  requestCount.add(1);
  requestDuration.add(response.timings.duration);

  // Check response
  const success = check(response, {
    'status is 200': (r) => r.status === 200,
    'response time < 5s': (r) => r.timings.duration < 5000,
  });

  errorRate.add(!success);

  if (!success) {
    console.log(`Request failed: status=${response.status}, body=${response.body}`);
  }

  // Small random sleep to add variation
  sleep(Math.random() * 0.5);
  // JSON decoding to extract trace ID
  const responseBody = JSON.parse(response.body);
  console.log(`Request to ${TARGET_URL} completed with status ${response.status} with trace ID: ${responseBody.traceId}`);
}

export function handleSummary(data) {
  console.log('Load test completed');
  console.log(`Total requests: ${data.metrics.requests.values.count}`);
  console.log(`Error rate: ${(data.metrics.errors.values.rate * 100).toFixed(2)}%`);
  console.log(`Avg duration: ${data.metrics.http_req_duration.values.avg.toFixed(2)}ms`);
  console.log(`P95 duration: ${data.metrics.http_req_duration.values['p(95)'].toFixed(2)}ms`);
  
  return {};
}
