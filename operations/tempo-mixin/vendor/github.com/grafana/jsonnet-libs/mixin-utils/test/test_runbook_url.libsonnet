local utils = import '../utils.libsonnet';
local test = import 'github.com/jsonnet-libs/testonnet/main.libsonnet';

local groups = [
  {
    name: 'group1',
    rules: [
      {
        record: 'record1',
      },
      {
        alert: 'MimirIngesterUnhealthy',
        annotations: {
          message: 'An ingester is unhealthy.',
        },
      },
    ],
  },
  {
    name: 'group2',
    rules: [
      {
        alert: 'MemcachedDown',
        annotations: {
          runbook_url: 'https://example.com/stale',
        },
      },
    ],
  },
];

test.new(std.thisFile)

+ test.case.new(
  'withRunbookURL keeps the alert name as-is by default',
  test.expect.eq(
    actual=utils.withRunbookURL('https://example.com/runbooks.md#%s', groups),
    expected=[
      {
        name: 'group1',
        rules: [
          {
            record: 'record1',
          },
          {
            alert: 'MimirIngesterUnhealthy',
            annotations: {
              message: 'An ingester is unhealthy.',
              runbook_url: 'https://example.com/runbooks.md#MimirIngesterUnhealthy',
            },
          },
        ],
      },
      {
        name: 'group2',
        rules: [
          {
            alert: 'MemcachedDown',
            annotations: {
              runbook_url: 'https://example.com/runbooks.md#MemcachedDown',
            },
          },
        ],
      },
    ]
  )
)

+ test.case.new(
  'withRunbookURL lowercases the alert name when asked',
  test.expect.eq(
    actual=utils.withRunbookURL('https://example.com/runbooks.md#%s', groups, lowercase=true),
    expected=[
      {
        name: 'group1',
        rules: [
          {
            record: 'record1',
          },
          {
            alert: 'MimirIngesterUnhealthy',
            annotations: {
              message: 'An ingester is unhealthy.',
              runbook_url: 'https://example.com/runbooks.md#mimiringesterunhealthy',
            },
          },
        ],
      },
      {
        name: 'group2',
        rules: [
          {
            alert: 'MemcachedDown',
            annotations: {
              runbook_url: 'https://example.com/runbooks.md#memcacheddown',
            },
          },
        ],
      },
    ]
  )
)

+ test.case.new(
  'withRunbookURL honours annotation_key together with lowercase',
  test.expect.eq(
    actual=utils.withRunbookURL(
      'https://example.com/runbooks.md#%s',
      [groups[1]],
      annotation_key='runbook_url_internal',
      lowercase=true,
    ),
    expected=[
      {
        name: 'group2',
        rules: [
          {
            alert: 'MemcachedDown',
            annotations: {
              runbook_url: 'https://example.com/stale',
              runbook_url_internal: 'https://example.com/runbooks.md#memcacheddown',
            },
          },
        ],
      },
    ]
  )
)
