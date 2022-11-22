# Ubiquiti EdgeRouter Mixin

The Ubiquiti/UBNT EdgeRouter mixin is a single Grafana dashboard based on a custom SNMP module for the [SNMP prometheus exporter](https://github.com/prometheus/snmp_exporter).

Ubiquiti manufactures and sells wireless data communication and wired products for enterprises and homes under multiple brand names. This mixin targets their [EdgeRouter](https://uisp.com/wired/edgemax) series of Ethernet routers.

The dashboard in this mixin is mostly identical to [this dashboard](https://github.com/WaterByWind/grafana-dashboards/tree/master/UBNT-EdgeRouter) built by [WaterByWind](https://github.com/WaterByWind). It has been adapted to use the SNMP exporter, rather than (old and deprecated versions of) telegraf and influxdb.

## SNMP Exporter Module

In `./snmp_generator` you will find a mostly-copied, but slightly modified (a few additional MIBs are included) set of files from the [SNMP exporter generator](https://github.com/prometheus/snmp_exporter/tree/58b902ede4f6bee7a150566ac7fae05ef0a4b1fb/generator).

If you are familiar with the SNMP exporter, you can use the provided `./snmp_generator/generator.yml` file to merge with the other module(s) which you use.

If you are only using the SNMP exporter to scrape metrics for this mixin, you can simply use the included  `./snmp_generator/snmp.yml` file to [configure the exporter](https://github.com/prometheus/snmp_exporter/tree/58b902ede4f6bee7a150566ac7fae05ef0a4b1fb#configuration).

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