## Grafana Agent
This example highlights setting up the Grafana Agent in a simple tracing pipeline.

1. First start up the stack.

```console
docker compose up -d
```

At this point, the following containers should be spun up -

```console
docker compose ps
```
```
       Name                     Command               State                                Ports                              
-----------------------------------------------------------------------------------------------------------
agent_agent_1        /bin/agent -config.file=/e ...   Up                                                                      
agent_grafana_1      /run.sh                          Up      0.0.0.0:3000->3000/tcp,:::3000->3000/tcp                        
agent_k6-tracing_1   /k6-tracing run /example-s ...   Up                                                                      
agent_prometheus_1   /bin/prometheus --config.f ...   Up      0.0.0.0:9090->9090/tcp,:::9090->9090/tcp                        
agent_tempo_1        /tempo -config.file=/etc/t ...   Up      0.0.0.0:32768->14268/tcp,:::32768->14268/tcp,                   
                                                              0.0.0.0:32772->3200/tcp,:::32772->3200/tcp,                     
                                                              0.0.0.0:32771->4317/tcp,:::32771->4317/tcp,                     
                                                              0.0.0.0:32770->4318/tcp,:::32770->4318/tcp,                     
                                                              0.0.0.0:32769->9411/tcp,:::32769->9411/tcp 
```

2. If you're interested you can see the wal/blocks as they are being created.
```console
ls tempo-data/
```

3. Navigate to [Grafana](http://localhost:3000/explore) select the Tempo data source and use the "Search"
tab to find traces.

4. To stop the setup use -

```console
docker compose down -v
```
