import tracing from 'k6/x/tracing';
import { sleep } from 'k6';
import { randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';


const endpoint = __ENV.ENDPOINT || "otel-collector:4317"
const orgid = __ENV.TEMPO_X_SCOPE_ORGID || "k6-test"

const client = new tracing.Client({
    endpoint,
    exporter: tracing.EXPORTER_OTLP,
    tls: {
        insecure: true,
    },
    headers: {
        "X-Scope-Orgid": orgid
    }
});

// --- Helper functions ---

// Return current timestamp in nanoseconds
function nsNow() {
    return Date.now() * 1e6;
}

// Return random duration in nanoseconds between minMs and maxMs
function randDuration(minMs, maxMs) {
    return (Math.random() * (maxMs - minMs) + minMs) * 1e6;
}

// Create a single span
function makeSpan(service, name, parentId, attributes = {}, minMs = 50, maxMs = 300) {
    const start = nsNow();
    const duration = randDuration(minMs, maxMs);
    return {
        service,
        name,
        parent_id: parentId || "",
        start_time_unix_nano: start,
        end_time_unix_nano: start + duration,
        attributes,
    };
}

// Generate a single workflow trace
function generateWorkflowTrace() {
    const workflowName = "ci-pipeline";
    const workflowId = `workflow-run-${Math.floor(Math.random() * 10000)}`;
    const service = "github-actions";

    // Root span
    const workflowSpan = makeSpan(service, workflowName, "", {
        "gha.workflow": workflowName,
        "gha.run_id": workflowId,
        "gha.actor": "octocat",
        "gha.repo": "example/repo",
    }, 2000, 4000);

    // Jobs
    const jobBuild = makeSpan(service, "build", workflowSpan.span_id, {
        "gha.job.name": "build",
        "gha.job.status": "success",
    }, 1000, 2500);

    const jobDeploy = makeSpan(service, "deploy", workflowSpan.span_id, {
        "gha.job.name": "deploy",
        "gha.job.status": "success",
    }, 800, 2000);

    // Build steps
    const stepCheckout = makeSpan(service, "checkout", jobBuild.span_id, {
        "gha.step.name": "Checkout repository",
        "gha.step.status": "success",
    }, 100, 300);

    const stepInstall = makeSpan(service, "install", jobBuild.span_id, {
        "gha.step.name": "Install dependencies",
        "gha.step.status": "success",
    }, 200, 800);

    const stepTest = makeSpan(service, "test", jobBuild.span_id, {
        "gha.step.name": "Run tests",
        "gha.step.status": "success",
    }, 400, 1200);

    // Deploy steps
    const stepPush = makeSpan(service, "push-image", jobDeploy.span_id, {
        "gha.step.name": "Push Docker image",
        "gha.step.status": "success",
    }, 200, 600);

    const stepDeploy = makeSpan(service, "deploy-service", jobDeploy.span_id, {
        "gha.step.name": "Deploy service",
        "gha.step.status": "success",
    }, 400, 1000);

    // Create the trace generator
    const generator = new tracing.TemplatedGenerator({
        spans: [
            workflowSpan,
            jobBuild,
            stepCheckout,
            stepInstall,
            stepTest,
            jobDeploy,
            stepPush,
            stepDeploy,
        ],
    });

    // Push the trace batch
    const traces = generator.traces();
    client.push(traces);
}

// --- k6 default function ---
export default function () {
    // Emit a new workflow trace every few seconds
    generateWorkflowTrace();

    sleep(randomIntBetween(1, 5));
}


export function teardown() {
    client.shutdown();
}