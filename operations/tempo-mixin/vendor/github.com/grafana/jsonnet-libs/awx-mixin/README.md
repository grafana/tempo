# AWX Mixin

AWX mixin is a single Grafana dashboard based on the metrics exported from [AWX by default](https://docs.ansible.com/ansible-tower/latest/html/administration/metrics.html) at the path `api/v2/metrics`.

AWX is the open source upstream source for Ansible Tower. This mixin is compatble with either AWX or Ansible Tower.

AWX *may* be clustered and have several instances which can execute jobs, but the metrics endpoint aggregates all of those details, meaning you only need to target one node of the cluster to scrape all available metrics.

The dashboard focuses on showing overall cluster details, and the rate of job launches and completions by type. In the overview, a list of all discovered clusters will be presented, and can be clicked on to see metrics specific to that cluster.

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