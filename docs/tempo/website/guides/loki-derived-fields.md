---
title: Loki Derived Fields
---

Guide on integrating Tempo with Loki and Derived fields in Grafana.

## Requirements

### Application is instrumented for tracing

To instrument an application for
tracing, use a client library like [OpenTracing](opentracing.io) or [OpenTelemetry](opentelemetry.io).

Example instrumentation with Jaeger Go Client:
```go
    import "github.com/opentracing/opentracing-go"
    jaegercfg "github.com/uber/jaeger-client-go/config"

    func main() {
        cfg, err := jaegercfg.FromEnv()
        if err != nil {
            // handle err
        }

        tracer, closer, err := cfg.NewTracer()
        if err != nil {
            // handle err
        }
    	defer closer.Close()

        opentracing.SetGlobalTracer(tracer)

        span := opentracing.StartSpan("operation_name")
        // do some work
        defer span.Finish()
        ...
    }
```

### Application logs traceIDs

`traceID` can be extracted from the context using a function provided by the client library
To extend the above example:

```go
    import (
        "context"
        "time"

        "github.com/opentracing/opentracing-go"
        jaegercfg "github.com/uber/jaeger-client-go/config"
    )

    // code copied from github.com/weaveworks/common/middleware/http_tracing.go
    // ExtractTraceID extracts the trace id, if any from the context.
    func ExtractTraceID(ctx context.Context) (string, bool) {
        sp := opentracing.SpanFromContext(ctx)
        if sp == nil {
            return "", false
        }
        sctx, ok := sp.Context().(jaeger.SpanContext)
        if !ok {
            return "", false
        }

        return sctx.TraceID().String(), true
    }

    func doWork(ctx context.Context) {
        span, _ := opentracing.StartSpanFromContext(ctx, "main.doWork")
        defer span.Finish()
        traceID, ok := ExtractTraceID(ctx)
        if !ok {
            return
        }

        // sleep to simulate work
        time.Sleep(10 * time.Second)
        fmt.Println("traceID:", traceID)
    }

    func main() {
        cfg, err := jaegercfg.FromEnv()
        if err != nil {
            // handle err
        }

        tracer, closer, err := cfg.NewTracer()
        if err != nil {
            // handle err
        }
    	defer closer.Close()

        opentracing.SetGlobalTracer(tracer)

        span := opentracing.StartSpan("operation_name")
        doWork(opentracing.ContextWithSpan(context.Background(), span))
        defer span.Finish()
        ...
    }
```

The traceIDs logged by this application can be pasted into the Tempo Query UI in Grafana.

### Configure Tempo Datasource in Grafana

Follow the guide to configure Tempo datasource in Grafana.

### Loki datasource is configured in Grafana. (optional)

To configure the Loki datasource in Grafana follow the guide - https://grafana.com/docs/grafana/latest/features/datasources/loki/

### Derived fields are set up. (optional)

Refer to this document for more information on setting up derived fields in Grafana.
https://grafana.com/docs/grafana/latest/features/datasources/loki/#derived-fields

Once derived fields are set up and the required datasources are configured,
we can jump from logs -> traces within Grafana.

#### References

- https://grafana.com/blog/2020/05/22/new-in-grafana-7.0-trace-viewer-and-integrations-with-jaeger-and-zipkin/
- https://grafana.com/blog/2020/03/31/how-to-successfully-correlate-metrics-logs-and-traces-in-grafana/
