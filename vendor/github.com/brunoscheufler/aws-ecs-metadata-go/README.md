# AWS ECS Metadata Go

A minimal wrapper library to fetch Elastic Container Service (ECS) Task metadata from any Go service running in container provisioned by AWS Fargate.

Based on the Fargate platform version, you'll have access to different versions of the Task Metadata Endpoint. If you're running on 1.4.0,
you'll be able to access [Version 4](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-metadata-endpoint-v4.html), Fargate 1.3.0
and later support [Version 3](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-metadata-endpoint-v3.html) of the endpoint.

## installation

```bash
go get github.com/brunoscheufler/aws-ecs-metadata-go
```

## usage

This library allows you to retrieve the most recent metadata format available in your environment
based on the environment variables Fargate will provide. This means, that using `GetTaskMetadata` you'll
receive an empty interface which maps to either Version 3 or Version 4 of the Task Metadata structure.

```go
package main

import (
    "context"
    metadata "github.com/brunoscheufler/aws-ecs-metadata-go"
    "log"
    "net/http"
)

func main() {
    // Fetch ECS Task metadata from environment
    meta, err := metadata.Get(context.Background(), &http.Client{})
    if err != nil {
        panic(err)
    }

    // Based on the Fargate platform version, we'll have access
    // to v3 or v4 of the ECS Metadata format
    switch m := meta.(type) {
    case *metadata.TaskMetadataV3:
        log.Printf("%s %s:%s", m.Cluster, m.Family, m.Revision)
    case *metadata.TaskMetadataV4:
        log.Printf("%s(%s) %s:%s", m.Cluster, m.AvailabilityZone, m.Family, m.Revision)
    }
}
```
