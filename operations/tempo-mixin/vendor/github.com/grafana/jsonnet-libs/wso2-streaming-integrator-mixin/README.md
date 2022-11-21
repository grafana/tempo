# WSO2 Streaming Integrator Mixin

This mixin was designed based on the dashboards publicly available on [WSO2 Github](https://github.com/wso2/streaming-integrator/tree/master/modules/distribution/carbon-home/resources/dashboards). It is updated to use the most recent panel versions and enables both a course and fine grained evaluation of your Streaming Integrator instances and Siddhi server and applications.

1- Streaming Integrator Overall Statistics
![image](https://user-images.githubusercontent.com/9431850/150526548-4c25fe44-7d59-4357-9b60-fd09891ca8d3.png)
2- Streaming Integrator Application Statistics
![image](https://user-images.githubusercontent.com/9431850/150526244-38ca324b-df0b-4957-8d4d-d30c8fb2d168.png)
3- Siddhi Overall Statistics
![image](https://user-images.githubusercontent.com/9431850/150526144-6771082a-68e1-4d44-bdf5-98a9ee736eb8.png)
4- Siddhi Server Statistics
![image](https://user-images.githubusercontent.com/9431850/150526190-2ca23803-f08f-42b1-bd11-1ddeb2ae6a7b.png)
5- Siddhi Aggregation Statistics
![image](https://user-images.githubusercontent.com/9431850/150526607-75aeac3d-8b96-4b98-881d-47180b133cff.png)
6- Siddhi On-Demand Query Statistics
![image](https://user-images.githubusercontent.com/9431850/150526656-493988c6-b6b6-4f57-b0b5-204b35809a98.png)
7- Siddhi Query Statistics
![image](https://user-images.githubusercontent.com/9431850/150526735-7269c63b-1b12-4920-bece-b0e9a2ddf8cb.png)
8- Siddhi Sink Statistics
![image](https://user-images.githubusercontent.com/9431850/150526777-bc654fe9-9941-4a76-809d-24ca0bb81be0.png)
9- Siddhi Source Statistics
![image](https://user-images.githubusercontent.com/9431850/150526835-8ce768ce-0f4c-4888-87e5-187052a8f14b.png)
10- Siddhi Table Statistics
![image](https://user-images.githubusercontent.com/9431850/150526887-095fc69a-919d-4bce-a055-464ba5e30120.png)
11- Siddhi Stream Statistics
![image](https://user-images.githubusercontent.com/9431850/150684114-fc646457-5a6f-443d-8958-17b8115413ed.png)

To use it, you need to have `mixtool` and `jsonnetfmt` installed. If you have a working Go development environment, it's easiest to run the following:

```bash
$ go get github.com/monitoring-mixins/mixtool/cmd/mixtool
$ go get github.com/google/go-jsonnet/cmd/jsonnetfmt
```git commit -m ""

You can then build a directory `dashboard_out` with the JSON dashboard files for Grafana:

```bash
$ make build
```

For more advanced uses of mixins, see [Prometheus Monitoring Mixins docs](https://github.com/monitoring-mixins/docs).

