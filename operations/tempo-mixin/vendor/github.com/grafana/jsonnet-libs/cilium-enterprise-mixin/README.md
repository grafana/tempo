# Cilium Mixin

The Cilium Mixin is a set of configurable, reusable, and extensible alerts as well as dashboards based on the metrics exported by [Cilium's internal Prometheus exporter](https://docs.cilium.io/en/stable/operations/metrics/#installation). 

This integration includes the following dashboards.

- Cilium / Overview
- Cilium / Operator
- Cilium / Agent Overview
- Cilium / Components / Agent
- Cilium / Components / API
- Cilium / Components / BPF
- Cilium / Components / Conntrack
- Cilium / Components / Datapath
- Cilium / Components / External HA FQDN Proxy
- Cilium / Components / FQDN Proxy
- Cilium / Components / Identities
- Cilium / Components / Kubernetes
- Cilium / Components / L3 Policy
- Cilium / Components / L7 Proxy
- Cilium / Components / Network
- Cilium / Components / Nodes
- Cilium / Components /Policy
- Cilium / Components / Resource Utilization
- Hubble / Overview
- Hubble / Timescape


## Cilium Overview & Cilium Agent Overview
These two dashboards give a general overview of the state of the Cilium deployment, as reported by the Cilium Agent. 

***Cilium Overview***

![image](https://storage.googleapis.com/grafanalabs-integration-assets/cilium-enterprise/screenshots/cilium_overview_1.png)

***Cilium Overview***

![image](https://storage.googleapis.com/grafanalabs-integration-assets/cilium-enterprise/screenshots/cilium_overview_2.png)

***Cilium Agent Overview***

![image](https://storage.googleapis.com/grafanalabs-integration-assets/cilium-enterprise/screenshots/cilium_agent_overview_1.png)

***Cilium Agent Overview***

![image](https://storage.googleapis.com/grafanalabs-integration-assets/cilium-enterprise/screenshots/cilium_agent_overview_2.png)


## Cilium Operator Overview
This dashboard provides information on the state of the Cilium Operator; its resource utilization and IPAM status.

***Cilium Operator Overview***

![image](https://storage.googleapis.com/grafanalabs-integration-assets/cilium-enterprise/screenshots/cilium_operator_overview_1.png)

***Cilium Operator Overview***

![image](https://storage.googleapis.com/grafanalabs-integration-assets/cilium-enterprise/screenshots/cilium_operator_overview_2.png)

## Hubble Overview & Hubble Timescape

The Hubble Overview and Hubble Timescape dashboards provide detailed insights into the state of the Hubble observability platform, including aggregate network information.

***Hubble Overview***

![image](https://storage.googleapis.com/grafanalabs-integration-assets/cilium-enterprise/screenshots/hubble_overview_1.png)

***Hubble Timescape***

![image](https://storage.googleapis.com/grafanalabs-integration-assets/cilium-enterprise/screenshots/hubble/hubble_timescape_1.png)


## Dashboard Links

On the top right of dashboards, you will find dropdowns to quickly switch between the dashboards while keeping the time range and variable selection the same.

## How to use this mixin
The mixin creates recording and alerting rules for Prometheus and suitable 
dashboards for Grafana.

To use them, you need to have `mixtool` and `jsonnetfmt` installed. If you
have a working Go development environment, it's easiest to run the following:
```bash
$ go get github.com/monitoring-mixins/mixtool/cmd/mixtool
$ go get github.com/google/go-jsonnet/cmd/jsonnetfmt
```

You can then build the Prometheus rules files `alerts.yaml` and
`rules.yaml` and a directory `dashboard_out` with the JSON dashboard files
for Grafana:
```bash
$ make build
```

For more advanced uses of mixins, see
https://github.com/monitoring-mixins/docs.