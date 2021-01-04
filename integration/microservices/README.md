# tempo-load-test
This repo aims to make it easier to measure and analyze tempo performance in micro-services mode.  There are already many examples for running tempo under load, but they use the single-binary approach and are not representative of what is occuring in larger installations.  Here tempo is run with separate containers for distributor and ingesters, and replication factor = 2, meaning that the distributor will mirror all incoming traces to 2 ingesters.  

![dashboard](/dashboard.png)

# What this repo contains
1. Tempo in micro-services mode
    1. 1x distributor
    1. 2x ingesters
    1. ReplicationFactor=2 meaning that the distributor mirrors incoming traces
1. S3/Min.IO virtual storage
1. Dashboard and metrics using
    1. Prometheus
    1. Grafana
    1. cadvisor - to gather container cpu usage and other metrics

# Instructions
This repo is expected to be used in conjuction with tempo development in a rapid feedback loop.  It is assumed you have a working go installation and a copy of tempo already cloned somewhere.

1. Build the tempo container
    1. Run `make docker-tempo`
    1. This tags a local image `tempo:latest`
1. Run this repo with docker-compose
    1. `docker-compose up -d`
    1. Browse to dashboard at http://localhost:3000/d/iaJI4FxMk/tempo-benchmarking
    1. When finished run `docker-compose down`
    
*Repeat steps 1-2 to see how code changes affect performance.*

## Controlling load
The synthetic-load-generator is included and configured to issue 1000 spans/s per instance.  By default 2 instances are ran which will issue 2000 spans/s.  Change the `scale:` value in `docker-compose.yaml` to increase or decrease the load as desired.

1. Edit docker-compose.yaml
1. Run `docker-compose up -d` to dynamically add or remove load generator instances.

# Key Metrics
As tempo is designed to be very horizontally scaleable, the key metrics are _per volume unit_, i.e. spans / s / cpu core.  
