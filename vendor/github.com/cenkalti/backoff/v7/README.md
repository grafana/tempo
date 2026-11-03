# Exponential Backoff [![GoDoc][godoc image]][godoc]

This is a Go port of the exponential backoff algorithm from [Google's HTTP Client Library for Java][google-http-java-client].

[Exponential backoff][exponential backoff wiki]
is an algorithm that uses feedback to multiplicatively decrease the rate of some process,
in order to gradually find an acceptable rate.
The retries exponentially increase and stop increasing when a certain threshold is met.

## Install

```
go get github.com/cenkalti/backoff/v7
```

Note the `/v7` at the end of the import path.

## Usage

For most cases, wrap the operation you want to retry in `Retry`:

```go
result, err := backoff.Retry(ctx, func() (string, error) {
	resp, err := http.Get("https://www.example.com")
	if err != nil {
		return "", err // transient: Retry will try again
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode >= 500:
		return "", fmt.Errorf("server error: %s", resp.Status) // retried
	case resp.StatusCode >= 400:
		// client errors won't fix themselves, so stop retrying.
		return "", backoff.Permanent(fmt.Errorf("client error: %s", resp.Status))
	}
	return "ok", nil
}, backoff.WithMaxTries(5))
```

`Retry` runs the operation at least once and keeps retrying with exponential
backoff until it succeeds, returns a `Permanent` error, or a limit is reached.
See [example_test.go][example] for a fuller example, and the [package docs][godoc]
for the available options (`WithBackOff`, `WithMaxTries`, `WithMaxElapsedTime`,
`WithNotify`).

If `Retry` does not fit your needs, copy it from [retry.go][retry-src] and adapt it.

### Handling errors

On failure, `Retry` always returns a `*RetryError`. It carries the last operation error (`LastErr`) and the reason retrying stopped (`Cause`). Inspect it with `errors.Is`, or reach the struct with `AsRetryError`:

```go
result, err := backoff.Retry(ctx, operation)
switch {
case errors.Is(err, backoff.ErrPermanent):
	// the operation returned a Permanent error
case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
	// the caller's context was cancelled or its deadline expired
case errors.Is(err, backoff.ErrMaxElapsedTime):
	// the WithMaxElapsedTime budget was exhausted
case errors.Is(err, backoff.ErrExhausted):
	// WithMaxTries was reached or the backoff policy returned Stop
}

// The last operation error is always available, whatever the cause:
if re := backoff.AsRetryError(err); re != nil {
	log.Printf("gave up after last error: %v", re.LastErr)
}
```

Mark an error non-retriable with `backoff.Permanent(err)`; `Retry` stops immediately and returns a `*RetryError` whose `Cause` is `ErrPermanent` and whose `LastErr` is `err`.

### Bounding total time

Two independent limits cap how long `Retry` runs, and they behave differently:

- A **context deadline** (`context.WithTimeout`) is reactive: it interrupts the wait between attempts and — if your operation observes the context — can abort an in-flight attempt. `Retry` reports it as `context.DeadlineExceeded`.
- **`WithMaxElapsedTime`** bounds only retry scheduling: it is checked between attempts, never interrupts a running operation, and is reported as `ErrMaxElapsedTime`.

`WithMaxElapsedTime` defaults to 15 minutes, so **both limits are active unless you override it** — pass `backoff.WithMaxElapsedTime(0)` to rely solely on the context.

## Contributing

* I would like to keep this library as small as possible.
* Please don't send a PR without opening an issue and discussing it first.
* If proposed change is not a common use case, I will probably not accept it.

[godoc]: https://pkg.go.dev/github.com/cenkalti/backoff/v7
[godoc image]: https://pkg.go.dev/badge/github.com/cenkalti/backoff/v7.svg

[google-http-java-client]: https://github.com/google/google-http-java-client/blob/da1aa993e90285ec18579f1553339b00e19b3ab5/google-http-client/src/main/java/com/google/api/client/util/ExponentialBackOff.java
[exponential backoff wiki]: http://en.wikipedia.org/wiki/Exponential_backoff

[retry-src]: https://github.com/cenkalti/backoff/blob/v7/retry.go
[example]: https://github.com/cenkalti/backoff/blob/v7/example_test.go
