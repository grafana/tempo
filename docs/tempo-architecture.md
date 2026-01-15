# Tempo Architecture Overview

This document provides a comprehensive view of Tempo components across three repositories:
- **tempo** (main): Core tracing backend
- **adaptive-traces**: Intelligent sampling layer
- **backend-enterprise**: Gateway and enterprise features

## High-Level Architecture

```mermaid
flowchart TB
    subgraph Clients["Client Applications"]
        APP[Applications]
        GRAFANA[Grafana]
    end

    subgraph Enterprise["backend-enterprise (Gateway Layer)"]
        GW[Cloud Backend Gateway]
        AUTH[Auth/AuthZ]
        LBAC[Label-Based Access Control]
        BILLING[Billing Gateway]
        LICENSE[License Validation]
        OTLPGW[OTLP Gateway]

        GW --> AUTH
        GW --> LBAC
        GW --> BILLING
        GW --> LICENSE
        GW --> OTLPGW
    end

    subgraph Adaptive["adaptive-traces (Sampling Layer)"]
        SGW[Sampler Gateway<br/>Head Sampling]
        KAFKA[(Kafka)]
        SAMPLER[Sampler<br/>Tail Sampling]
        POLICY[Policy Store]
        DECIDX[Decision Index]
        CANARY[Canary]

        SGW --> KAFKA
        KAFKA --> SAMPLER
        POLICY <--> SAMPLER
        SAMPLER <--> DECIDX
    end

    subgraph Tempo["tempo (Core Backend)"]
        subgraph Write["Write Path"]
            DIST[Distributor]
            ING[Ingester]
            BB[Block Builder]
            MG[Metrics Generator]
        end

        subgraph Read["Read Path"]
            QF[Query Frontend]
            Q[Querier]
            LS[Live Store]
        end

        subgraph Storage["Storage Layer"]
            TEMPODB[(TempoDB)]
            WAL[WAL]
            S3[(S3)]
            GCS[(GCS)]
            AZURE[(Azure)]
            LOCAL[(Local)]
        end

        subgraph Backend["Backend Work"]
            SCHED[Backend Scheduler]
            WORKER[Backend Worker]
            COMP[Compactor]
        end

        subgraph Coord["Coordination"]
            MEMBER[MemberlistKV]
            OVER[Overrides]
            CACHE[(Cache)]
            RING[Partition Ring]
        end
    end

    APP -->|OTLP/Jaeger/Zipkin| GW
    GRAFANA -->|Queries| GW
    GW --> SGW
    GW --> DIST
    SAMPLER -->|Sampled Traces| DIST

    DIST --> ING
    DIST --> BB
    DIST --> MG

    ING --> TEMPODB
    BB --> TEMPODB

    TEMPODB --> WAL
    TEMPODB --> S3
    TEMPODB --> GCS
    TEMPODB --> AZURE
    TEMPODB --> LOCAL

    QF --> Q
    Q --> TEMPODB
    Q --> ING
    Q --> MG
    Q --> LS

    SCHED --> WORKER
    WORKER --> COMP
    COMP --> TEMPODB

    MEMBER --> RING
    OVER --> DIST
    OVER --> ING
    CACHE --> QF
```

## Detailed Component Diagram

```mermaid
flowchart LR
    subgraph GatewayLayer["Gateway Layer (backend-enterprise)"]
        direction TB
        CGW["Cloud Backend Gateway<br/>━━━━━━━━━━━━━━━━<br/>• Request routing<br/>• Multi-backend support"]

        subgraph AuthBlock["Authentication"]
            AUTH["Auth/AuthZ<br/>• OAuth/JWT<br/>• GCP JWT"]
            LBAC["LBAC<br/>• Label policies<br/>• Access control"]
        end

        subgraph EnterpriseFeatures["Enterprise Features"]
            BILL["Billing<br/>• Cost tracking<br/>• Usage metrics"]
            LIC["Licensing<br/>• Validation<br/>• Product keys"]
            ADMIN["Admin API<br/>• Token mgmt<br/>• Config"]
        end
    end

    subgraph SamplingLayer["Sampling Layer (adaptive-traces)"]
        direction TB
        SGW["Sampler Gateway<br/>━━━━━━━━━━━━━━━━<br/>• Head sampling<br/>• Rate limiting<br/>• Probabilistic"]

        KFK[("Kafka<br/>━━━━━━━━<br/>Partitioned<br/>ingestion")]

        SAMP["Sampler<br/>━━━━━━━━━━━━━━━━<br/>• Tail sampling<br/>• Anomaly detection<br/>• Diversity sampling"]

        subgraph SamplerSupport["Support Components"]
            POL["Policy Store<br/>• Per-tenant policies"]
            DEC["Decision Index<br/>• Keep/drop log"]
            CAN["Canary<br/>• Health monitor"]
        end
    end

    subgraph TempoCore["Tempo Core"]
        direction TB

        subgraph Ingestion["Ingestion Components"]
            DIST["Distributor<br/>━━━━━━━━━━━━━━━━<br/>• Entry point<br/>• Rate limiting<br/>• Validation<br/>• Sharding"]

            ING["Ingester<br/>━━━━━━━━━━━━━━━━<br/>• Memory buffer<br/>• WAL<br/>• Flush to storage"]

            BB["Block Builder<br/>━━━━━━━━━━━━━━━━<br/>• Kafka consumer<br/>• Block creation"]

            MG["Metrics Generator<br/>━━━━━━━━━━━━━━━━<br/>• Span metrics<br/>• RED metrics<br/>• Remote write"]
        end

        subgraph Query["Query Components"]
            QF["Query Frontend<br/>━━━━━━━━━━━━━━━━<br/>• Request sharding<br/>• Caching<br/>• Load balancing"]

            QR["Querier<br/>━━━━━━━━━━━━━━━━<br/>• TraceQL engine<br/>• Multi-source fetch"]

            LS["Live Store<br/>━━━━━━━━━━━━━━━━<br/>• Realtime data<br/>• In-memory"]
        end

        subgraph StorageLayer["Storage"]
            TDB[("TempoDB<br/>━━━━━━━━<br/>• Reader<br/>• Writer<br/>• Compactor")]

            WAL["WAL<br/>Write-Ahead Log"]

            subgraph Backends["Object Storage"]
                S3["S3"]
                GCS["GCS"]
                AZ["Azure"]
                LOC["Local"]
            end
        end

        subgraph BackendWork["Backend Work"]
            SCHED["Backend Scheduler<br/>• Work scheduling<br/>• Job coordination"]

            WORK["Backend Worker<br/>• Job execution<br/>• Tenant index"]

            COMP["Compactor<br/>• Block merging<br/>• Retention<br/>• Deduplication"]
        end

        subgraph Coordination["Coordination"]
            MEM["MemberlistKV<br/>Ring management"]
            OVR["Overrides<br/>Per-tenant config"]
            CCH[("Cache<br/>Redis/Memcached")]
            PRG["Partition Ring<br/>Kafka partitions"]
        end
    end

    CGW --> SGW
    CGW --> DIST
    SGW --> KFK
    KFK --> SAMP
    SAMP --> DIST

    DIST --> ING
    DIST --> BB
    DIST --> MG

    ING --> TDB
    BB --> TDB
    TDB --> WAL
    TDB --> Backends

    QF --> QR
    QR --> TDB
    QR --> ING
    QR --> LS

    SCHED --> WORK
    WORK --> COMP
    COMP --> TDB
```

## Data Flow Diagram

```mermaid
sequenceDiagram
    participant App as Application
    participant GW as Gateway<br/>(backend-enterprise)
    participant SG as Sampler Gateway<br/>(adaptive-traces)
    participant K as Kafka
    participant S as Sampler
    participant D as Distributor
    participant I as Ingester
    participant DB as TempoDB
    participant ST as Object Storage

    Note over App,ST: Write Path (Trace Ingestion)

    App->>GW: Send traces (OTLP)
    GW->>GW: Auth & tenant resolution

    alt With Sampling
        GW->>SG: Forward traces
        SG->>SG: Head sampling
        SG->>K: Write to partition
        K->>S: Consume traces
        S->>S: Tail sampling (anomaly, diversity)
        S->>D: Sampled traces
    else Direct Ingestion
        GW->>D: Forward traces
    end

    D->>D: Validate & rate limit
    D->>I: Route to ingester
    I->>I: Buffer in memory + WAL
    I->>DB: Flush block
    DB->>ST: Write to object storage
```

```mermaid
sequenceDiagram
    participant G as Grafana
    participant GW as Gateway
    participant QF as Query Frontend
    participant Q as Querier
    participant I as Ingester
    participant DB as TempoDB
    participant C as Cache

    Note over G,C: Read Path (Query Execution)

    G->>GW: TraceQL query
    GW->>GW: Auth & LBAC
    GW->>QF: Forward query
    QF->>C: Check cache

    alt Cache Hit
        C-->>QF: Cached result
    else Cache Miss
        QF->>QF: Shard request
        QF->>Q: Parallel queries

        par Query Sources
            Q->>I: Recent data
            Q->>DB: Historical data
        end

        Q-->>QF: Merge results
        QF->>C: Cache result
    end

    QF-->>GW: Response
    GW-->>G: Traces
```

## Component Responsibilities

```mermaid
mindmap
    root((Tempo<br/>Ecosystem))
        Gateway Layer
            Cloud Gateway
                Request routing
                Multi-tenant
            Authentication
                OAuth/JWT
                GCP JWT
            Authorization
                LBAC
                Label policies
            Enterprise
                Billing
                Licensing
                Admin API
        Sampling Layer
            Sampler Gateway
                Head sampling
                Rate limiting
            Sampler
                Tail sampling
                Anomaly detection
                Diversity sampling
            Support
                Policy Store
                Decision Index
                Canary
        Tempo Core
            Write Path
                Distributor
                Ingester
                Block Builder
                Metrics Generator
            Read Path
                Query Frontend
                Querier
                Live Store
            Storage
                TempoDB
                WAL
                S3/GCS/Azure
            Backend
                Scheduler
                Worker
                Compactor
            Coordination
                MemberlistKV
                Overrides
                Cache
```

## Deployment Modes

```mermaid
graph TB
    subgraph Single["SingleBinary Mode"]
        ALL[All Components<br/>in One Process]
    end

    subgraph Scalable["ScalableSingleBinary Mode"]
        PROC[Single Process]
        EXT[(External Storage)]
        PROC --> EXT
    end

    subgraph Distributed["Distributed Mode (Microservices)"]
        direction LR
        D1[Distributor]
        D2[Distributor]
        I1[Ingester]
        I2[Ingester]
        I3[Ingester]
        Q1[Querier]
        Q2[Querier]
        QF1[Query Frontend]
        C1[Compactor]
        MG1[Metrics Generator]
        BB1[Block Builder]
        LS1[Live Store]
        BS[Backend Scheduler]
        BW1[Backend Worker]
        BW2[Backend Worker]
    end
```

## Storage Architecture

```mermaid
graph TB
    subgraph Writers["Write Sources"]
        ING[Ingester]
        BB[Block Builder]
    end

    subgraph TempoDB["TempoDB Layer"]
        WAL[Write-Ahead Log]
        ENC[Encoding Layer<br/>Parquet]
        BLK[Block Manager]
        RET[Retention Manager]
    end

    subgraph Backends["Object Storage Backends"]
        S3[(AWS S3)]
        GCS[(Google GCS)]
        AZ[(Azure Blob)]
        LOCAL[(Local FS)]
    end

    subgraph Maintenance["Maintenance"]
        COMP[Compactor]
        SCHED[Backend Scheduler]
        WORK[Backend Worker]
    end

    ING --> WAL
    BB --> WAL
    WAL --> ENC
    ENC --> BLK
    BLK --> Backends

    COMP --> BLK
    SCHED --> WORK
    WORK --> BLK
    RET --> BLK
```

## Component Summary

| Layer | Repository | Component | Purpose |
|-------|------------|-----------|---------|
| **Gateway** | backend-enterprise | Cloud Backend Gateway | Multi-tenant request routing |
| | | Auth/AuthZ | Authentication and authorization |
| | | LBAC | Label-based access control |
| | | Billing Gateway | Cost tracking and usage |
| | | License Validation | Enterprise license management |
| **Sampling** | adaptive-traces | Sampler Gateway | Head sampling, rate limiting |
| | | Sampler | Tail sampling with policies |
| | | Policy Store | Per-tenant sampling policies |
| | | Decision Index | Sampling decision log |
| **Ingestion** | tempo | Distributor | Entry point, validation, routing |
| | | Ingester | Memory buffer, WAL, flush |
| | | Block Builder | Kafka-based block creation |
| | | Metrics Generator | Span-derived metrics |
| **Query** | tempo | Query Frontend | Sharding, caching, load balancing |
| | | Querier | TraceQL execution, multi-source |
| | | Live Store | Real-time in-memory data |
| **Storage** | tempo | TempoDB | Core storage abstraction |
| | | WAL | Write-ahead log |
| | | S3/GCS/Azure/Local | Object storage backends |
| **Backend** | tempo | Backend Scheduler | Work scheduling |
| | | Backend Worker | Job execution |
| | | Compactor | Block optimization, retention |
| **Coordination** | tempo | MemberlistKV | Distributed ring management |
| | | Overrides | Per-tenant configuration |
| | | Cache | Redis/Memcached caching |
| | | Partition Ring | Kafka partition management |
