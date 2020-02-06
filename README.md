# Frigg

## To Run

```
make install-tools
make vendor
make docker-frigg
make docker-frigg-query
cd example
docker-compose up
```

- Frigg
  - http://localhost:3100
- Frigg-Query
  - http://localhost:16686
- Grafana
  - http://localhost:3000
- Prometheus
  - http://localhost:9090


## To Do

- [x] GCS Support
- [ ] Concurrent Queries
- [ ] Caching
- [ ] Compactor
- [ ] Optimize!

Let's go!
