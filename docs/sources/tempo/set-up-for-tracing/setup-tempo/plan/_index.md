---
title: Plan your Tempo deployment
menuTitle: Plan your deployment
description: Plan your Grafana Tempo deployment
aliases:
weight: 200
---

# Plan your Tempo deployment

To plan your Grafana Tempo deployment, you should:

1. Identify your use case
   Decide what you need from Tempo. For example, consider if you want a basic installation, horizontal scalability, multi-tenancy, or advanced data storage. Your use case determines the deployment architecture and resource requirements.

2. Estimate tracing data volume
   Assess the amount of tracing data your environment will generate. This helps you size your cluster and storage.
   Refer to the [Size your cluster](./size/) section for guidance on calculating storage and performance needs.

3. Choose a deployment mode.
   Tempo supports monolithic and microservices deployment modes.

   - Use the monolithic mode for simple, single-tenant environments or smaller workloads.
   - Use the microservices mode for large-scale, multi-tenant, or highly available deployments.
     Review the [Deployment modes documentation](./deployment-modes/) to compare options and select the best fit for your use case.

By following these recommendations, you can plan a Tempo deployment that matches your operational requirements and scales with your tracing data.

## Next steps

Once you've planned your deployment, you can [Deploy your Tempo instance](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/deploy/) using the deployment mode you selected.
