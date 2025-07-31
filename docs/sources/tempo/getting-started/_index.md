---
title: Get started
menuTitle: Get started
description: Learn about Tempo architecture, concepts, and first steps.
weight: 200
aliases:
- /docs/tempo/getting-started
refs:
  examples:
    - pattern: /docs/tempo/
      destination: https://grafana.com/docs/tempo/<TEMPO_VERSION>/getting-started/example-demo-app/
    - pattern: /docs/enterprise-traces/
      destination: https://grafana.com/docs/enterprise-traces/<ENTERPRISE_TRACES_VERSION>/setup/
  setup:
    - pattern: /docs/tempo/
      destination: https://grafana.com/docs/tempo/<TEMPO_VERSION>/setup/
    - pattern: /docs/enterprise-traces/
      destination: https://grafana.com/docs/enterprise-traces/<ENTERPRISE_TRACES_VERSION>/setup/
  deploy:
    - pattern: /docs/tempo/
      destination: https://grafana.com/docs/tempo/<TEMPO_VERSION>/setup/deployment/
    - pattern: /docs/enterprise-traces/
      destination: https://grafana.com/docs/enterprise-traces/<ENTERPRISE_TRACES_VERSION>/setup/hardware-requirements/
  configure-alloy:
    - pattern: /docs/tempo/
      destination: https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/grafana-alloy/
    - pattern: /docs/enterprise-traces/
      destination: https://grafana.com/docs/enterprise-traces/<ENTERPRISE_TRACES_VERSION>/setup/set-up-get-tenants/
---
<!-- Get started pages are mounted in Grafana Drilldown and in GET. Refer to params.yaml in the website repo. -->

# Get started

Grafana Tempo is an open source, easy-to-use, and high-scale distributed tracing backend. Tempo lets you search for traces, generate metrics from spans, and link your tracing data with logs and metrics.
Grafana Tempo also powers Grafana Cloud Traces and Grafana Enterprise Traces.

Distributed tracing visualizes the lifecycle of a request as it passes through a set of applications.
For more information about traces, refer to [Introduction to traces](https://grafana.com/docs/tempo/<TEMPO_VERSION>/introduction/).

Getting started with Tempo is follows these basic steps.

First, check out the [examples](ref:examples) for ideas on how to get started.

Next, review the [Setup documentation](ref:setup) for step-by-step instructions.

Tempo offers different deployment options, depending on your needs. Refer to the [plan your deployment](ref:deploy) section for more information.

{{< admonition type="note" >}}
Grafana Alloy is already set up to use Tempo.
Refer to [Grafana Alloy configuration for tracing](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/grafana-alloy/).
{{< /admonition >}}

<!-- how to get started with distributed tracing -->
{{< youtube id="zDrA7Ly3ovU" >}}


