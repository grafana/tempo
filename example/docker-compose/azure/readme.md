## Azure
In this example tempo is configured to write data to Azure via [Azurite](https://github.com/Azure/Azurite) which presents an Azure compatible API.

1. First start up the azure stack.

```console
docker-compose up -d
```

At this point, the following containers should be spun up -

```console
docker-compose ps
```
```
       Name                     Command               State                                 Ports                             
------------------------------------------------------------------------------------------------------------------------------
azure_azure-cli_1    az storage container creat ...   Exit 0                                                                  
azure_azurite_1      docker-entrypoint.sh azuri ...   Up       0.0.0.0:10000->10000/tcp,:::10000->10000/tcp, 10001/tcp,       
                                                               10002/tcp                                                      
azure_grafana_1      /run.sh                          Up       0.0.0.0:3000->3000/tcp,:::3000->3000/tcp                       
azure_k6-tracing_1   /k6-tracing run /example-s ...   Up                                                                      
azure_prometheus_1   /bin/prometheus --config.f ...   Up       0.0.0.0:9090->9090/tcp,:::9090->9090/tcp                       
azure_tempo_1        /tempo -config.file=/etc/t ...   Up       0.0.0.0:32779->14268/tcp,:::32779->14268/tcp,                  
                                                               0.0.0.0:3200->3200/tcp,:::3200->3200/tcp   
```

2. If you're interested you can see the wal/blocks as they are being created.  Check [Azure Storage Explorer](https://azure.microsoft.com/en-us/features/storage-explorer/) 
and [Azurite docs](https://docs.microsoft.com/en-us/azure/storage/common/storage-use-azurite).
   
3. Navigate to [Grafana](http://localhost:3000/explore) select the Tempo data source and use the "Search"
tab to find traces. Also notice that you can query Tempo metrics from the Prometheus data source setup in 
Grafana.

4. To stop the setup use -

```console
docker-compose down -v
```
