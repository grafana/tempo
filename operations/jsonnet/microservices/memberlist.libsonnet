{
  local k = import 'ksonnet-util/kausal.libsonnet',
  local service = $.core.v1.service,
  local servicePort = $.core.v1.servicePort,

  // Headless service (= no assigned IP, DNS returns all targets instead) pointing to gossip network members.
  gossip_ring_service:
    service.new(
      'gossip-ring',  // name
      {
        [$._config.gossip_member_label]: 'true',
      },
      [
        servicePort.newNamed('gossip-ring', $._config.gossip_ring_port, $._config.gossip_ring_port) +
        servicePort.withProtocol('TCP'),
      ],
    ) + service.mixin.spec.withClusterIp('None'),
}
