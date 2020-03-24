{
    _images+:: {
        tempo: 'joeelliott/canary-frigg:d46e19ef',
        tempo_query: 'joeelliott/canary-frigg-query:d46e19ef',
    },

    _config+:: {
        port: 3100,
        pvc_size: error 'Must specify a pvc size',
        pvc_storage_class: error 'Must specify a pvc storage class',
        receivers: error 'Must specify receivers',
        ballast_size_mbs: '1024',
        jaeger_ui: {
            base_path: '/',
        }
    },

    tempo_container+::
        $.util.resourcesRequests('3', '3Gi') +
        $.util.resourcesLimits('5', '5Gi')
}