import { randomSeed, check, sleep } from 'k6';
import http from "k6/http";
import { generateId, Span } from './modules/util.js';

const WRITE_ENDPOINT = __ENV.WRITE_ENDPOINT || "http://0.0.0.0:9411";
const DISTRIBUTOR_ENDPOINT = __ENV.DISTRIBUTOR_ENDPOINT || "http://0.0.0.0:3200";
const INGESTER_ENDPOINT = __ENV.INGESTER_ENDPOINT || "http://0.0.0.0:3200";
const QUERY_ENDPOINT = __ENV.QUERY_ENDPOINT || "http://0.0.0.0:3200";
const QUERIER_ENDPOINT = __ENV.QUERIER_ENDPOINT || "http://0.0.0.0:3200";

const WRITE_WAIT = 1;
const READ_WAIT = 1;
const STEADY_CHECK_WAIT = 5;
const ORG_ID = 0;
const START_TIME = Math.round((new Date()).getTime() / 1000);

export const options = {
  scenarios: {
    writePath: {
      exec: 'writePath',
      executor: 'constant-vus',
      vus: 1,
      duration: '30s',
    },
    readPath: {
      exec: 'readPath',
      executor: 'constant-vus',
      vus: 1,
      duration: '30s',
    },
    steadyStateCheck: {
      exec: 'steadyStateCheck',
      executor: 'constant-vus',
      vus: 1,
      duration: '35s',
    },
  },
  thresholds: {
    // the rate of successful checks should be higher than 90% for the read path
    'checks{type:read}': [{ threshold: 'rate>0.9', abortOnFail: true }],
    // the rate of successful checks should be higher than 99% for the write path
    'checks{type:write}': [{ threshold: 'rate>0.99', abortOnFail: true }],
    // the rate of successful checks should be higher than 90% for the steady checks
    'checks{type:steady}': [{ threshold: 'rate>0.9', abortOnFail: true }],
    http_req_duration: ['p(99)<1500'], // 99% of requests must complete below 1.5s
  },
};

export function writePath() {
  randomSeed(START_TIME + __ITER);

  var traceId = generateId(14);
  var trace = [Span({ traceId: traceId })];
  var payload = JSON.stringify(trace);

  let res = http.post(WRITE_ENDPOINT, payload);
  check(res, {
    'write status is 202': (r) => r.status === 202,
  }, { type: 'write' });

  sleep(WRITE_WAIT);
}

export function readPath() {
  sleep(READ_WAIT);

  randomSeed(START_TIME + __ITER);

  var traceId = generateId(14);
  let params = {
    headers: { 'X-Scope-OrgIDr': ORG_ID },
  }

  console.log(`type=read traceId=${traceId}`);

  let res = http.get(`${QUERY_ENDPOINT}/api/traces/${traceId}`, params);
  check(res, {
    'read status is 200': (r) => r.status === 200,
  }, { type: 'read' });
}

export function steadyStateCheck() {
  // Check Distributors health
  let res = http.get(`${DISTRIBUTOR_ENDPOINT}/ready`);
  check(res, {
    'distributor status is 200': (r) => r.status === 200,
  }, { type: 'steady', service: 'distributor' });

  // Check Ingesters health
  res = http.get(`${INGESTER_ENDPOINT}/ready`);
  check(res, {
    'ingester status is 200': (r) => r.status === 200,
  }, { type: 'steady', service: 'ingester' });

  // Check Tempo-Query health
  res = http.get(`${QUERY_ENDPOINT}/ready`)
  check(res, {
    'tempo-query status is 200': (r) => r.status === 200,
  }, { type: 'steady', service: 'query' });

  // Check Querier health
  res = http.get(`${QUERIER_ENDPOINT}/ready`);
  check(res, {
    'querier status is 200': (r) => r.status === 200,
  }, { type: 'steady', service: 'querier' });

  sleep(STEADY_CHECK_WAIT)
}