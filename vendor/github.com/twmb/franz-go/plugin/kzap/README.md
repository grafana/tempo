kzap
===

kzap is a plug-in package to hook Uber's [zap](https://github.com/uber-go/zap)
into a [`kgo.Logger`](https://pkg.go.dev/github.com/twmb/franz-go/pkg/kgo#Logger)

To use,

```go
cl, err := kgo.NewClient(
        kgo.WithLogger(kzap.New(zapLogger)),
        // ...other opts
)
```

By default, the logger chooses the highest level possible that is enabled on
the zap logger, and then sticks with that level forever. A variable level
can be chosen by specifying the `LevelFn` option. See the documentation on
[`Level`](https://pkg.go.dev/github.com/twmb/franz-go/plugin/kzap#Level) or [`LevelFn`](https://pkg.go.dev/github.com/twmb/franz-go/plugin/kzap#LevelFn) for more info.
