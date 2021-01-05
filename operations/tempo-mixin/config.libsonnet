{
  _config+:: {
    jobs: {
      gateway: 'cortex-gw',
      query_frontend: 'query-frontend',
      querier: 'querier',
      ingester: 'ingester',
      distributor: 'distributor',
      compactor: 'compactor',
    },
  },
}