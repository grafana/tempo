[
  {
    // import the microservices example
    local tempo = import '../tempo-microservices/main.jsonnet',

    name: 'cluster name',
    apiServer: 'https://0.0.0.0:6443',
    namespace: 'namespace',

    data: tempo,

    dataOverride: {
      _images+:: {
        // images can be overridden here if desired
      },

      _config+:: {

        // config can be overridden here if desired

      },

    },

  },
]
