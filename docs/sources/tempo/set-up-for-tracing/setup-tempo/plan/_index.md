---
title: Plan your Tempo deployment
menuTitle: Plan your deployment
description: Plan your Grafana Tempo deployment
weight: 200
---

# Plan your Tempo deployment

Before you deploy Tempo, you should consider how you plan to use traces and Tempo as well as any special requirements for Tempo 3.0. 

## Considerations for Tempo 3.0

Before planning your Tempo deployment, be aware of the following infrastructure requirements:

- Object storage: Tempo stores trace data in object storage such as S3, GCS, or Azure Storage. Object storage is required for microservices deployments and recommended for production. Monolithic deployments can use local filesystem storage for development and test environments.
- Kafka-compatible system: Microservices mode requires a Kafka-compatible system, such as Apache Kafka, Redpanda, or WarpStream, as the durable queue that decouples the write and read paths. Monolithic mode doesn't use Kafka.

Include these infrastructure components in your deployment planning and resource estimation.

## Develop a deployment plan

To plan your Grafana Tempo deployment, you should:

1. Identify your use case
   Decide what you need from Tempo. For example, consider if you want a basic installation, horizontal scalability, multi-tenancy, or advanced data storage. Your use case determines the deployment architecture and resource requirements.

2. Estimate tracing data volume
   Assess the amount of tracing data your environment will generate. This helps you size your cluster and storage.
   Refer to the [Size your cluster](./size/) section for guidance on calculating storage and performance needs.

3. Choose a deployment mode.
   Tempo supports monolithic and microservices deployment modes.

   - Use monolithic mode for getting started, development, or smaller workloads. No Kafka is required.
   - Use microservices mode for production, high-volume, or highly available deployments. Requires a Kafka-compatible system.

   Review the [Deployment modes](./deployment-modes/) documentation to compare options and select the best fit for your use case.

By following these recommendations, you can plan a Tempo deployment that matches your operational requirements and scales with your tracing data.

## Next steps

Once you've planned your deployment, you can [Deploy your Tempo instance](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/deploy/) using the deployment mode you selected.
