# Tempo k6 tests

These scripts are used for smoke/load/stress/soak testing Tempo deployments with [k6](https://github.com/loadimpact/k6).

## Prerequisites

- k6 >=0.29.0 (Installation instructions [here](https://github.com/loadimpact/k6#install))

- Tempo deployment (monolith or microservices)


## Tests

### Smoke tests

Smoke Test's role is to verify that Tempo can handle minimal load, without any problems.

**`smoke_test.js`**

On this test we run three scenarios in parallel:
- writePath: generates a trace and pushes it to Tempo (using the Zipkin HTTP receiver).
- readPath: generates a traceId and queries it from Tempo.
- steadyCheck: health checks the services that are part of the read and write path.


### Stress tests

Stress Tests are concerned with assessing the limits of Tempo and its stability under extreme conditions.

**`stress_test_write_path.js`**

On this test we run two scenarios in parallel:
- writePath: generates a trace with multiple spans and pushes it to Tempo (using the Zipkin HTTP receiver).
  - This scenario has stages, where the VUs are increased during the test run. 
- steadyCheck: health checks the services that are part of the read path.

## Run

If you're running Tempo locally in monolith mode (for e.g while developing), the scripts are ready to use.

If you have the k6 binary installed:
```bash
$ k6 run script.js
```

If you're using the Docker image:
```bash
$ docker run -v $PWD:/src -i --network host loadimpact/k6 run --quiet /src/script.js
```

**But... I want to configure the endpoints!**

If you are running Tempo in microservices mode, or another place that it's not localhost you've to change the default endpoints of the test.

You can use environment variables or the `-e` CLI flag for that:
```bash
$ k6 run -e INGESTER_ENDPOINT=tempo.example.com script.js
```

## Resources

- Zipkin HTTP API [docs](https://zipkin.io/zipkin-api/#/default/get_trace__traceId_).
- k6 [docs](https://k6.io/docs/).