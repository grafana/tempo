## Docker-compose

So you found your way to the docker-compose examples?  This is a great place to get started with Tempo, learn
some basic configuration and learn about various trace discovery flows.

If you are interested in more complex configuraiton we would recommend the [tanka/jsonnet examples](../tk/readme.md).

### Build Images

This step is not necessary, but it can be nice for local testing.  Rebuilding these images will cause 
docker-compose to use whatever local code you have in the examples.

Run the following from the project root folder to build `tempo:latest` and `tempo-query:latest` images
that will be used in the test environment:

```console
make docker-images
```

## Local Storage

1. First start up the local stack.
```
docker-compose -f docker-compose.local.yaml up
```

2. If you're interested you can see the wal/blocks as they are being created.
```
ls /tmp/tempo
```

3. The synthetic-load-generator is now printing out trace ids it's flushing into Tempo.  Logs are in the form

`Emitted traceId <traceid> for service frontend route /cart`

Copy one of these trace ids.

4. Navigate to [Grafana](http://localhost:3000/explore?orgId=1&left=%5B%22now-1h%22,%22now%22,%22Tempo%22,%7B%7D%5D) and paste the trace id to request it from Tempo. 
Also notice that you can query Tempo metrics from the Prometheus data source setup in Grafana.

## S3

1. First start up the s3 stack.
```
docker-compose up
```

2. If you're interested you can see the wal/blocks as they are being created.  Navigate to minio at 
http://localhost:9000 and use the username/password of `tempo`/`supersecret`.


3. The synthetic-load-generator is now printing out trace ids it's flushing into Tempo.  Logs are in the form
```
Emitted traceId <traceid> for service frontend route /cart
```
Copy one of these trace ids.

4. Navigate to [Grafana](http://localhost:3000/explore?orgId=1&left=%5B%22now-1h%22,%22now%22,%22Tempo%22,%7B%7D%5D) and paste the trace id to request it from Tempo. 
Also notice that you can query Tempo metrics from the Prometheus data source setup in Grafana.

## Loki

1. First we have to install the Loki docker driver.  This allows applications in our docker-compose to ship their logs
to Loki.

```
docker plugin install grafana/loki-docker-driver:latest --alias loki --grant-all-permissions
```

2. Next start up the Loki stack.
```
docker-compose -f docker-compose.loki.yaml up
```

3. Navigate to [Grafana](http://localhost:3000/explore?orgId=1&left=%5B%22now-1h%22,%22now%22,%22Loki%22,%7B%7D%5D) and query Loki a few times to generate some traces.
Something like the below works, but feel free to explore other options!
```
{container_name="dockercompose_loki_1"}
```

4. Now let's execute a query specifically looking for some trace ids.  In an operational scenario you would normally be using Loki to search for things like
query path, or status code, but we're just going to do this for the example:

```
{container_name="dockercompose_loki_1"} |= "traceID"
```

5. Drop down the log line and click the Tempo link to jump directly from logs to traces!

![Tempo link](./tempo-link.png)