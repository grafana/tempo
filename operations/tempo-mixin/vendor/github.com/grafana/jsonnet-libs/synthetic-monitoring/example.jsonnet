local sm = import 'sm.libsonnet';

{
  syntheticMonitoring+:: {
    grafanaHttpCheck: sm.http.new('grafana', 'https://grafana.com/')
                      + sm.withProbes('all'),  // enable all probes
    grafanaPingCheck: sm.ping.new('grafana', 'grafana.com')
                      + sm.withProbes('continents'),  // one check per continent
    grafanaDnsCheck: sm.dns.new('grafana-dns', 'grafana.com')  // Combination of Job name and Target must be unique
                     + sm.withProbes('europe'),  // just check from Europe
    grafanaTcpCheck: sm.tcp.new('grafana', 'grafana.com:443')
                     + sm.withProbes('small'),  // just use a smaller, predefined set of checks
  },
}
