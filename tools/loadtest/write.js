import { sleep, check } from "k6"
import http from "k6/http"
import { Span } from './modules/util.js';

const WRITE_ENDPOINT = __ENV.WRITE_ENDPOINT != null ? __ENV.WRITE_ENDPOINT : "http://0.0.0.0:9411";
const DISTRIBUTOR_ENDPOINT = __ENV.DISTRIBUTOR_ENDPOINT != null ? __ENV.DISTRIBUTOR_ENDPOINT : "http://0.0.0.0:3100";
const INGESTER_ENDPOINT = __ENV.INGESTER_ENDPOINT != null ? __ENV.INGESTER_ENDPOINT : "http://0.0.0.0:3100";

const STEADY_CHECK_WAIT = 5;

export let options = {
  scenarios: {
    writePath: {
      executor: 'ramping-vus',
      exec: 'writePath',
      startVUs: 1,
      stages: [
        { duration: '10m', target: 5 },
        { duration: '7m', target: 10 },
        { duration: '4m', target: 15 },
        { duration: '1m', target: 20 },
        { duration: '5m', target: 0 },
      ],
      gracefulRampDown: '5s'
    },
    steadyStateCheck: {
      executor: 'constant-vus',
      exec: 'steadyStateCheck',
      vus: 1,
      duration: '27m'
    }
  },
  thresholds: {
    // the rate of successful checks should be higher than 99% for the write path
    'checks{type:write}': [{ threshold: 'rate>0.99', abortOnFail: true }],
    // the rate of successful checks should be higher than 90% for the steady checks
    'checks{type:steady}': [{ threshold: 'rate>0.9', abortOnFail: true }],
    http_req_duration: ['p(99)<1500'], // 99% of requests must complete below 1.5s
  }
}

export function writePath() {
  var rootSpan = Span();
  var rootId = rootSpan.traceId;
  var trace = [
    rootSpan,
    Span({ traceId: rootId, parentId: rootId }),
    Span({ traceId: rootId, parentId: rootId })
  ];

  var payload = JSON.stringify(trace);

  console.log(`type=write traceId=${rootId}`);

  let res = http.post(WRITE_ENDPOINT, payload);
  check(res, {
    'write is status 202': (r) => r.status === 202
  }, { type: 'write' });
}


export function steadyStateCheck() {
  // Check Distributors health
  res = http.get(`${DISTRIBUTOR_ENDPOINT}/ready`);
  check(res, {
    'distributor is status 200': (r) => r.status === 200
  }, { type: 'steady', service: 'ingester' });

  // Check Ingesters health
  let res = http.get(`${INGESTER_ENDPOINT}/ready`);
  check(res, {
    'ingester is status 200': (r) => r.status === 200
  }, { type: 'steady', service: 'ingester' });

  sleep(STEADY_CHECK_WAIT)
}