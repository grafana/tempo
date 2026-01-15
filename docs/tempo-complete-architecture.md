# Tempo Complete Architecture - All Components & Paths

This diagram shows all Tempo Rhythm components with typical replica counts and all possible data paths.

## Complete System Diagram

```mermaid
flowchart TB
    subgraph Clients["Client Applications"]
        APP[("Applications<br/>OTLP/Jaeger/Zipkin")]
        GRAFANA[("Grafana<br/>Queries")]
    end

    subgraph Enterprise["backend-enterprise"]
        GW["Cloud Backend Gateway<br/>━━━━━━━━━━━━━━━━━━━━<br/>• Auth/AuthZ<br/>• Multi-tenancy<br/>• LBAC<br/>• Licensing<br/>━━━━━━━━━━━━━━━━━━━━<br/>Replicas: 3+"]
    end

    subgraph Adaptive["adaptive-traces"]
        SGW["Sampler Gateway<br/>━━━━━━━━━━━━━━━━━━━━<br/>• Head sampling (if policies)<br/>• Pass-through proxy (if no policies)<br/>• Rate limiting<br/>━━━━━━━━━━━━━━━━━━━━<br/>Replicas: 3+"]

        KAFKA_AT[("Kafka<br/>(Adaptive Traces)<br/>━━━━━━━━━━━━━━<br/>Partitions: N")]

        SAMPLER["Sampler<br/>━━━━━━━━━━━━━━━━━━━━<br/>• Tail sampling<br/>• Anomaly detection<br/>• Diversity sampling<br/>• Policy-based decisions<br/>━━━━━━━━━━━━━━━━━━━━<br/>Replicas: 3+"]

        POLICY["Policy Store<br/>━━━━━━━━━━━━━━━━━━━━<br/>Per-tenant policies"]

        DECIDX["Decision Index<br/>━━━━━━━━━━━━━━━━━━━━<br/>Keep/drop decisions"]
    end

    subgraph Tempo["Tempo (Rhythm Architecture)"]
        subgraph Ingestion["Ingestion Layer"]
            DIST["Distributor<br/>━━━━━━━━━━━━━━━━━━━━<br/>• Validation<br/>• Rate limiting<br/>• Routing to Kafka<br/>━━━━━━━━━━━━━━━━━━━━<br/>Replicas: 5"]
        end

        KAFKA_TEMPO[("Kafka<br/>(Tempo Ingest)<br/>━━━━━━━━━━━━━━<br/>Partitions: N")]

        subgraph WriteLayer["Write Layer"]
            BB1["Block Builder 1"]
            BB2["Block Builder 2"]
            BB3["Block Builder 3"]
            BBN["Block Builder N<br/>━━━━━━━━━━━━━━━━━━━━<br/>• Consumes Kafka<br/>• Creates blocks<br/>━━━━━━━━━━━━━━━━━━━━<br/>Replicas: N<br/>(matches partitions)"]
        end

        subgraph ReadLayer["Read Layer"]
            subgraph LiveStoreZones["Live Store (2 zones)"]
                LS_A["Live Store Zone-A<br/>━━━━━━━━━━━━━━━━━━━━<br/>Replicas: N"]
                LS_B["Live Store Zone-B<br/>━━━━━━━━━━━━━━━━━━━━<br/>Replicas: N"]
            end

            MG["Metrics Generator<br/>━━━━━━━━━━━━━━━━━━━━<br/>• Span metrics<br/>• Service graphs<br/>• RED metrics<br/>━━━━━━━━━━━━━━━━━━━━<br/>Replicas: 3+"]
        end

        subgraph QueryLayer["Query Layer"]
            QF["Query Frontend<br/>━━━━━━━━━━━━━━━━━━━━<br/>• Request sharding<br/>• Caching<br/>• Load balancing<br/>━━━━━━━━━━━━━━━━━━━━<br/>Replicas: 2"]

            Q["Querier<br/>━━━━━━━━━━━━━━━━━━━━<br/>• TraceQL execution<br/>• Multi-source fetch<br/>━━━━━━━━━━━━━━━━━━━━<br/>Replicas: 5"]
        end

        subgraph StorageLayer["Storage Layer"]
            TDB[("TempoDB<br/>━━━━━━━━━━━━━━<br/>• Block management<br/>• WAL")]

            subgraph ObjectStorage["Object Storage"]
                S3[("S3 / GCS / Azure")]
            end
        end

        subgraph BackendWork["Backend Work"]
            SCHED["Backend Scheduler<br/>━━━━━━━━━━━━━━━━━━━━<br/>• Work scheduling<br/>• Job coordination<br/>━━━━━━━━━━━━━━━━━━━━<br/>Replicas: 1<br/>(singleton)"]

            WORK1["Backend Worker 1"]
            WORK2["Backend Worker 2"]
            WORKN["Backend Worker N<br/>━━━━━━━━━━━━━━━━━━━━<br/>• Job execution<br/>• Tenant index writes<br/>━━━━━━━━━━━━━━━━━━━━<br/>Replicas: 2+"]

            COMP["Compactor<br/>━━━━━━━━━━━━━━━━━━━━<br/>• Block merging<br/>• Retention enforcement<br/>• Deduplication<br/>━━━━━━━━━━━━━━━━━━━━<br/>Replicas: 5"]
        end

        subgraph Coordination["Coordination"]
            CACHE[("Cache<br/>━━━━━━━━━━━━━━<br/>Redis/Memcached<br/>Replicas: 3")]
            MEMBER["MemberlistKV<br/>Ring Management"]
        end
    end

    %% Write Path - With Adaptive Traces
    APP -->|"OTLP traces"| GW
    GW -->|"All tenants"| SGW

    SGW -->|"WITH policies:<br/>Head sampling"| KAFKA_AT
    KAFKA_AT --> SAMPLER
    SAMPLER <--> POLICY
    SAMPLER <--> DECIDX
    SAMPLER -->|"Sampled traces"| DIST

    %% Write Path - Without Adaptive Traces
    SGW -.->|"WITHOUT policies:<br/>Pass-through proxy"| DIST

    %% Tempo Write Path
    DIST --> KAFKA_TEMPO
    KAFKA_TEMPO --> BB1
    KAFKA_TEMPO --> BB2
    KAFKA_TEMPO --> BB3
    KAFKA_TEMPO --> BBN
    KAFKA_TEMPO --> LS_A
    KAFKA_TEMPO --> LS_B
    KAFKA_TEMPO --> MG

    BB1 --> TDB
    BB2 --> TDB
    BB3 --> TDB
    BBN --> TDB
    TDB --> S3

    %% Query Path
    GRAFANA -->|"TraceQL"| GW
    GW --> QF
    QF <--> CACHE
    QF --> Q
    Q --> LS_A
    Q --> LS_B
    Q --> TDB

    %% Backend Work Path
    SCHED --> WORK1
    SCHED --> WORK2
    SCHED --> WORKN
    WORK1 --> COMP
    WORK2 --> COMP
    WORKN --> COMP
    COMP --> TDB

    %% Styling
    style SGW fill:#4fc3f7
    style SAMPLER fill:#4fc3f7
    style KAFKA_AT fill:#4fc3f7
    style GW fill:#a5d6a7
    style DIST fill:#81c784
    style QF fill:#ce93d8
    style Q fill:#ce93d8
    style SCHED fill:#ffcc80
    style COMP fill:#ffcc80

    linkStyle 2 stroke:#4fc3f7,stroke-width:2px
    linkStyle 3 stroke:#4fc3f7,stroke-width:2px
    linkStyle 4 stroke:#4fc3f7,stroke-width:2px
    linkStyle 5 stroke:#4fc3f7,stroke-width:2px
    linkStyle 6 stroke:#4fc3f7,stroke-width:2px
    linkStyle 7 stroke:#b0bec5,stroke-width:2px,stroke-dasharray: 5 5
```

## Data Paths Legend

```mermaid
flowchart LR
    subgraph Legend["Path Legend"]
        direction LR
        P1["With Adaptive Traces"] ---|"Solid blue"| P2["Sampling applied"]
        P3["Without Adaptive Traces"] -.-|"Dashed gray"| P4["Pass-through"]
        P5["Query Path"] ---|"Purple"| P6["Read operations"]
        P7["Backend Work"] ---|"Orange"| P8["Maintenance"]
    end

    style P1 fill:#4fc3f7
    style P2 fill:#4fc3f7
    style P3 fill:#b0bec5
    style P4 fill:#b0bec5
    style P5 fill:#ce93d8
    style P6 fill:#ce93d8
    style P7 fill:#ffcc80
    style P8 fill:#ffcc80
```

## Component Replica Summary

| Component | Replicas | Type | Scaling Notes |
|-----------|----------|------|---------------|
| **Gateway** | 3+ | Deployment | Scales with request volume |
| **Sampler Gateway** | 3+ | Deployment | Scales with ingestion rate |
| **Sampler** | 3+ | StatefulSet | Scales with Kafka partitions |
| **Distributor** | 5 | Deployment | Scales with ingestion rate |
| **Block Builder** | N | StatefulSet | Matches Kafka partition count |
| **Live Store Zone-A** | N | StatefulSet | Scales with query load + partitions |
| **Live Store Zone-B** | N | StatefulSet | Scales with query load + partitions |
| **Metrics Generator** | 3+ | StatefulSet | Scales with trace volume |
| **Query Frontend** | 2 | Deployment | Usually 2 for HA |
| **Querier** | 5 | Deployment | Scales with query load |
| **Backend Scheduler** | **1** | StatefulSet | **Always singleton** |
| **Backend Worker** | 2+ | StatefulSet | Scales with compaction work |
| **Compactor** | 5 | Deployment | Scales with block count |
| **Cache (Memcached)** | 3 | StatefulSet | Scales with cache needs |

## Detailed Path Flows

### Path 1: Write with Adaptive Traces (Sampling Enabled)

```mermaid
flowchart LR
    A[App] --> B[Gateway<br/>3+]
    B --> C[Sampler GW<br/>3+]
    C -->|Head Sample| D[Kafka AT<br/>N parts]
    D --> E[Sampler<br/>3+]
    E -->|Tail Sample| F[Distributor<br/>5]
    F --> G[Kafka Tempo<br/>N parts]
    G --> H[Block Builder<br/>N]
    G --> I[Live Store<br/>2×N]
    H --> J[(Storage)]

    style C fill:#4fc3f7
    style D fill:#4fc3f7
    style E fill:#4fc3f7
```

### Path 2: Write without Adaptive Traces (Pass-through)

```mermaid
flowchart LR
    A[App] --> B[Gateway<br/>3+]
    B --> C[Sampler GW<br/>3+]
    C -->|Proxy| F[Distributor<br/>5]
    F --> G[Kafka Tempo<br/>N parts]
    G --> H[Block Builder<br/>N]
    G --> I[Live Store<br/>2×N]
    H --> J[(Storage)]

    style C fill:#b0bec5
```

### Path 3: Query

```mermaid
flowchart LR
    A[Grafana] --> B[Gateway<br/>3+]
    B --> C[Query Frontend<br/>2]
    C <--> D[(Cache<br/>3)]
    C --> E[Querier<br/>5]
    E --> F[Live Store<br/>2×N]
    E --> G[(TempoDB)]

    style C fill:#ce93d8
    style E fill:#ce93d8
```

### Path 4: Backend Work

```mermaid
flowchart LR
    A[Backend Scheduler<br/>1] --> B[Backend Worker<br/>2+]
    B --> C[Compactor<br/>5]
    C --> D[(TempoDB)]
    C -->|Retention| D

    style A fill:#ffcc80
    style B fill:#ffcc80
    style C fill:#ffcc80
```

## Kafka Partitions & Scaling

The number of Kafka partitions (N) determines scaling for several components:

```mermaid
flowchart TB
    KP["Kafka Partitions (N)"]

    KP --> BB["Block Builders<br/>N replicas<br/>(1 per partition)"]
    KP --> LS["Live Store<br/>N replicas per zone<br/>(2 zones = 2N total)"]
    KP --> MG["Metrics Generator<br/>Consumes all partitions"]
    KP --> SAMP["Sampler<br/>Partition-aware"]

    style KP fill:#ffeb3b
```

## High Availability Notes

- **Live Store**: 2 zones (zone-a, zone-b) for HA, anti-affinity ensures replicas on different nodes
- **Backend Scheduler**: Singleton - only 1 replica, uses leader election
- **Query Frontend**: Minimum 2 for HA
- **All StatefulSets**: Use persistent volumes for durability
