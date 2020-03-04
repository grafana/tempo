{
    _images+:: {
        frigg: 'joeelliott/canary-frigg:de4f5e36',
        frigg_query: 'joeelliott/canary-frigg-query:de4f5e36',
    },

    _config+:: {
        compactor: {
            pvc_size: error 'Must specify a compactor pvc size',
            pvc_storage_class: error 'Must specify a compactor pvc storage class',
        },
        querier: {
            pvc_size: error 'Must specify a querier pvc size',
            pvc_storage_class: error 'Must specify a querier pvc storage class',
        },
        ingester: {
            pvc_size: error 'Must specify an ingester pvc size',
            pvc_storage_class: error 'Must specify an ingester pvc storage class',
        },
        receivers: error 'Must specify receivers',
        ballast_size_mbs: '1024',
        jaeger_ui: {
            base_path: '/',
        }
    },

    frigg_compactor_container+::
        $.util.resourcesRequests('500m', '3Gi') +
        $.util.resourcesLimits('1', '5Gi'),

    frigg_distributor_container+::
        $.util.resourcesRequests('3', '3Gi') +
        $.util.resourcesLimits('5', '5Gi'),

    frigg_ingester_container+::
        $.util.resourcesRequests('3', '3Gi') +
        $.util.resourcesLimits('5', '5Gi'),

    frigg_querier_container+::
        $.util.resourcesRequests('500m', '1Gi') +
        $.util.resourcesLimits('1', '2Gi'),
}