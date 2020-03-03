{
    _images+:: {
        frigg: 'joeelliott/canary-frigg:de4f5e36',
        frigg_query: 'joeelliott/canary-frigg-query:de4f5e36',
    },

    _config+:: {
        pvc_size: error 'Must specify a pvc size',
        pvc_storage_class: error 'Must specify a pvc storage class',
        receivers: error 'Must specify receivers',
        ballast_size_mbs: '1024',
        jaeger_ui: {
            base_path: '/',
        }
    },

    frigg_container+::
        $.util.resourcesRequests('3', '3Gi') +
        $.util.resourcesLimits('5', '5Gi')
}