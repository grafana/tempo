# rclone Mixin

rclone mixin is one Grafana dashboard based on the metrics exported by [rclone with the `--rc-enable-metrics` flag](https://rclone.org/flags/).

## Common Use-Cases

rclone is a tremendously flexible tool that can be used and invoked in several different ways. This mixin assumes that it is being used in one of two specific configurations.

### Single remote control (rc) server

This is a mode where rclone is run with the `rcd` [command](https://rclone.org/commands/rclone_rcd/). In this mode the rclone server runs continuously and responds to [remote control](https://rclone.org/rc/) requests through an API.

In this case, you would configure prometheus, or the grafana agent with a static scrape config to fetch the metrics from that one rclone target.

### Kubernetes cron jobs

This is a mode where rclone is run in a kubernetes cron job, executed with one of the [file commands](https://rclone.org/commands/) such as sync, copy, purge, etc.

In this case, you would configure prometheus, or the grafana agent with a `kubernetes_sd_config` which would scrape metrics for each new pod which is created for the cronjob.

These pods are listed in the "Historical Instances" table panel on the dashboard.

This could probably be extended to other service discovery scrape methods to operate similarly.

## Generate config files

You can manually generate dashboards, but first you should install some tools:

```bash
go install github.com/jsonnet-bundler/jsonnet-bundler/cmd/jb@latest
go install github.com/google/go-jsonnet/cmd/jsonnet@latest
# or in brew: brew install go-jsonnet
```

For linting and formatting, you would also need `mixtool` and `jsonnetfmt` installed. If you
have a working Go development environment, it's easiest to run the following:

```bash
go install github.com/monitoring-mixins/mixtool/cmd/mixtool@latest
go install github.com/google/go-jsonnet/cmd/jsonnetfmt@latest
```

The files in `dashboards_out` need to be imported
into your Grafana server.  The exact details will be depending on your environment.

```bash
make
```

For more advanced uses of mixins, see
https://github.com/monitoring-mixins/docs.