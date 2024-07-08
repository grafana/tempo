## Vulture

This example set up a local Tempo instance and Tempo vulture.

1. First create the storage directory with the correct permissions and start up the local stack.

```console
mkdir tempo-data/
docker compose up -d
```

At this point, the following containers should be spun up -

```console
docker compose ps
```
```
NAME                IMAGE                          COMMAND                  SERVICE   CREATED         STATUS         PORTS
vulture-tempo-1     grafana/tempo:latest           "/tempo -config.file…"   tempo     2 minutes ago   Up 2 minutes   0.0.0.0:3200->3200/tcp, 0.0.0.0:14250->14250/tcp
vulture-vulture-1   grafana/tempo-vulture:latest   "/tempo-vulture -tem…"   vulture   2 minutes ago   Up 2 minutes  
```

2. If you're interested you can see the wal/blocks as they are being created.

```console
ls tempo-data/
```

3. Tail logs of a container (eg: tempo)
```bash
docker logs vulture_tempo_1 -f
```

4. To stop the setup use -

```console
docker compose down -v
```

you can use Grafana or tempo-cli to make a query.

tempo-cli: `$ tempo-cli query api search "0.0.0.0:3200" --use-grpc "{}" "2023-12-05T08:11:18Z" "2023-12-05T08:12:18Z" --org-id="test"`
