{
    _config+: {
        pvc_size: '',
        pvc_storage_class: '',
        ballast_size_mbs: '1024',
        jaeger_ui: {
            query_base_path: '',
        }
    },

    _images+: {
        frigg: 'joeelliott/canary-frigg:de4f5e36',
        frigg_query: 'joeelliott/canary-frigg-query:de4f5e36',
    },

    frigg_container+::
        $.util.resourcesRequests('3', '3Gi') +
        $.util.resourcesLimits('5', '5Gi')
}