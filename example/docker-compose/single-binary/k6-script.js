import { sleep } from 'k6';
import tracing from 'k6/x/tracing';
import { randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

export const options = {
    vus: 1,
    duration: '20m',
};

const endpoint = __ENV.ENDPOINT || 'otel-collector:4317';
const orgid = __ENV.TEMPO_X_SCOPE_ORGID || 'k6-test';
const client = new tracing.Client({
    endpoint,
    exporter: tracing.EXPORTER_OTLP,
    tls: { insecure: true },
    headers: { 'X-Scope-Orgid': orgid },
});

const traceDefaults = {
    attributeSemantics: tracing.SEMANTICS_HTTP,
    attributes: { 'one': 'three' },
    randomAttributes: { count: 2, cardinality: 5 },
    randomEvents: { count: 0.1, exceptionCount: 0.2, randomAttributes: { count: 6, cardinality: 20 } },
    resource: { randomAttributes: { count: 3 } },
};

// Returns `count` identical DB client spans each parented to parentIdx.
// `name` must be unique per group — different names create separate pruning groups.
// No randomAttributes — value variation would split spans across groups.
function repeatedDbSpans(name, table, parentIdx, count) {
    const spans = [];
    for (let i = 0; i < count; i++) {
        spans.push({
            service: 'postgres',
            name: name,
            parentIdx: parentIdx,
            attributeSemantics: tracing.SEMANTICS_DB,
            attributes: {
                'db.system': 'postgresql',
                'db.name': 'shop',
                'db.operation': 'SELECT',
                'db.sql.table': table,
            },
        });
    }
    return spans;
}

// Returns `count` identical HTTP client spans each parented to parentIdx.
// `name` must be unique per group — different names create separate pruning groups.
function repeatedHttpSpans(name, route, service, parentIdx, count) {
    const spans = [];
    for (let i = 0; i < count; i++) {
        spans.push({
            service: service,
            name: name,
            parentIdx: parentIdx,
            attributeSemantics: tracing.SEMANTICS_HTTP,
            attributes: {
                'http.method': 'GET',
                'http.route': route,
                'http.response.status_code': 200,
            },
        });
    }
    return spans;
}

const traceTemplates = [
    // ── Existing small traces (unchanged) ────────────────────────────────────
    {
        defaults: traceDefaults,
        spans: [
            { service: 'shop-backend', name: 'list-articles', duration: { min: 200, max: 900 }, resource: { attributes: { 'namespace': 'shop' } } },
            { service: 'shop-backend', name: 'authenticate', duration: { min: 50, max: 100 }, resource: { randomAttributes: { count: 4 } } },
            { service: 'auth-service', name: 'authenticate', resource: { randomAttributes: { count: 2 }, attributes: { 'namespace': 'auth' } } },
            { service: 'shop-backend', name: 'fetch-articles', parentIdx: 0 },
            { service: 'article-service', name: 'list-articles', links: [{ attributes: { 'link-type': 'parent-child' }, randomAttributes: { count: 2, cardinality: 5 } }], resource: { attributes: { 'namespace': 'shop' } } },
            { service: 'article-service', name: 'select-articles', attributeSemantics: tracing.SEMANTICS_DB },
            { service: 'postgres', name: 'query-articles', attributeSemantics: tracing.SEMANTICS_DB, randomAttributes: { count: 5 }, resource: { attributes: { 'namespace': 'db' } } },
        ],
    },
    {
        defaults: {
            attributes: { 'numbers': ['one', 'two', 'three'] },
            attributeSemantics: tracing.SEMANTICS_HTTP,
            randomEvents: { count: 2, randomAttributes: { count: 3, cardinality: 10 } },
        },
        spans: [
            { service: 'shop-backend', name: 'article-to-cart', duration: { min: 400, max: 1200 } },
            { service: 'shop-backend', name: 'authenticate', duration: { min: 70, max: 200 } },
            { service: 'auth-service', name: 'authenticate' },
            { service: 'shop-backend', name: 'get-article', parentIdx: 0 },
            { service: 'article-service', name: 'get-article' },
            { service: 'article-service', name: 'select-articles', attributeSemantics: tracing.SEMANTICS_DB },
            { service: 'postgres', name: 'query-articles', attributeSemantics: tracing.SEMANTICS_DB, randomAttributes: { count: 2 } },
            { service: 'shop-backend', name: 'place-articles', parentIdx: 0 },
            { service: 'cart-service', name: 'place-articles', attributes: { 'article.count': 1, 'http.response.status_code': 201 } },
            { service: 'cart-service', name: 'persist-cart' },
        ],
    },
    {
        defaults: traceDefaults,
        spans: [
            { service: 'shop-backend', attributes: { 'http.response.status_code': 403 } },
            { service: 'shop-backend', name: 'authenticate', attributes: { 'http.request.header.accept': ['application/json'] } },
            { service: 'auth-service', name: 'authenticate', attributes: { 'http.status_code': 403 }, randomEvents: { count: 0.5, exceptionCount: 2, randomAttributes: { count: 5, cardinality: 5 } } },
        ],
    },
    {
        defaults: traceDefaults,
        spans: [
            { service: 'shop-backend' },
            { service: 'shop-backend', name: 'authenticate', attributes: { 'http.request.header.accept': ['application/json'] } },
            { service: 'auth-service', name: 'authenticate' },
            { service: 'cart-service', name: 'checkout', randomEvents: { count: 0.5, exceptionCount: 2, exceptionOnError: true, randomAttributes: { count: 5, cardinality: 5 } } },
            { service: 'billing-service', name: 'payment', randomLinks: { count: 0.5, randomAttributes: { count: 3, cardinality: 10 } }, randomEvents: { exceptionOnError: true, randomAttributes: { count: 4 } } },
        ],
    },

    // ── Large traces for span pruning ─────────────────────────────────────────
    //
    // Each template has 4 pruneable groups, each with 30 identical leaf spans under
    // a dedicated parent. Different span names per group → 4 separate pruning groups
    // → 4 summary spans after pruning (125 spans total → 9 after pruning).
    //
    // Fetch a trace with ?span_pruning=true to see the effect.
    //
    // Span-index layout (shared by both templates):
    //   0          root
    //   1          parent-A  (→ 0)   2–31   30x leaves-A  (→ 1)
    //   32         parent-B  (→ 0)   33–62  30x leaves-B  (→ 32)
    //   63         parent-C  (→ 0)   64–93  30x leaves-C  (→ 63)
    //   94         parent-D  (→ 0)   95–124 30x leaves-D  (→ 94)

    // Template 4: checkout flow — four N+1 DB patterns in one request.
    // A single checkout triggers repeated queries across four tables.
    // After pruning: 4 summary DB spans instead of 120 individual ones.
    {
        defaults: { attributeSemantics: tracing.SEMANTICS_HTTP },
        spans: [
            // index 0: root
            { service: 'shop-backend', name: 'POST /checkout', duration: { min: 2000, max: 5000 } },

            // group A: cart items  (indexes 1–31)
            { service: 'shop-backend', name: 'load-cart-items', parentIdx: 0, duration: { min: 500, max: 1200 } },
            ...repeatedDbSpans('db.select-cart', 'cart_items', 1, 30),

            // group B: inventory checks  (indexes 32–62)
            { service: 'shop-backend', name: 'check-inventory', parentIdx: 0, duration: { min: 400, max: 1000 } },
            ...repeatedDbSpans('db.select-inventory', 'inventory', 32, 30),

            // group C: price lookups  (indexes 63–93)
            { service: 'shop-backend', name: 'fetch-prices', parentIdx: 0, duration: { min: 300, max: 800 } },
            ...repeatedDbSpans('db.select-price', 'prices', 63, 30),

            // group D: order confirmations  (indexes 94–124)
            { service: 'shop-backend', name: 'send-confirmations', parentIdx: 0, duration: { min: 600, max: 1500 } },
            ...repeatedHttpSpans('http.post-confirm', '/api/v1/confirm', 'notification-service', 94, 30),
        ],
    },

    // Template 5: batch data pipeline — four fan-out stages in one job.
    // Each stage fans out to 30 identical worker calls.
    // After pruning: 4 summary spans instead of 120 individual ones.
    {
        defaults: { attributeSemantics: tracing.SEMANTICS_HTTP },
        spans: [
            // index 0: root
            { service: 'pipeline-service', name: 'POST /process-batch', duration: { min: 3000, max: 8000 } },

            // group A: record validation  (indexes 1–31)
            { service: 'pipeline-service', name: 'validate-records', parentIdx: 0, duration: { min: 800, max: 2000 } },
            ...repeatedHttpSpans('http.validate', '/api/v1/validate', 'validator-service', 1, 30),

            // group B: record transformation  (indexes 32–62)
            { service: 'pipeline-service', name: 'transform-records', parentIdx: 0, duration: { min: 600, max: 1500 } },
            ...repeatedDbSpans('db.transform', 'raw_records', 32, 30),

            // group C: record enrichment  (indexes 63–93)
            { service: 'pipeline-service', name: 'enrich-records', parentIdx: 0, duration: { min: 700, max: 1800 } },
            ...repeatedHttpSpans('http.enrich', '/api/v1/enrich', 'enrichment-service', 63, 30),

            // group D: result persistence  (indexes 94–124)
            { service: 'pipeline-service', name: 'store-results', parentIdx: 0, duration: { min: 500, max: 1200 } },
            ...repeatedDbSpans('db.insert-result', 'processed_records', 94, 30),
        ],
    },
];

export default function () {
    const templateIndex = randomIntBetween(0, traceTemplates.length - 1);
    const gen = new tracing.TemplatedGenerator(traceTemplates[templateIndex]);
    client.push(gen.traces());
    sleep(randomIntBetween(1, 5));
}

export function teardown() {
    client.shutdown();
}
