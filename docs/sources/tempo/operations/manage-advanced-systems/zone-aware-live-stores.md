---
title: Zone-aware replication for live-stores
menuTitle: Zone-aware live-stores
description: Configure zone-aware live-stores for Tempo
weight: 600
aliases:
  - ../zone-aware-ingesters/ # https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/zone-aware-ingesters/
---

# Zone-aware replication for live-stores

Zone awareness is a feature that ensures data is replicated across failure domains (which we refer to as "zones") to provide greater reliability.
A failure domain is whatever you define it to be, but commonly may be an availability zone, data center, or server rack.

When zone awareness is set up for live-stores, each Tempo partition is owned by one live-store per zone.
This means that if a live-store in one zone becomes unavailable, the live-store in the other zone can continue serving queries for that partition.
While data is replicated across zones (RF2), the read quorum is 1. The queriers only need a response from one live-store per partition.
This provides high availability without requiring data deduplication on the read path.
