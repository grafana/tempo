---
title: Missing traces in Tempo
weight: 472
---

# Some of my traces are missing in Tempo
This could happen because of a number of reasons and some have been detailed in this blog post -
[Where did all my spans go? A guide to diagnosing dropped spans in Jaeger distributed tracing](https://grafana.com/blog/2020/07/09/where-did-all-my-spans-go-a-guide-to-diagnosing-dropped-spans-in-jaeger-distributed-tracing/).

### Diagnosing the issue 
If the pipeline is not reporting any dropped spans, check whether application spans are being dropped by Tempo. The following metrics help determine this -
- `tempo_receiver_refused_spans`. The value of `tempo_receiver_refused_spans` should be 0.
If the value of `tempo_receiver_refused_spans` is greater than 0, then the possible reason is the application spans are being dropped due to rate limiting.

#### Solution
- The rate limiting may be appropriate and does not need to be fixed. The metric simply explained the cause of the missing spans, and there is nothing more to be done.
- If more ingestion volume is needed, increase the configuration for the rate limiting, by adding this CLI flag to Tempo at startup - https://github.com/grafana/tempo/blob/78f3554ca30bd5a4dec01629b8b7b2b0b2b489be/modules/overrides/limits.go#L42
  