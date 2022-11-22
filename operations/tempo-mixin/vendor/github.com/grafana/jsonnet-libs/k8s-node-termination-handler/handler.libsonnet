{
  // backwards compatible function
  new(config)::
    local _config = {
      namespace: 'kube-system',
      slack_webhook: '',
    } + config;
    (import 'gke.libsonnet') + _config,
}
