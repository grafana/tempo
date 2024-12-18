# Microservices mode

This Nomad job will deploy Tempo in
[microservices mode](https://grafana.com/docs/tempo/latest/setup/deployment/#microservices-mode) using S3 backend.

## Usage

### Prerequisites
- S3 compatible storage
- [Nomad memory oversubscription](https://developer.hashicorp.com/nomad/tutorials/advanced-scheduling/memory-oversubscription)

Have a look at the job file and Tempo configuration file and change it to suite your environment.

### Run job

Inside directory with job run:

```shell
nomad job run tempo.hcl
```

To deploy a different version change `variable.version` default value or
specify from command line:

```shell
nomad job run -var="version=2.6.1" tempo.hcl
```

### Scale Tempo

Nomad CLI

```shell
nomad job scale tempo distributor <count>
```
