local utils = import '../utils.libsonnet';
local test = import 'github.com/jsonnet-libs/testonnet/main.libsonnet';

local config = {
  prometheusAlerts: {
    groups: [
      {
        name: 'group1',
        rules: [
          {
            alert: 'alert1',
          },
          {
            alert: 'alert2',
          },
        ],
      },
      {
        name: 'group2',
        rules: [
          {
            alert: 'alert3',
          },
          {
            alert: 'alert4',
          },
        ],
      },
    ],
  },
  prometheusRules: {
    groups: [
      {
        name: 'group3',
        rules: [
          {
            record: 'record1',
          },
          {
            record: 'record2',
          },
        ],
      },
      {
        name: 'group4',
        rules: [
          {
            record: 'record3',
          },
          {
            record: 'record4',
          },
        ],
      },
    ],
  },
};

test.new(std.thisFile)

+ test.case.new(
  'removeAlertRuleGroup',
  test.expect.eq(
    actual=config + utils.removeAlertRuleGroup('group1'),
    expected={
      prometheusAlerts: {
        groups: [
          {
            name: 'group2',
            rules: [
              {
                alert: 'alert3',
              },
              {
                alert: 'alert4',
              },
            ],
          },
        ],
      },
      prometheusRules: {
        groups: [
          {
            name: 'group3',
            rules: [
              {
                record: 'record1',
              },
              {
                record: 'record2',
              },
            ],
          },
          {
            name: 'group4',
            rules: [
              {
                record: 'record3',
              },
              {
                record: 'record4',
              },
            ],
          },
        ],
      },
    }
  )
)

+ test.case.new(
  'removeRecordingRuleGroup',
  test.expect.eq(
    actual=config + utils.removeRecordingRuleGroup('group4'),
    expected={
      prometheusAlerts: {
        groups: [
          {
            name: 'group1',
            rules: [
              {
                alert: 'alert1',
              },
              {
                alert: 'alert2',
              },
            ],
          },
          {
            name: 'group2',
            rules: [
              {
                alert: 'alert3',
              },
              {
                alert: 'alert4',
              },
            ],
          },
        ],
      },
      prometheusRules: {
        groups: [
          {
            name: 'group3',
            rules: [
              {
                record: 'record1',
              },
              {
                record: 'record2',
              },
            ],
          },
        ],
      },
    }
  )
)

+ test.case.new(
  'removeAlertRuleGroup with groupname from recording rules (noop)',
  test.expect.eq(
    actual=config + utils.removeAlertRuleGroup('group4'),
    expected=config
  )
)
+ test.case.new(
  'removeRecordingRuleGroup with groupname from alert rules (noop)',
  test.expect.eq(
    actual=config + utils.removeRecordingRuleGroup('group2'),
    expected=config
  )
)

+ test.case.new(
  'removeAlerts',
  test.expect.eq(
    actual=config + utils.removeAlerts(['alert1', 'alert4']),
    expected={
      prometheusAlerts: {
        groups: [
          {
            name: 'group1',
            rules: [
              {
                alert: 'alert2',
              },
            ],
          },
          {
            name: 'group2',
            rules: [
              {
                alert: 'alert3',
              },
            ],
          },
        ],
      },
      prometheusRules: {
        groups: [
          {
            name: 'group3',
            rules: [
              {
                record: 'record1',
              },
              {
                record: 'record2',
              },
            ],
          },
          {
            name: 'group4',
            rules: [
              {
                record: 'record3',
              },
              {
                record: 'record4',
              },
            ],
          },
        ],
      },
    }
  )
)

+ test.case.new(
  'removeAlerts - object (backwards compatible)',
  test.expect.eq(
    actual=config + utils.removeAlerts({ alert1: {}, alert4: {} }),
    expected={
      prometheusAlerts: {
        groups: [
          {
            name: 'group1',
            rules: [
              {
                alert: 'alert2',
              },
            ],
          },
          {
            name: 'group2',
            rules: [
              {
                alert: 'alert3',
              },
            ],
          },
        ],
      },
      prometheusRules: {
        groups: [
          {
            name: 'group3',
            rules: [
              {
                record: 'record1',
              },
              {
                record: 'record2',
              },
            ],
          },
          {
            name: 'group4',
            rules: [
              {
                record: 'record3',
              },
              {
                record: 'record4',
              },
            ],
          },
        ],
      },
    }
  )
)
