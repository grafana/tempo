local frigg = import '../../../operations/jsonnet/single-binary/frigg.libsonnet';

frigg {
    _config +:: {
        namespace: 'default',
        pvc_size: '30Gi',
        pvc_storage_class: 'local-path'
    },

    frigg_container+::
        $.util.resourcesRequests('1', '500Mi')
}
