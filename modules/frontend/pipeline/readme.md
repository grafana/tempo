# pipeline

This package contains elements for building discrete/http and streaming/grpc request pipelines. It is designed to be used as building blocks 
that are constructed in frontend.go. Something like the following:

```
	searchPipeline := pipeline.Build(
		asyncPipeline(cfg, newAsyncSearchSharder(reader, o, cfg.Search.Sharder, logger), logger),
		[]pipeline.Middleware{cacheWare, statusCodeWare, retryWare},
		next)

  http := newSearchHTTPHandler(cfg, searchPipeline, logger)
  grpc := newSearchStreamingGRPCHandler(cfg, searchPipeline, apiPrefix, logger)
```

## standard pipeline

The standard pipeline looks like the following:

collector -> pipeline -> http.Roundtripper

The final http.Roundtripper is the old cortex.frontend which then farms requests out to the queriers. This
can also easily be replaced with a mock to write integration tests.

### collector

There are two collectors that are designed to be used directly with a desired endpoint: GRPC and HTTP. Both types
of collectors require a combiner from the frontend/combiner package. These type aware combiners handle the specifics
of unmarshalling and combining the type while the collectors handles the accumulation of responses and the ergonomics 
of streaming/discrete responses.

The combiners are responsible for aggregating and combining the results as well as translating the results into
either an HTTP or GRPC response.

#### GRPC
The GRPC collector is designed to return a series of streaming diffs which can then be returned to a client from a GRPC server.

```
func NewGRPCCollector[T combiner.TResponse](next AsyncRoundTripper[*http.Response], combiner combiner.GRPCCombiner[T], send func(T) error) *GRPCCollector[T]
```

It takes a callback that happens to nicely match up with the server.Send function which allows a user to write
something like:

```
  collector := pipeline.NewGRPCCollector[*tempopb.SearchResponse](next, c, srv.Send)
  err := collector.RoundTrip(req)
```

#### HTTP
The HTTP collector is designed to return a single, discrete response like a traditional http endpoint:

```
func NewHTTPCollector(next AsyncRoundTripper[*http.Response], combiner combiner.Combiner) http.RoundTripper
```

Since it returns a roundtripper it's quite easy to use with existing http code. Something like:

```
		// build and use roundtripper
		combiner := combiner.NewTypedSearch(int(limit))
		rt := pipeline.NewHTTPCollector(next, combiner)

		return rt.RoundTrip(req)
```

### pipeline
The pipeline referenced above can be built in many different ways, but a standard pipeline will look like:

| ------------ async ------------- |   | ------------- sync -------------- |
async multitenant -> async sharding -> cache -> status code rewrite -> retry

The first two elements are part of the asynchronous pipeline. These elements often create many jobs for one 
request and asynchronously pipe responses back to the collector level for recombination.

The last 3 elements are part of the synchronous pipeline. This part of the pipeline always returns one
response for each request.

#### async multitenant

Creates one job for every tenant in the tenant header and passes them forward.

#### async sharding

Most pipelines include a "job sharding" step that breaks the request into many smaller requests to be farmed 
out to queriers. "async sharding" is this step. It is not required for all endpoints.

#### cache

If the context includes a cache key the cache layer will automatically attempt to retrieve it from cache
and shortcircuit the reponse back up the pipeline.

#### status code rewrite

This steps maps status codes from the querier into status codes appropriate for the pipeline. In particular
it will take 400s and map them to 500.

#### retry

Retry is a lift and shift of the previous retry middleware. If a 

## error handling

- All errors in the pipeline must be propagated back faithfully
- A pipeline item should turn an error into a context aware response if possible. i.e.
  ```
  if err := parse(req.Url); err != nil {
    return http.Response(400, err), nil
  }
  ```
- The combiner aggregates responses and turns them into a GRPC response/error or an HTTP response/error

## context cancellation

Currently context is only cancelled in the collectors. The collector is watching for any of the following events to know 
that all subjobs can be cancelled:

- Error return
  i.e. err return param on pipeline items
- Bad Request return
  i.e. The combiner receives a 500 or 400
- Response limit reached
- All jobs exhausted
- Context canceled (due to client disconnect)
