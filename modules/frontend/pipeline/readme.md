# pipeline

## basic structure
collector -> pipeline -> http.Roundtripper (cortex frontend vs. testing)

### collector
grpc or http
takes a combiner

### pipeline
async multitenant -> async sharding -> cache -> retry

pipeline items are responsible for faithfully propagating requests and errors up but should not attempt to cancel, log or terminate them

## error handling
- all middleware adds a label
- combiner takes errors/status code and interprets them into http status/grpc errors
- collectors consume the results of a combiner and send them out
  - collectors assumes the semantics of a gRPC or HTTP endpoint and provides errors/responses that can be sent directly out
- errors must be sent to the calling pipeline item and processing stop

## cancellation
- what cancels/terminates the pipeline?
  - should be collectors. they have all the info w/ the help of the combiner

## pipeline communication
sharder <-> combiner

## logging
- nothing logs? responsibility of caller?
