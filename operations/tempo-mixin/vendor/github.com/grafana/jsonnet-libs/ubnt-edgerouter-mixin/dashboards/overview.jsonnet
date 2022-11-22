local grafana = import 'grafonnet/grafana.libsonnet';
local dashboard = grafana.dashboard;
local row = grafana.row;
local prometheus = grafana.prometheus;

local gBuilder = import 'grafana-builder/grafana.libsonnet';

local utils = import 'snmp-mixin/lib/utils.libsonnet';

local sharedMatcher = 'job=~"$job", instance=~"$instance", snmp_target=~"$snmp_target"';

local ifMatcher = 'ifIndex=~"$interface", ' + sharedMatcher;

local ipversionlabelmatcher(label, version) = '%s="%d", %s' % [label, version, sharedMatcher];
local ratequery(metric, matcher) = 'rate(' + metric + '{' + matcher + '}[$__rate_interval])';

local legendDecoration(right=true) = {
  options+: {
    legend: {
      showLegend: true,
      displayMode: 'table',
      placement: if right then 'right' else 'bottom',
      calcs: ['max', 'last'],
    },
  },
};

local onefieldnegativey(field) = {
  fieldConfig+: {
    overrides+: [
      {
        matcher: {
          id: 'byName',
          options: field,
        },
        properties: [
          {
            id: 'custom.transform',
            value: 'negative-Y',
          },
        ],
      },
    ],
  },
};

local outnegativey = {
  fieldConfig+: {
    overrides+: [
      {
        matcher: {
          id: 'byRegexp',
          options: '.*Out',
        },
        properties: [
          {
            id: 'custom.transform',
            value: 'negative-Y',
          },
        ],
      },
    ],
  },
};

local stack = {
  fieldConfig+: {
    defaults+: {
      custom+: {
        stacking: {
          mode: 'normal',
          group: 'A',
        },
      },
    },
  },
};

{
  queries:: {
    sysName: 'sysName{' + sharedMatcher + '}',
    sysDescr: 'sysDescr{' + sharedMatcher + '}',
    sysContact: 'sysContact{' + sharedMatcher + '}',
    sysLocation: 'sysLocation{' + sharedMatcher + '}',
    sysUptime: 'hrSystemUptime{' + sharedMatcher + '} / 100',  // Somehow this is neither seconds nor milliseconds, but rather 100th's of a second? ¯\_(ツ)_/¯
    sysUsers: 'hrSystemNumUsers{' + sharedMatcher + '}',
    sysProcesses: 'hrSystemProcesses{' + sharedMatcher + '}',
    cpuLoadAverage: 'laLoadInt{' + sharedMatcher + '} / 100',
    interruptRate: ratequery('ssRawInterrupts', sharedMatcher),
    contextRate: ratequery('ssRawContexts', sharedMatcher),
    blockIOSentRate: ratequery('ssIORawSent', sharedMatcher),
    blockIOReceivedRate: ratequery('ssIORawReceived', sharedMatcher),
    allCpuTicks: 'increase(ssCpuRawUser{' + sharedMatcher + '}[$__rate_interval]) + increase(ssCpuRawNice{' + sharedMatcher + '}[$__rate_interval]) + increase(ssCpuRawSystem{' + sharedMatcher + '}[$__rate_interval]) + increase(ssCpuRawIdle{' + sharedMatcher + '}[$__rate_interval])  + increase(ssCpuRawWait{' + sharedMatcher + '}[$__rate_interval]) + increase(ssCpuRawKernel{' + sharedMatcher + '}[$__rate_interval]) + increase(ssCpuRawInterrupt{' + sharedMatcher + '}[$__rate_interval]) + increase(ssCpuRawSoftIRQ{' + sharedMatcher + '}[$__rate_interval])',
    cpuUser: 'increase(ssCpuRawUser{' + sharedMatcher + '}[$__rate_interval]) / (' + $.queries.allCpuTicks + ')',
    cpuNice: 'increase(ssCpuRawNice{' + sharedMatcher + '}[$__rate_interval]) / (' + $.queries.allCpuTicks + ')',
    cpuSystem: 'increase(ssCpuRawSystem{' + sharedMatcher + '}[$__rate_interval]) / (' + $.queries.allCpuTicks + ')',
    cpuIdle: 'increase(ssCpuRawIdle{' + sharedMatcher + '}[$__rate_interval]) / (' + $.queries.allCpuTicks + ')',
    cpuWait: 'increase(ssCpuRawWait{' + sharedMatcher + '}[$__rate_interval]) / (' + $.queries.allCpuTicks + ')',
    cpuKernel: 'increase(ssCpuRawKernel{' + sharedMatcher + '}[$__rate_interval]) / (' + $.queries.allCpuTicks + ')',
    cpuInterrupt: 'increase(ssCpuRawInterrupt{' + sharedMatcher + '}[$__rate_interval]) / (' + $.queries.allCpuTicks + ')',
    cpuSoftIRQ: 'increase(ssCpuRawSoftIRQ{' + sharedMatcher + '}[$__rate_interval]) / (' + $.queries.allCpuTicks + ')',
    processorLoad: 'hrProcessorLoad{' + sharedMatcher + '}',
    memBuffer: 'memBuffer{' + sharedMatcher + '}',
    memCached: 'memCached{' + sharedMatcher + '}',
    memShared: 'memShared{' + sharedMatcher + '}',
    memFree: 'memTotalFree{' + sharedMatcher + '}',
    memAvailReal: 'memAvailReal{' + sharedMatcher + '}',
    // TODO: This is ugly because it filters on the hrStorageDescr label, which could be unreliable. Should either merge labels with the hrStorageType entry, or change the SNMP scraping.
    memUtilized: 'hrStorageUsed{hrStorageDescr="Physical memory",' + sharedMatcher + '} / hrStorageSize{hrStorageDescr="Physical memory",' + sharedMatcher + '}',
    ifInOctets: ratequery('ifHCInOctets', ifMatcher) + ' * 8',
    ifOutOctets: ratequery('ifHCOutOctets', ifMatcher) + ' * 8',
    ifInBroadcast: ratequery('ifHCInBroadcastPkts', ifMatcher),
    ifOutBroadcast: ratequery('ifHCOutBroadcastPkts', ifMatcher),
    ifInMulticast: ratequery('ifHCInMulticastPkts', ifMatcher),
    ifOutMulticast: ratequery('ifHCOutMulticastPkts', ifMatcher),
    ifInUcast: ratequery('ifHCInUcastPkts', ifMatcher),
    ifOutUcast: ratequery('ifHCOutUcastPkts', ifMatcher),
    ifInDiscards: ratequery('ifInDiscards', ifMatcher),
    ifOutDiscards: ratequery('ifOutDiscards', ifMatcher),
    ifInErrors: ratequery('ifInErrors', ifMatcher),
    ifOutErrors: ratequery('ifOutErrors', ifMatcher),
    ifInUnknownProtos: ratequery('ifInUnknownProtos', ifMatcher),
    ipv4SystemStatsHCInBcastPkts: ratequery('ipSystemStatsHCInBcastPkts', ipversionlabelmatcher('ipSystemStatsIPVersion', 1)),
    ipv4SystemStatsHCOutBcastPkts: ratequery('ipSystemStatsHCOutBcastPkts', ipversionlabelmatcher('ipSystemStatsIPVersion', 1)),
    ipv4SystemStatsHCInMcastPkts: ratequery('ipSystemStatsHCInMcastPkts', ipversionlabelmatcher('ipSystemStatsIPVersion', 1)),
    ipv4SystemStatsHCOutMcastPkts: ratequery('ipSystemStatsHCOutMcastPkts', ipversionlabelmatcher('ipSystemStatsIPVersion', 1)),
    ipv4SystemStatsHCInReceives: ratequery('ipSystemStatsHCInReceives', ipversionlabelmatcher('ipSystemStatsIPVersion', 1)),
    ipv4SystemStatsHCOutTransmits: ratequery('ipSystemStatsHCOutTransmits', ipversionlabelmatcher('ipSystemStatsIPVersion', 1)),

    ipv6SystemStatsHCInBcastPkts: ratequery('ipSystemStatsHCInBcastPkts', ipversionlabelmatcher('ipSystemStatsIPVersion', 2)),
    ipv6SystemStatsHCOutBcastPkts: ratequery('ipSystemStatsHCOutBcastPkts', ipversionlabelmatcher('ipSystemStatsIPVersion', 2)),
    ipv6SystemStatsHCInMcastPkts: ratequery('ipSystemStatsHCInMcastPkts', ipversionlabelmatcher('ipSystemStatsIPVersion', 2)),
    ipv6SystemStatsHCOutMcastPkts: ratequery('ipSystemStatsHCOutMcastPkts', ipversionlabelmatcher('ipSystemStatsIPVersion', 2)),
    ipv6SystemStatsHCInReceives: ratequery('ipSystemStatsHCInReceives', ipversionlabelmatcher('ipSystemStatsIPVersion', 2)),
    ipv6SystemStatsHCOutTransmits: ratequery('ipSystemStatsHCOutTransmits', ipversionlabelmatcher('ipSystemStatsIPVersion', 2)),

    ipv4SystemStatsHCInForwDatagrams: ratequery('ipSystemStatsHCInForwDatagrams', ipversionlabelmatcher('ipSystemStatsIPVersion', 1)),
    ipv4SystemStatsHCOutForwDatagrams: ratequery('ipSystemStatsHCOutForwDatagrams', ipversionlabelmatcher('ipSystemStatsIPVersion', 1)),

    ipv6SystemStatsHCInForwDatagrams: ratequery('ipSystemStatsHCInForwDatagrams', ipversionlabelmatcher('ipSystemStatsIPVersion', 2)),
    ipv6SystemStatsHCOutForwDatagrams: ratequery('ipSystemStatsHCOutForwDatagrams', ipversionlabelmatcher('ipSystemStatsIPVersion', 2)),

    ipv4SystemStatsReasmReqds: ratequery('ipSystemStatsReasmReqds', ipversionlabelmatcher('ipSystemStatsIPVersion', 1)),
    ipv4SystemStatsReasmOKs: ratequery('ipSystemStatsReasmOKs', ipversionlabelmatcher('ipSystemStatsIPVersion', 1)),
    ipv4SystemStatsReasmFails: ratequery('ipSystemStatsReasmFails', ipversionlabelmatcher('ipSystemStatsIPVersion', 1)),
    ipv4SystemStatsOutFragReqds: ratequery('ipSystemStatsOutFragReqds', ipversionlabelmatcher('ipSystemStatsIPVersion', 1)),
    ipv4SystemStatsOutFragCreates: ratequery('ipSystemStatsOutFragCreates', ipversionlabelmatcher('ipSystemStatsIPVersion', 1)),
    ipv4SystemStatsOutFragOKs: ratequery('ipSystemStatsOutFragOKs', ipversionlabelmatcher('ipSystemStatsIPVersion', 1)),
    ipv4SystemStatsOutFragFails: ratequery('ipSystemStatsOutFragFails', ipversionlabelmatcher('ipSystemStatsIPVersion', 1)),

    ipv6SystemStatsReasmReqds: ratequery('ipSystemStatsReasmReqds', ipversionlabelmatcher('ipSystemStatsIPVersion', 2)),
    ipv6SystemStatsReasmOKs: ratequery('ipSystemStatsReasmOKs', ipversionlabelmatcher('ipSystemStatsIPVersion', 2)),
    ipv6SystemStatsReasmFails: ratequery('ipSystemStatsReasmFails', ipversionlabelmatcher('ipSystemStatsIPVersion', 2)),
    ipv6SystemStatsOutFragReqds: ratequery('ipSystemStatsOutFragReqds', ipversionlabelmatcher('ipSystemStatsIPVersion', 2)),
    ipv6SystemStatsOutFragCreates: ratequery('ipSystemStatsOutFragCreates', ipversionlabelmatcher('ipSystemStatsIPVersion', 2)),
    ipv6SystemStatsOutFragOKs: ratequery('ipSystemStatsOutFragOKs', ipversionlabelmatcher('ipSystemStatsIPVersion', 2)),
    ipv6SystemStatsOutFragFails: ratequery('ipSystemStatsOutFragFails', ipversionlabelmatcher('ipSystemStatsIPVersion', 2)),

    ipv4SystemStatsInAddrErrors: ratequery('ipSystemStatsInAddrErrors', ipversionlabelmatcher('ipSystemStatsIPVersion', 1)),
    ipv4SystemStatsInHdrErrors: ratequery('ipSystemStatsInHdrErrors', ipversionlabelmatcher('ipSystemStatsIPVersion', 1)),
    ipv4SystemStatsInNoRoutes: ratequery('ipSystemStatsInNoRoutes', ipversionlabelmatcher('ipSystemStatsIPVersion', 1)),
    ipv4SystemStatsInTruncatedPkts: ratequery('ipSystemStatsInTruncatedPkts', ipversionlabelmatcher('ipSystemStatsIPVersion', 1)),
    ipv4SystemStatsInDiscards: ratequery('ipSystemStatsInDiscards', ipversionlabelmatcher('ipSystemStatsIPVersion', 1)),
    ipv4SystemStatsOutDiscards: ratequery('ipSystemStatsOutDiscards', ipversionlabelmatcher('ipSystemStatsIPVersion', 1)),

    ipv6SystemStatsInAddrErrors: ratequery('ipSystemStatsInAddrErrors', ipversionlabelmatcher('ipSystemStatsIPVersion', 2)),
    ipv6SystemStatsInHdrErrors: ratequery('ipSystemStatsInHdrErrors', ipversionlabelmatcher('ipSystemStatsIPVersion', 2)),
    ipv6SystemStatsInNoRoutes: ratequery('ipSystemStatsInNoRoutes', ipversionlabelmatcher('ipSystemStatsIPVersion', 2)),
    ipv6SystemStatsInTruncatedPkts: ratequery('ipSystemStatsInTruncatedPkts', ipversionlabelmatcher('ipSystemStatsIPVersion', 2)),
    ipv6SystemStatsInDiscards: ratequery('ipSystemStatsInDiscards', ipversionlabelmatcher('ipSystemStatsIPVersion', 2)),
    ipv6SystemStatsOutDiscards: ratequery('ipSystemStatsOutDiscards', ipversionlabelmatcher('ipSystemStatsIPVersion', 2)),

    ipv4SystemStatsHCInDelivers: ratequery('ipSystemStatsHCInDelivers', ipversionlabelmatcher('ipSystemStatsIPVersion', 1)),
    ipv4SystemStatsHCOutRequests: ratequery('ipSystemStatsHCOutRequests', ipversionlabelmatcher('ipSystemStatsIPVersion', 1)),

    ipv6SystemStatsHCInDelivers: ratequery('ipSystemStatsHCInDelivers', ipversionlabelmatcher('ipSystemStatsIPVersion', 2)),
    ipv6SystemStatsHCOutRequests: ratequery('ipSystemStatsHCOutRequests', ipversionlabelmatcher('ipSystemStatsIPVersion', 2)),

    ipv4SystemStatsInUnknownProtos: ratequery('ipSystemStatsInUnknownProtos', ipversionlabelmatcher('ipSystemStatsIPVersion', 1)),
    ipv4SystemStatsOutNoRoutes: ratequery('ipSystemStatsOutNoRoutes', ipversionlabelmatcher('ipSystemStatsIPVersion', 1)),

    ipv6SystemStatsInUnknownProtos: ratequery('ipSystemStatsInUnknownProtos', ipversionlabelmatcher('ipSystemStatsIPVersion', 2)),
    ipv6SystemStatsOutNoRoutes: ratequery('ipSystemStatsOutNoRoutes', ipversionlabelmatcher('ipSystemStatsIPVersion', 2)),

    tcpActiveOpens: ratequery('tcpActiveOpens', sharedMatcher),
    tcpPassiveOpens: ratequery('tcpPassiveOpens', sharedMatcher),
    tcpAttemptFails: ratequery('tcpAttemptFails', sharedMatcher),
    tcpEstabResets: ratequery('tcpEstabResets', sharedMatcher),

    tcpCurrEstab: ratequery('tcpCurrEstab', sharedMatcher),

    tcpInSegs: ratequery('tcpInSegs', sharedMatcher),
    tcpOutSegs: ratequery('tcpOutSegs', sharedMatcher),
    tcpRetransSegs: ratequery('tcpRetransSegs', sharedMatcher),
    tcpInErrs: ratequery('tcpInErrs', sharedMatcher),
    tcpOutRsts: ratequery('tcpOutRsts', sharedMatcher),

    udpInDatagrams: ratequery('udpInDatagrams', sharedMatcher),
    udpOutDatagrams: ratequery('udpOutDatagrams', sharedMatcher),
    udpInErrors: ratequery('udpInErrors', sharedMatcher),
    udpNoPorts: ratequery('udpNoPorts', sharedMatcher),

    icmpIpv4StatsInMsgs: ratequery('icmpStatsInMsgs', ipversionlabelmatcher('icmpStatsIPVersion', 1)),
    icmpIpv4StatsOutMsgs: ratequery('icmpStatsOutMsgs', ipversionlabelmatcher('icmpStatsIPVersion', 1)),
    icmpIpv4StatsInErrors: ratequery('icmpStatsInErrors', ipversionlabelmatcher('icmpStatsIPVersion', 1)),
    icmpIpv4StatsOutErrors: ratequery('icmpStatsOutErrors', ipversionlabelmatcher('icmpStatsIPVersion', 1)),

    icmpIpv6StatsInMsgs: ratequery('icmpStatsInMsgs', ipversionlabelmatcher('icmpStatsIPVersion', 2)),
    icmpIpv6StatsOutMsgs: ratequery('icmpStatsOutMsgs', ipversionlabelmatcher('icmpStatsIPVersion', 2)),
    icmpIpv6StatsInErrors: ratequery('icmpStatsInErrors', ipversionlabelmatcher('icmpStatsIPVersion', 2)),
    icmpIpv6StatsOutErrors: ratequery('icmpStatsOutErrors', ipversionlabelmatcher('icmpStatsIPVersion', 2)),

    icmpIpv4MsgStatsInPkts: ratequery('icmpMsgStatsInPkts', ipversionlabelmatcher('icmpMsgStatsIPVersion', 1)),
    icmpIpv4MsgStatsOutPkts: ratequery('icmpMsgStatsOutPkts', ipversionlabelmatcher('icmpMsgStatsIPVersion', 1)),

    icmpIpv6MsgStatsInPkts: ratequery('icmpMsgStatsInPkts', ipversionlabelmatcher('icmpMsgStatsIPVersion', 2)),
    icmpIpv6MsgStatsOutPkts: ratequery('icmpMsgStatsOutPkts', ipversionlabelmatcher('icmpMsgStatsIPVersion', 2)),

    ipRoutingDiscards: 'ipRoutingDiscards{' + sharedMatcher + '}',
    inetCidrRouteNumber: 'inetCidrRouteNumber{' + sharedMatcher + '}',
    ipForwardNumber: 'ipForwardNumber{' + sharedMatcher + '}',

    snmpInPkts: ratequery('snmpInPkts', sharedMatcher),
    snmpOutPkts: ratequery('snmpOutPkts', sharedMatcher),
    snmpInGetNext: ratequery('snmpInGetNexts', sharedMatcher),
    snmpInGetRequests: ratequery('snmpInGetRequests', sharedMatcher),
    snmpInGetResponses: ratequery('snmpInGetResponses', sharedMatcher),
    snmpOutGetNext: ratequery('snmpOutGetNexts', sharedMatcher),
    snmpOutGetRequests: ratequery('snmpOutGetRequests', sharedMatcher),
    snmpOutGetResponses: ratequery('snmpOutGetResponses', sharedMatcher),

    snmpInTotalReqVars: ratequery('snmpInTotalReqVars', sharedMatcher),
  },

  panels:: {
    infoTable:
      gBuilder.tablePanel(
        [$.queries.sysName, $.queries.sysDescr, $.queries.sysContact, $.queries.sysLocation],
        {
          sysName: { alias: 'Name' },
          snmp_target: { alias: 'Hostname' },
          sysDescr: { alias: 'Description' },
          sysContact: { alias: 'Contact' },
          sysLocation: { alias: 'Location' },
        },
      ) +
      {
        datasource: '$datasource',
        title: 'System Information',
        description: 'System details for each discovered edgerouter device',
        fieldConfig+: {
          defaults+: {
            unit: 'string',
          },
        },
        transformations: [
          {
            id: 'joinByField',
            options: {
              byField: 'snmp_target',
              mode: 'outer',
            },
          },
          {
            id: 'filterFieldsByName',
            options: {
              include: {
                pattern: 'snmp_target|sys(Name|Descr|Contact|Location)',
              },
            },
          },
          {
            id: 'organize',
            options: {
              excludeByName: {},
              indexByName: {
                sysName: 0,
                snmp_target: 1,
                sysDescr: 2,
                sysContact: 3,
                sysLocation: 4,
              },
              renameByName: {},
            },
          },
        ],
      },
    sysUptime:
      grafana.statPanel.new(
        'Uptime',
        description='The time since the network management portion of the system was last re-initialized.',
        datasource='$datasource',
        colorMode='background',
        graphMode='none',
        noValue='No Data',
        unit='s',
        reducerFunction='last',
      )
      .addTarget(
        grafana.prometheus.target($.queries.sysUptime),
      ) + $.fixedColorMode('light-blue'),
    users:
      grafana.statPanel.new(
        'Users',
        description='The number of user sessions for which this host is storing state information. A session is a collection of processes requiring a single act of user authentication and possibly subject to collective job control.',
        datasource='$datasource',
        colorMode='background',
        noValue='No Data',
        reducerFunction='last',
        unit='short',
      )
      .addTarget(
        grafana.prometheus.target($.queries.sysUsers),
      ) + $.fixedColorMode('light-purple'),
    processes:
      grafana.statPanel.new(
        'Processes',
        description='The number of process contexts currently loaded or running on this system.',
        datasource='$datasource',
        colorMode='background',
        noValue='No Data',
        reducerFunction='last',
        unit='short',
      )
      .addTarget(
        grafana.prometheus.target($.queries.sysProcesses),
      ) + $.fixedColorMode('light-blue'),
    cpuLoadAverage:
      grafana.graphPanel.new(
        'CPU Load Average',
        description='The 1,5 and 15 minute load averages.',
        datasource='$datasource',
      )
      .addTarget(
        grafana.prometheus.target($.queries.cpuLoadAverage, legendFormat='{{laNames}}')
      ) + utils.timeSeriesOverride(
        unit='short',
        fillOpacity=10,
        showPoints='never',
      ),
    interruptsAndContexts:
      grafana.graphPanel.new(
        'Interrupts / Ctx-Switches (/sec)',
        description='Number of context switches, and interrupts processed.',
        datasource='$datasource',
      )
      .addTargets([
        grafana.prometheus.target($.queries.contextRate, legendFormat='Context Switches'),
        grafana.prometheus.target($.queries.interruptRate, legendFormat='Interrupts'),
      ]) + utils.timeSeriesOverride(
        unit='short',
        fillOpacity=10,
        showPoints='never',
      ),
    blockIO:
      grafana.graphPanel.new(
        'Block I/O (/sec)',
        description='Number of blocks sent to, or received from, a block device.',
        datasource='$datasource',
      )
      .addTargets([
        grafana.prometheus.target($.queries.blockIOReceivedRate, legendFormat='Blocks Received'),
        grafana.prometheus.target($.queries.blockIOSentRate, legendFormat='Blocks Sent'),
      ]) + utils.timeSeriesOverride(
        unit='short',
        fillOpacity=10,
        showPoints='never',
      ) + onefieldnegativey('Blocks Sent'),
    cpuTime:
      grafana.graphPanel.new(
        'CPU Time',
        description='Percentage of CPU time spent by state.',
        datasource='$datasource',
      )
      .addTargets([
        grafana.prometheus.target($.queries.cpuUser, legendFormat='User'),
        grafana.prometheus.target($.queries.cpuNice, legendFormat='Nice'),
        grafana.prometheus.target($.queries.cpuSystem, legendFormat='System'),
        grafana.prometheus.target($.queries.cpuIdle, legendFormat='Idle'),
        grafana.prometheus.target($.queries.cpuWait, legendFormat='I/O Wait'),
        grafana.prometheus.target($.queries.cpuKernel, legendFormat='Kernel'),
        grafana.prometheus.target($.queries.cpuInterrupt, legendFormat='Interrupt'),
        grafana.prometheus.target($.queries.cpuSoftIRQ, legendFormat='Soft IRQ'),
      ]) + stack + utils.timeSeriesOverride(
        unit='percentunit',
        fillOpacity=10,
        showPoints='never',
      ),
    processorLoad:
      grafana.graphPanel.new(
        'Processor Load (1 Min Average)',
        description='The average, over the last minute, of the percentage of time that this processor was not idle.',
        datasource='$datasource'
      )
      .addTarget(
        grafana.prometheus.target($.queries.processorLoad, legendFormat='Device ID: {{hrDeviceIndex}}')
      ) + utils.timeSeriesOverride(
        unit='short',
        fillOpacity=10,
        showPoints='never',
      ),
    memDistribution:
      grafana.graphPanel.new(
        'Memory Distribution',
        description='The total amount of real or virtual memory currently allocated, by type.',
        datasource='$datasource'
      )
      .addTargets([
        grafana.prometheus.target($.queries.memBuffer, legendFormat='Buffer'),
        grafana.prometheus.target($.queries.memCached, legendFormat='Cached'),
        grafana.prometheus.target($.queries.memFree, legendFormat='Free'),
        grafana.prometheus.target($.queries.memShared, legendFormat='Shared'),
      ]) + utils.timeSeriesOverride(
        unit='deckbytes',
        fillOpacity=10,
        showPoints='never',
      ),
    memAvail:
      grafana.graphPanel.new(
        'Memory Available',
        description='The amount of real/physical memory currently unused or available.',
        datasource='$datasource'
      )
      .addTarget(
        grafana.prometheus.target($.queries.memAvailReal, legendFormat='Real')
      ) + utils.timeSeriesOverride(
        unit='deckbytes',
        fillOpacity=10,
        showPoints='never',
      ),
    memUtilized:
      grafana.graphPanel.new(
        'Memory Utilized',
        description='The amount of physical memory which is currently allocated.',
        datasource='$datasource'
      )
      .addTarget(
        grafana.prometheus.target($.queries.memUtilized, legendFormat='{{hrStorageDescr}}')
      ) + utils.timeSeriesOverride(
        unit='percentunit',
        fillOpacity=10,
        showPoints='never',
      ),
    ifTraffic:
      grafana.graphPanel.new(
        'Interface Traffic',
        description='Total throughput per interface, by direction.',
        datasource='$datasource'
      )
      .addTargets([
        grafana.prometheus.target($.queries.ifInOctets, legendFormat='{{ifName}}:In'),
        grafana.prometheus.target($.queries.ifOutOctets, legendFormat='{{ifName}}:Out'),
      ]) + utils.timeSeriesOverride(
        unit='bps',
        fillOpacity=10,
        showPoints='never',
      ) + outnegativey,
    ifTrafficDist:
      grafana.graphPanel.new(
        'Interface Traffic Distribution',
        description='Packet throughput per interface, by type and direction.',
        datasource='$datasource'
      )
      .addTargets([
        grafana.prometheus.target($.queries.ifInBroadcast, legendFormat='{{ifName}}:Bcast:In'),
        grafana.prometheus.target($.queries.ifOutBroadcast, legendFormat='{{ifName}}:Bcast:Out'),
        grafana.prometheus.target($.queries.ifInMulticast, legendFormat='{{ifName}}:Mcast:In'),
        grafana.prometheus.target($.queries.ifOutMulticast, legendFormat='{{ifName}}:Mcast:Out'),
        grafana.prometheus.target($.queries.ifInUcast, legendFormat='{{ifName}}:Ucast:In'),
        grafana.prometheus.target($.queries.ifOutUcast, legendFormat='{{ifName}}:Ucast:Out'),
      ]) + utils.timeSeriesOverride(
        unit='pps',
        fillOpacity=10,
        showPoints='never',
      ) + outnegativey,
    ifTrafficErrsDrops:
      grafana.graphPanel.new(
        'Interface Drops/Errors',
        description='Number of packets dropped, in error, or unknown, by interface.',
        datasource='$datasource',
      )
      .addTargets([
        grafana.prometheus.target($.queries.ifInDiscards, legendFormat='{{ifName}}:Drop:In'),
        grafana.prometheus.target($.queries.ifOutDiscards, legendFormat='{{ifName}}:Drop:Out'),
        grafana.prometheus.target($.queries.ifInErrors, legendFormat='{{ifName}}:Err:In'),
        grafana.prometheus.target($.queries.ifOutErrors, legendFormat='{{ifName}}:Err:Out'),
        grafana.prometheus.target($.queries.ifInUnknownProtos, legendFormat='{{ifName}}:Unkwn:In'),
      ]) + utils.timeSeriesOverride(
        unit='pps',
        fillOpacity=10,
        showPoints='never',
      ) + outnegativey,
    ipDistribution:
      grafana.graphPanel.new(
        'IP Distribution',
        description='Datagram packet throughput by direction and IP version.',
        datasource='$datasource'
      )
      .addTargets([
        grafana.prometheus.target($.queries.ipv4SystemStatsHCInBcastPkts, legendFormat='IPv4:Bcast:In'),
        grafana.prometheus.target($.queries.ipv4SystemStatsHCOutBcastPkts, legendFormat='IPv4:Bcast:Out'),
        grafana.prometheus.target($.queries.ipv4SystemStatsHCInMcastPkts, legendFormat='IPv4:Mcast:In'),
        grafana.prometheus.target($.queries.ipv4SystemStatsHCOutMcastPkts, legendFormat='IPv4:Mcast:Out'),
        grafana.prometheus.target($.queries.ipv4SystemStatsHCInReceives, legendFormat='IPv4:Total:In'),
        grafana.prometheus.target($.queries.ipv4SystemStatsHCOutTransmits, legendFormat='IPv4:Total:Out'),

        grafana.prometheus.target($.queries.ipv6SystemStatsHCInBcastPkts, legendFormat='IPv6:Bcast:In'),
        grafana.prometheus.target($.queries.ipv6SystemStatsHCOutBcastPkts, legendFormat='IPv6:Bcast:Out'),
        grafana.prometheus.target($.queries.ipv6SystemStatsHCInMcastPkts, legendFormat='IPv6:Mcast:In'),
        grafana.prometheus.target($.queries.ipv6SystemStatsHCOutMcastPkts, legendFormat='IPv6:Mcast:Out'),
        grafana.prometheus.target($.queries.ipv6SystemStatsHCInReceives, legendFormat='IPv6:Total:In'),
        grafana.prometheus.target($.queries.ipv6SystemStatsHCOutTransmits, legendFormat='IPv6:Total:Out'),
      ]) + utils.timeSeriesOverride(
        unit='pps',
        fillOpacity=10,
        showPoints='never',
      ) + outnegativey,
    ipFrag:
      grafana.graphPanel.new(
        'IP Fragmentation (/sec)',
        description='Number of fragements by action, direction and IP version.',
        datasource='$datasource'
      )
      .addTargets([
        grafana.prometheus.target($.queries.ipv4SystemStatsReasmReqds, legendFormat='IPv4:ReasmReqd:In'),
        grafana.prometheus.target($.queries.ipv4SystemStatsReasmOKs, legendFormat='IPv4:ReasmOK:In'),
        grafana.prometheus.target($.queries.ipv4SystemStatsReasmFails, legendFormat='IPv4:ReasmFail:In'),
        grafana.prometheus.target($.queries.ipv4SystemStatsOutFragCreates, legendFormat='IPv4:FragCreate:Out'),
        grafana.prometheus.target($.queries.ipv4SystemStatsOutFragFails, legendFormat='IPv4:FragFail:Out'),
        grafana.prometheus.target($.queries.ipv4SystemStatsOutFragOKs, legendFormat='IPv4:FragOK:Out'),
        grafana.prometheus.target($.queries.ipv4SystemStatsOutFragReqds, legendFormat='IPv4:FragReqd:Out'),


        grafana.prometheus.target($.queries.ipv6SystemStatsReasmReqds, legendFormat='IPv6:ReasmReqd:In'),
        grafana.prometheus.target($.queries.ipv6SystemStatsReasmOKs, legendFormat='IPv6:ReasmOK:In'),
        grafana.prometheus.target($.queries.ipv6SystemStatsReasmFails, legendFormat='IPv6:ReasmFail:In'),
        grafana.prometheus.target($.queries.ipv6SystemStatsOutFragCreates, legendFormat='IPv6:FragCreate:Out'),
        grafana.prometheus.target($.queries.ipv6SystemStatsOutFragFails, legendFormat='IPv6:FragFail:Out'),
        grafana.prometheus.target($.queries.ipv6SystemStatsOutFragOKs, legendFormat='IPv6:FragOK:Out'),
        grafana.prometheus.target($.queries.ipv6SystemStatsOutFragReqds, legendFormat='IPv6:FragReqd:Out'),
      ]) + utils.timeSeriesOverride(
        unit='short',
        fillOpacity=10,
        showPoints='never',
      ) + outnegativey,
    ipForward:
      grafana.graphPanel.new(
        'IP Forwarding',
        description='Number of IP datagrams forwarded by direction and IP version.',
        datasource='$datasource'
      )
      .addTargets([
        grafana.prometheus.target($.queries.ipv4SystemStatsHCInForwDatagrams, legendFormat='IPv4:In'),
        grafana.prometheus.target($.queries.ipv4SystemStatsHCOutForwDatagrams, legendFormat='IPv4:Out'),
        grafana.prometheus.target($.queries.ipv6SystemStatsHCInForwDatagrams, legendFormat='IPv6:In'),
        grafana.prometheus.target($.queries.ipv6SystemStatsHCOutForwDatagrams, legendFormat='IPv6:Out'),
      ]) + utils.timeSeriesOverride(
        unit='short',
        fillOpacity=10,
        showPoints='never',
      ) + outnegativey,
    ipForwardDropErr:
      grafana.graphPanel.new(
        'IP Forwarding Drops/Errors',
        description='Number of IP forwarding datagrams dropped, or in error, by reason, direction, and IP version.',
        datasource='$datasource'
      )
      .addTargets([
        grafana.prometheus.target($.queries.ipv4SystemStatsInAddrErrors, legendFormat='IPv4:AddrError:In'),
        grafana.prometheus.target($.queries.ipv4SystemStatsInHdrErrors, legendFormat='IPv4:HdrError:In'),
        grafana.prometheus.target($.queries.ipv4SystemStatsInNoRoutes, legendFormat='IPv4:NoRoute:In'),
        grafana.prometheus.target($.queries.ipv4SystemStatsInTruncatedPkts, legendFormat='IPv4:Truncated:In'),
        grafana.prometheus.target($.queries.ipv4SystemStatsInDiscards, legendFormat='IPv4:Drop:In'),
        grafana.prometheus.target($.queries.ipv4SystemStatsOutDiscards, legendFormat='IPv4:Drop:Out'),

        grafana.prometheus.target($.queries.ipv6SystemStatsInAddrErrors, legendFormat='IPv6:AddrError:In'),
        grafana.prometheus.target($.queries.ipv6SystemStatsInHdrErrors, legendFormat='IPv6:HdrError:In'),
        grafana.prometheus.target($.queries.ipv6SystemStatsInNoRoutes, legendFormat='IPv6:NoRoute:In'),
        grafana.prometheus.target($.queries.ipv6SystemStatsInTruncatedPkts, legendFormat='IPv6:Truncated:In'),
        grafana.prometheus.target($.queries.ipv6SystemStatsInDiscards, legendFormat='IPv6:Drop:In'),
        grafana.prometheus.target($.queries.ipv6SystemStatsOutDiscards, legendFormat='IPv6:Drop:Out'),
      ]) + utils.timeSeriesOverride(
        unit='short',
        fillOpacity=10,
        showPoints='never',
      ) + outnegativey,
    ipNonForward:
      grafana.graphPanel.new(
        'IP Non-Forwarding',
        description='Number of IP datagrams by direction and IP version.',
        datasource='$datasource'
      )
      .addTargets([
        grafana.prometheus.target($.queries.ipv4SystemStatsHCInDelivers, legendFormat='IPv4:In'),
        grafana.prometheus.target($.queries.ipv4SystemStatsHCOutRequests, legendFormat='IPv4:Out'),

        grafana.prometheus.target($.queries.ipv6SystemStatsHCInDelivers, legendFormat='IPv6:In'),
        grafana.prometheus.target($.queries.ipv6SystemStatsHCOutRequests, legendFormat='IPv6:Out'),
      ]) + utils.timeSeriesOverride(
        unit='pps',
        fillOpacity=10,
        showPoints='never',
      ) + outnegativey,
    ipNonForwardDropErr:
      grafana.graphPanel.new(
        'IP Non-Forwarding Drops/Errors',
        description='Number of IP datagrams dropped, or in error, by reason, direction, and IP version.',
        datasource='$datasource'
      )
      .addTargets([
        grafana.prometheus.target($.queries.ipv4SystemStatsInUnknownProtos, legendFormat='IPv4:UnkwnProto:In'),
        grafana.prometheus.target($.queries.ipv4SystemStatsOutNoRoutes, legendFormat='IPv4:NoRoute:Out'),

        grafana.prometheus.target($.queries.ipv6SystemStatsInUnknownProtos, legendFormat='IPv6:UnknwnProto:In'),
        grafana.prometheus.target($.queries.ipv6SystemStatsOutNoRoutes, legendFormat='IPv6:NoRoute:Out'),
      ]) + utils.timeSeriesOverride(
        unit='pps',
        fillOpacity=10,
        showPoints='never',
      ) + outnegativey,
    tcpStateTransitions:
      grafana.graphPanel.new(
        'TCP State Transitions (/sec)',
        description='Number of TCP connections by type.',
        datasource='$datasource'
      )
      .addTargets([
        grafana.prometheus.target($.queries.tcpActiveOpens, legendFormat='ActiveOpen'),
        grafana.prometheus.target($.queries.tcpPassiveOpens, legendFormat='PassiveOpen'),
        grafana.prometheus.target($.queries.tcpAttemptFails, legendFormat='Fail'),
        grafana.prometheus.target($.queries.tcpEstabResets, legendFormat='Reset'),
      ]) + utils.timeSeriesOverride(
        unit='short',
        fillOpacity=10,
        showPoints='never',
      ),
    tcpSessions:
      grafana.graphPanel.new(
        'TCP Sessions',
        description='The number of TCP connections for which the current state is either ESTABLISHED or CLOSE-WAIT.',
        datasource='$datasource'
      )
      .addTarget(
        grafana.prometheus.target($.queries.tcpCurrEstab, legendFormat='Connections'),
      ) + utils.timeSeriesOverride(
        unit='short',
        fillOpacity=10,
        showPoints='never',
      ),
    tcpNonForwardSegs:
      grafana.graphPanel.new(
        'TCP Traffic, Non-Forward (seg/sec)',
        description='Total number of TCP segments by type and direction.',
        datasource='$datasource'
      )
      .addTargets([
        grafana.prometheus.target($.queries.tcpInSegs, legendFormat='In'),
        grafana.prometheus.target($.queries.tcpOutSegs, legendFormat='Out'),
        grafana.prometheus.target($.queries.tcpRetransSegs, legendFormat='Retrans:Out'),
        grafana.prometheus.target($.queries.tcpInErrs, legendFormat='Error:In'),
        grafana.prometheus.target($.queries.tcpOutRsts, legendFormat='Reset:Out'),
      ]) + utils.timeSeriesOverride(
        unit='pps',
        fillOpacity=10,
        showPoints='never',
      ) + outnegativey,
    udpNonForwardDgrams:
      grafana.graphPanel.new(
        'UDP Traffic, Non-Forward (dgram/sec)',
        description='Number of UDP datagrams sent and recieved, in error, or without destination application on port.',
        datasource='$datasource'
      )
      .addTargets([
        grafana.prometheus.target($.queries.udpInDatagrams, legendFormat='In'),
        grafana.prometheus.target($.queries.udpOutDatagrams, legendFormat='Out'),
        grafana.prometheus.target($.queries.udpInErrors, legendFormat='Error'),
        grafana.prometheus.target($.queries.udpNoPorts, legendFormat='NoPort'),
      ]) + utils.timeSeriesOverride(
        unit='short',
        fillOpacity=10,
        showPoints='never',
      ) + outnegativey,
    icmpSummary:
      grafana.graphPanel.new(
        'ICMP Summary',
        description='Number of ICMP messages and errors by direction and IP version.',
        datasource='$datasource'
      )
      .addTargets([
        grafana.prometheus.target($.queries.icmpIpv4StatsInMsgs, legendFormat='IPv4:Msg:In'),
        grafana.prometheus.target($.queries.icmpIpv4StatsOutMsgs, legendFormat='IPv4:Msg:Out'),
        grafana.prometheus.target($.queries.icmpIpv4StatsInErrors, legendFormat='IPv4:Error:In'),
        grafana.prometheus.target($.queries.icmpIpv4StatsOutErrors, legendFormat='IPv4:Error:Out'),

        grafana.prometheus.target($.queries.icmpIpv6StatsInMsgs, legendFormat='IPv6:Msg:In'),
        grafana.prometheus.target($.queries.icmpIpv6StatsOutMsgs, legendFormat='IPv6:Msg:Out'),
        grafana.prometheus.target($.queries.icmpIpv6StatsInErrors, legendFormat='IPv6:Error:In'),
        grafana.prometheus.target($.queries.icmpIpv6StatsOutErrors, legendFormat='IPv6:Error:Out'),
      ]) + utils.timeSeriesOverride(
        unit='pps',
        fillOpacity=10,
        showPoints='never',
      ) + outnegativey,
    icmpMsgTypes:
      grafana.graphPanel.new(
        'ICMP Message Type',
        description='Number of ICMP messages by type, direction, and IP version.',
        datasource='$datasource'
      )
      .addTargets([
        grafana.prometheus.target($.queries.icmpIpv4MsgStatsInPkts, legendFormat='IPv4:{{icmpMsgStatsType}}:In'),
        grafana.prometheus.target($.queries.icmpIpv4MsgStatsOutPkts, legendFormat='IPv4:{{icmpMsgStatsType}}:Out'),

        grafana.prometheus.target($.queries.icmpIpv6MsgStatsInPkts, legendFormat='IPv6:{{icmpMsgStatsType}}:In'),
        grafana.prometheus.target($.queries.icmpIpv6MsgStatsOutPkts, legendFormat='IPv6:{{icmpMsgStatsType}}:Out'),
      ]) + utils.timeSeriesOverride(
        unit='pps',
        fillOpacity=10,
        showPoints='never',
      ) + outnegativey,
    routingForwarding:
      grafana.graphPanel.new(
        'Routing/Forwarding',
        description='Number of IP routing discards, total routes, and forward entries.',
        datasource='$datasource'
      )
      .addTargets([
        grafana.prometheus.target($.queries.ipRoutingDiscards, legendFormat='Routes Discarded'),
        grafana.prometheus.target($.queries.inetCidrRouteNumber, legendFormat='Route Entries'),
        grafana.prometheus.target($.queries.ipForwardNumber, legendFormat='Forward Entries'),
      ]) + utils.timeSeriesOverride(
        unit='short',
        fillOpacity=10,
        showPoints='never',
      ),
    snmpMessages:
      grafana.graphPanel.new(
        'SNMP Messages (/sec)',
        description='Number of SNMP packets and requests by direction and type.',
        datasource='$datasource'
      )
      .addTargets([
        grafana.prometheus.target($.queries.snmpInPkts, legendFormat='Messages:In'),
        grafana.prometheus.target($.queries.snmpOutPkts, legendFormat='Messages:Out'),
        grafana.prometheus.target($.queries.snmpInGetNext, legendFormat='GetNext:In'),
        grafana.prometheus.target($.queries.snmpInGetRequests, legendFormat='GetReq:In'),
        grafana.prometheus.target($.queries.snmpInGetResponses, legendFormat='GetResp:In'),
        grafana.prometheus.target($.queries.snmpOutGetNext, legendFormat='GetNext:Out'),
        grafana.prometheus.target($.queries.snmpOutGetRequests, legendFormat='GetReq:Out'),
        grafana.prometheus.target($.queries.snmpOutGetResponses, legendFormat='GetResp:Out'),
      ]) + utils.timeSeriesOverride(
        unit='pps',
        fillOpacity=10,
        showPoints='never',
      ) + outnegativey,
    snmpTotalObjects:
      grafana.graphPanel.new(
        'SNMP Objects Fetched (/sec)',
        description='The total number of MIB objects which have been retrieved successfully by the SNMP protocol entity as the result of receiving valid SNMP Get-Request and Get-Next PDUs.',
        datasource='$datasource'
      )
      .addTarget(
        grafana.prometheus.target($.queries.snmpInTotalReqVars, legendFormat='Objects'),
      ) + utils.timeSeriesOverride(
        unit='short',
        fillOpacity=10,
        showPoints='never',
      ),
  },

  fixedColorMode(color):: {
    fieldConfig+: {
      defaults+: {
        color: {
          mode: 'fixed',
          fixedColor: color,
        },
      },
    },
  },

  grafanaDashboards+:: {
    'ubnt-edgrouterx-overview.json':
      dashboard.new(
        'Ubiquiti EdgeRouter Overview',
        time_from='now-1h',
        uid=std.md5('ubnt-edgrouterx-overview.json'),
      ).addTemplates([
        // Data Source
        {
          current: {
            text: 'default',
            value: 'default',
          },
          hide: 0,
          label: 'Data Source',
          name: 'datasource',
          options: [],
          query: 'prometheus',
          refresh: 1,
          regex: '',
          type: 'datasource',
        },
        // Job
        grafana.template.new(
          'job',
          '$datasource',
          'label_values(up{snmp_target!=""}, job)',
          label='job',
          refresh='load',
          multi=true,
          includeAll=true,
          allValues='.+',
          sort=1,
        ),
        // Instance
        grafana.template.new(
          'instance',
          '$datasource',
          'label_values(up{snmp_target!="", job=~"$job"}, instance)',
          label='instance',
          refresh='load',
          multi=true,
          includeAll=true,
          allValues='.+',
          sort=1,
        ),
        // SNMP Target
        grafana.template.new(
          'snmp_target',
          '$datasource',
          'label_values(up{snmp_target!="", job=~"$job", instance=~"$instance"}, snmp_target)',
          label='snmp_target',
          refresh='load',
          multi=true,
          includeAll=true,
          allValues='.+',
          sort=1,
        ),
        // Interface
        grafana.template.new(
          'interface',
          '$datasource',
          'ifNiceName{job=~"$job", instance=~"$instance", snmp_target=~"$snmp_target"}',
          label='interface',
          refresh='load',
          multi=true,
          includeAll=true,
          allValues='.+',
          sort=1,
          regex='/ifIndex=\\"(?<value>[0-9]+)\\".*nicename=\\"(?<text>[\\w:\\s0-9\\.]+)\\"/',
        ),
      ])
      .addRow(
        row.new('System Identification')
        .addPanel($.panels.infoTable { span: 12, height: 3 })
      )
      .addRow(
        row.new('Uptime & System')
        .addPanel($.panels.sysUptime)
        .addPanel($.panels.users)
        .addPanel($.panels.processes),
      )
      .addRow(
        row.new('CPU')
        .addPanel($.panels.cpuLoadAverage + legendDecoration(right=false))
        .addPanel($.panels.interruptsAndContexts + legendDecoration(right=false))
        .addPanel($.panels.blockIO + legendDecoration(right=false))
        .addPanel($.panels.cpuTime { span: 6 } + legendDecoration())
        .addPanel($.panels.processorLoad { span: 6 } + legendDecoration()),
      )
      .addRow(
        row.new('Memory')
        .addPanel($.panels.memDistribution { span: 12 } + legendDecoration())
        .addPanel($.panels.memAvail { span: 6 } + legendDecoration(right=false))
        .addPanel($.panels.memUtilized { span: 6 } + legendDecoration(right=false)),
      )
      .addRow(
        row.new('Interface')
        .addPanel($.panels.ifTraffic { span: 12 } + legendDecoration())
        .addPanel($.panels.ifTrafficDist { span: 12 } + legendDecoration())
        .addPanel($.panels.ifTrafficErrsDrops { span: 12 } + legendDecoration()),
      )
      .addRow(
        row.new('IP')
        .addPanel($.panels.ipDistribution { span: 12 } + legendDecoration())
        .addPanel($.panels.ipFrag { span: 12 } + legendDecoration())
        .addPanel($.panels.ipForward { span: 6 } + legendDecoration())
        .addPanel($.panels.ipForwardDropErr { span: 6 } + legendDecoration())
        .addPanel($.panels.ipNonForward { span: 6 } + legendDecoration())
        .addPanel($.panels.ipNonForwardDropErr { span: 6 } + legendDecoration()),
      )
      .addRow(
        row.new('TCP')
        .addPanel($.panels.tcpStateTransitions { span: 6 } + legendDecoration())
        .addPanel($.panels.tcpSessions { span: 6 } + legendDecoration(right=false))
        .addPanel($.panels.tcpNonForwardSegs { span: 12 } + legendDecoration()),
      )
      .addRow(
        row.new('UDP')
        .addPanel($.panels.udpNonForwardDgrams { span: 12 } + legendDecoration()),
      )
      .addRow(
        row.new('ICMP')
        .addPanel($.panels.icmpSummary { span: 12 } + legendDecoration())
        .addPanel($.panels.icmpMsgTypes { span: 12 } + legendDecoration()),
      )
      .addRow(
        row.new('Routing/Forwarding')
        .addPanel($.panels.routingForwarding { span: 12 } + legendDecoration(right=false)),
      )
      .addRow(
        row.new('SNMP')
        .addPanel($.panels.snmpMessages { span: 6 } + legendDecoration())
        .addPanel($.panels.snmpTotalObjects { span: 6 } + legendDecoration(right=false)),
      ) + {
        graphTooltip: 1,
      },
  },
}
