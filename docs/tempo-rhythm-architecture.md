# Tempo Rhythm Architecture

This document describes the Tempo Rhythm (new) architecture for two tenant configurations:
1. **With Adaptive Traces** - Intelligent sampling enabled
2. **Without Adaptive Traces** - Direct ingestion

## Architecture: Tenant WITH Adaptive Traces

```mermaid
flowchart TB
    subgraph Clients["Client Applications"]
        APP[Applications<br/>OTLP Instrumented]
    end

    subgraph Enterprise["backend-enterprise"]
        GW[Cloud Backend Gateway]
        AUTH[Auth/AuthZ]
        OTLPGW[OTLP Gateway<br/>Stack Detection]
    end

    subgraph Adaptive["adaptive-traces"]
        SGW[Sampler Gateway<br/>━━━━━━━━━━━━━━━━<br/>• Head Sampling<br/>• Probabilistic<br/>• Rate Limiting]

        KAFKA_AT[("Kafka<br/>(Adaptive Traces)")]

        SAMPLER[Sampler<br/>━━━━━━━━━━━━━━━━<br/>• Tail Sampling<br/>• Anomaly Detection<br/>• Diversity Sampling<br/>• Dynamic Sampling]

        POLICY[Policy Store<br/>Per-tenant policies]
        DECIDX[Decision Index]

        SGW --> KAFKA_AT
        KAFKA_AT --> SAMPLER
        POLICY <--> SAMPLER
        SAMPLER <--> DECIDX
    end

    subgraph Tempo["Tempo (Rhythm Architecture)"]
        DIST[Distributor<br/>━━━━━━━━━━━━━━━━<br/>• Validation<br/>• Rate Limiting<br/>• Routing]

        KAFKA_TEMPO[("Kafka<br/>(Tempo Ingest)")]

        subgraph WriteWorkers["Write Workers"]
            BB1[Block Builder]
            BB2[Block Builder]
            BB3[Block Builder]
        end

        subgraph ReadLayer["Read Layer"]
            LS[Live Store<br/>━━━━━━━━━━━━━━━━<br/>• Real-time queries<br/>• Kafka consumer<br/>• In-memory traces]

            MG[Metrics Generator<br/>━━━━━━━━━━━━━━━━<br/>• Span metrics<br/>• Service graphs]
        end

        subgraph QueryLayer["Query Layer"]
            QF[Query Frontend<br/>━━━━━━━━━━━━━━━━<br/>• Sharding<br/>• Caching]
            Q[Querier]
        end

        subgraph Storage["Storage"]
            TDB[(TempoDB)]
            OBJ[(Object Storage<br/>S3/GCS/Azure)]
        end

        subgraph BackendWork["Backend Work"]
            SCHED[Backend Scheduler]
            WORK1[Backend Worker]
            WORK2[Backend Worker]
            COMP[Compactor]
        end

        DIST --> KAFKA_TEMPO
        KAFKA_TEMPO --> WriteWorkers
        KAFKA_TEMPO --> LS
        KAFKA_TEMPO --> MG
        WriteWorkers --> TDB
        TDB --> OBJ

        QF --> Q
        Q --> TDB
        Q --> LS

        SCHED --> WORK1
        SCHED --> WORK2
        WORK1 --> COMP
        WORK2 --> COMP
        COMP --> TDB
    end

    APP -->|OTLP| GW
    GW --> AUTH
    GW --> OTLPGW
    OTLPGW --> SGW
    SAMPLER -->|Sampled Traces| DIST

    style Adaptive fill:#e1f5fe
    style SGW fill:#4fc3f7
    style SAMPLER fill:#4fc3f7
```

### Data Flow (With Adaptive Traces)

```mermaid
sequenceDiagram
    participant App as Application
    participant GW as Gateway
    participant SG as Sampler Gateway
    participant KA as Kafka<br/>(Adaptive)
    participant S as Sampler
    participant D as Distributor
    participant KT as Kafka<br/>(Tempo)
    participant BB as Block Builder
    participant LS as Live Store
    participant DB as TempoDB
    participant ST as Object Storage

    App->>GW: OTLP traces
    GW->>GW: Auth & tenant resolution
    GW->>SG: Forward to sampling

    Note over SG: Head Sampling<br/>Probabilistic reduction

    SG->>KA: Write to Kafka partition
    KA->>S: Consume traces

    Note over S: Tail Sampling<br/>• Anomaly detection<br/>• Diversity sampling<br/>• Policy-based decisions

    S->>D: Sampled traces only
    D->>D: Validate & rate limit
    D->>KT: Write to Kafka

    par Parallel Consumers
        KT->>BB: Block Builder consumes
        KT->>LS: Live Store consumes
    end

    BB->>DB: Create blocks
    DB->>ST: Write to object storage

    Note over LS: Serves real-time<br/>queries from memory
```

---

## Architecture: Tenant WITHOUT Adaptive Traces

Tenants without adaptive traces still route through the Sampler Gateway, but it acts as a **pass-through proxy** directly to the Distributor (no sampling applied).

```mermaid
flowchart TB
    subgraph Clients["Client Applications"]
        APP[Applications<br/>OTLP Instrumented]
    end

    subgraph Enterprise["backend-enterprise"]
        GW[Cloud Backend Gateway]
        AUTH[Auth/AuthZ]
        OTLPGW[OTLP Gateway<br/>Stack Detection]
    end

    subgraph Adaptive["adaptive-traces"]
        SGW[Sampler Gateway<br/>━━━━━━━━━━━━━━━━<br/>• Pass-through proxy<br/>• No sampling<br/>• Rate Limiting only]
    end

    subgraph Tempo["Tempo (Rhythm Architecture)"]
        DIST[Distributor<br/>━━━━━━━━━━━━━━━━<br/>• Validation<br/>• Rate Limiting<br/>• Routing]

        KAFKA_TEMPO[("Kafka<br/>(Tempo Ingest)")]

        subgraph WriteWorkers["Write Workers"]
            BB1[Block Builder]
            BB2[Block Builder]
            BB3[Block Builder]
        end

        subgraph ReadLayer["Read Layer"]
            LS[Live Store<br/>━━━━━━━━━━━━━━━━<br/>• Real-time queries<br/>• Kafka consumer<br/>• In-memory traces]

            MG[Metrics Generator<br/>━━━━━━━━━━━━━━━━<br/>• Span metrics<br/>• Service graphs]
        end

        subgraph QueryLayer["Query Layer"]
            QF[Query Frontend<br/>━━━━━━━━━━━━━━━━<br/>• Sharding<br/>• Caching]
            Q[Querier]
        end

        subgraph Storage["Storage"]
            TDB[(TempoDB)]
            OBJ[(Object Storage<br/>S3/GCS/Azure)]
        end

        subgraph BackendWork["Backend Work"]
            SCHED[Backend Scheduler]
            WORK1[Backend Worker]
            WORK2[Backend Worker]
            COMP[Compactor]
        end

        DIST --> KAFKA_TEMPO
        KAFKA_TEMPO --> WriteWorkers
        KAFKA_TEMPO --> LS
        KAFKA_TEMPO --> MG
        WriteWorkers --> TDB
        TDB --> OBJ

        QF --> Q
        Q --> TDB
        Q --> LS

        SCHED --> WORK1
        SCHED --> WORK2
        WORK1 --> COMP
        WORK2 --> COMP
        COMP --> TDB
    end

    APP -->|OTLP| GW
    GW --> AUTH
    GW --> OTLPGW
    OTLPGW --> SGW
    SGW -->|Proxy passthrough| DIST

    style SGW fill:#b0bec5
    style DIST fill:#81c784
```

### Data Flow (Without Adaptive Traces)

```mermaid
sequenceDiagram
    participant App as Application
    participant GW as Gateway
    participant SG as Sampler Gateway
    participant D as Distributor
    participant KT as Kafka<br/>(Tempo)
    participant BB as Block Builder
    participant LS as Live Store
    participant MG as Metrics Generator
    participant DB as TempoDB
    participant ST as Object Storage

    App->>GW: OTLP traces
    GW->>GW: Auth & tenant resolution
    GW->>SG: Forward to Sampler Gateway

    Note over SG: No policies for tenant<br/>Acts as pass-through proxy

    SG->>D: Proxy all traces (no sampling)

    D->>D: Validate & rate limit
    D->>KT: Write to Kafka

    par Parallel Consumers
        KT->>BB: Block Builder consumes
        KT->>LS: Live Store consumes
        KT->>MG: Metrics Generator consumes
    end

    BB->>DB: Create blocks
    DB->>ST: Write to object storage

    MG->>MG: Generate span metrics

    Note over LS: Serves real-time<br/>queries from memory
```

---

## Query Path (Same for Both)

```mermaid
sequenceDiagram
    participant G as Grafana
    participant GW as Gateway
    participant QF as Query Frontend
    participant Q as Querier
    participant LS as Live Store
    participant DB as TempoDB
    participant C as Cache

    G->>GW: TraceQL query
    GW->>GW: Auth & LBAC
    GW->>QF: Forward query

    QF->>C: Check cache

    alt Cache Hit
        C-->>QF: Return cached
    else Cache Miss
        QF->>QF: Shard request
        QF->>Q: Execute query

        par Query Sources
            Q->>LS: Recent data (real-time)
            Q->>DB: Historical data (blocks)
        end

        Q-->>QF: Merge results
        QF->>C: Cache result
    end

    QF-->>GW: Response
    GW-->>G: Traces
```

---

## Side-by-Side Comparison

```mermaid
flowchart LR
    subgraph WithAT["WITH Adaptive Traces"]
        direction TB
        A1[Gateway] --> A2[Sampler Gateway]
        A2 -->|Head Sampling| A3[Kafka AT]
        A3 --> A4[Sampler]
        A4 -->|Tail Sampling| A5[Distributor]
        A5 --> A6[Kafka Tempo]
        A6 --> A7[Block Builder]
        A6 --> A8[Live Store]
    end

    subgraph WithoutAT["WITHOUT Adaptive Traces"]
        direction TB
        B1[Gateway] --> B2[Sampler Gateway]
        B2 -->|Pass-through| B5[Distributor]
        B5 --> B6[Kafka Tempo]
        B6 --> B7[Block Builder]
        B6 --> B8[Live Store]
    end

    style A2 fill:#4fc3f7
    style A3 fill:#4fc3f7
    style A4 fill:#4fc3f7
    style B2 fill:#b0bec5
```

## Component Summary (Rhythm Architecture)

| Component | Role | Notes |
|-----------|------|-------|
| **Gateway** | Entry point | Auth, tenant resolution, routing |
| **Sampler Gateway** | Head sampling / proxy | All tenants route through; pass-through if no policies |
| **Sampler** | Tail sampling | Only with Adaptive Traces |
| **Distributor** | Validation & routing | Writes to Kafka |
| **Kafka** | Message queue | Central to Rhythm architecture |
| **Block Builder** | Block creation | Consumes from Kafka, writes blocks |
| **Live Store** | Real-time queries | In-memory, serves recent data (2 zones) |
| **Metrics Generator** | Span metrics | RED metrics, service graphs |
| **Query Frontend** | Query optimization | Sharding, caching |
| **Querier** | Query execution | Reads from Live Store + TempoDB |
| **Backend Scheduler** | Work coordination | Schedules compaction jobs (singleton) |
| **Backend Worker** | Job execution | Runs compaction tasks |
| **Compactor** | Block optimization | Merges blocks, enforces retention |
| **TempoDB** | Storage abstraction | Manages blocks and WAL |
