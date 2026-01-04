// Initialize tracing before anything else
require('./tracing');

const express = require('express');
const { trace, SpanStatusCode, context, propagation } = require('@opentelemetry/api');

const app = express();
app.use(express.json());

const PORT = process.env.PORT || 8080;
const APP_NAME = process.env.APP_NAME || 'sample-service';
const TARGET_SERVICE = process.env.TARGET_SERVICE;

const tracer = trace.getTracer(APP_NAME);

// Simulate some work
function simulateWork(minMs = 10, maxMs = 100) {
  const duration = Math.floor(Math.random() * (maxMs - minMs + 1)) + minMs;
  return new Promise(resolve => setTimeout(resolve, duration));
}

// Make downstream call with trace context propagation
async function callDownstream(url, traceId) {
  const headers = {};
  propagation.inject(context.active(), headers);
  
  try {
    const response = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...headers,
      },
      body: JSON.stringify({ traceId, source: APP_NAME }),
    });
    return await response.json();
  } catch (error) {
    console.error(`Failed to call ${url}:`, error.message);
    return { error: error.message };
  }
}

// Health check endpoint
app.get('/health', (req, res) => {
  res.json({ status: 'healthy', service: APP_NAME });
});

// Main endpoint that creates traces
app.post('/process', async (req, res) => {
  const span = tracer.startSpan('process-request');
  
  try {
    span.setAttribute('app.name', APP_NAME);
    span.setAttribute('request.body', JSON.stringify(req.body));

    // Simulate processing
    await tracer.startActiveSpan('validate-input', async (validateSpan) => {
      await simulateWork(5, 20);
      validateSpan.setAttribute('validation.result', 'success');
      validateSpan.end();
    });

    // Simulate database operation
    await tracer.startActiveSpan('database-query', async (dbSpan) => {
      dbSpan.setAttribute('db.system', 'postgresql');
      dbSpan.setAttribute('db.operation', 'SELECT');
      await simulateWork(20, 50);
      dbSpan.end();
    });

    // Call downstream service if configured
    let downstreamResult = null;
    if (TARGET_SERVICE) {
      await tracer.startActiveSpan('call-downstream', async (downstreamSpan) => {
        downstreamSpan.setAttribute('http.url', TARGET_SERVICE);
        downstreamSpan.setAttribute('http.method', 'POST');
        
        const currentSpan = trace.getActiveSpan();
        const traceId = currentSpan?.spanContext().traceId;
        
        downstreamResult = await callDownstream(`${TARGET_SERVICE}/process`, traceId);
        
        downstreamSpan.setAttribute('downstream.response', JSON.stringify(downstreamResult));
        downstreamSpan.end();
      });
    }

    // Simulate more processing
    await tracer.startActiveSpan('finalize', async (finalSpan) => {
      await simulateWork(10, 30);
      finalSpan.end();
    });

    const currentSpan = trace.getActiveSpan();
    const traceId = currentSpan?.spanContext().traceId || span.spanContext().traceId;

    span.setStatus({ code: SpanStatusCode.OK });
    
    res.json({
      success: true,
      service: APP_NAME,
      traceId,
      downstream: downstreamResult,
      message: `Processed by ${APP_NAME}`,
    });
  } catch (error) {
    span.setStatus({ code: SpanStatusCode.ERROR, message: error.message });
    span.recordException(error);
    res.status(500).json({ error: error.message });
  } finally {
    span.end();
  }
});

// Endpoint to generate a complete trace across services
app.get('/generate-trace', async (req, res) => {
  const span = tracer.startSpan('generate-trace');
  
  try {
    span.setAttribute('app.name', APP_NAME);
    
    const traceId = span.spanContext().traceId;
    console.log(`[${APP_NAME}] Generating trace: ${traceId}`);

    // Simulate work
    await simulateWork(50, 150);

    // Call downstream if available
    let result = { service: APP_NAME, traceId };
    if (TARGET_SERVICE) {
      await context.with(trace.setSpan(context.active(), span), async () => {
        const downstream = await callDownstream(`${TARGET_SERVICE}/process`, traceId);
        result.downstream = downstream;
      });
    }

    span.setStatus({ code: SpanStatusCode.OK });
    res.json(result);
  } catch (error) {
    span.setStatus({ code: SpanStatusCode.ERROR, message: error.message });
    res.status(500).json({ error: error.message });
  } finally {
    span.end();
  }
});

app.listen(PORT, () => {
  console.log(`[${APP_NAME}] Server running on port ${PORT}`);
  console.log(`[${APP_NAME}] OTLP endpoint: ${process.env.OTEL_EXPORTER_OTLP_ENDPOINT}`);
  if (TARGET_SERVICE) {
    console.log(`[${APP_NAME}] Downstream service: ${TARGET_SERVICE}`);
  }
});
