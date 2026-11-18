// Sharding efficiency comparison: balanced (configmap_binpack:false) vs
// binpack (configmap_binpack:true), across several dashboard-size
// distributions and shard budgets. Emits a Markdown table for the PR
// description. Run: jsonnet -S -J vendor/ ./efficiency.libsonnet
local f = import 'sharding_fixture.libsonnet';

local budgets = [f.mib, 900000, 700000, 500000];

// Non-negative integer with thousands separators.
local comma(n) =
  local s = std.toString(n);
  local L = std.length(s);
  std.join('', [
    s[i] + (if (L - i - 1) > 0 && (L - i - 1) % 3 == 0 then ',' else '')
    for i in std.range(0, L - 1)
  ]);

local row(label, m) =
  local over = if m.over_1mib == 0 then '0' else '**%d** ⚠️' % m.over_1mib;
  '| %s | %d | %s | %s | %.1f%% | %.1f%% | %.2f |' % [
    label,
    m.shards,
    over,
    comma(m.max_shard),
    m.max_pct,
    m.avg_fill_pct,
    m.shards_vs_min,
  ];

local scenarioBlock(name) =
  local sizes = f.scenarios[name];
  local total = std.foldl(function(a, b) a + b, sizes, 0);
  local budgetRows(budget) = [
    row('**%s** · balanced' % comma(budget), f.metrics(sizes, false, budget)),
    row('· binpack', f.metrics(sizes, true, budget)),
  ];
  [
    '### %s' % name,
    '_%d dashboards, %s bytes total (theoretical min at 1 MiB: %d shards)_' % [
      std.length(sizes),
      comma(total),
      std.ceil(total / f.mib),
    ],
    '',
    '| budget / strategy | shards | over 1 MiB | max shard | max % | avg fill % | shards/min |',
    '|-------------------|:------:|:----------:|----------:|:-----:|:----------:|:----------:|',
  ]
  + std.flattenArrays([budgetRows(b) for b in budgets])
  + [''];

std.join('\n', [
                 '## Dashboard ConfigMap sharding: balanced vs binpack',
                 '',
                 'Hard Kubernetes ConfigMap limit: **1,048,576 bytes (1 MiB)**. `over 1 MiB` must',
                 'be 0 for correctness. `shards/min` = shards used vs the theoretical minimum',
                 '(1.00 = optimal packing; lower shard counts at equal correctness are better).',
                 'Dashboards are test strings of the listed byte lengths.',
                 '',
               ]
               + std.flattenArrays([scenarioBlock(n) for n in std.objectFields(f.scenarios)]))
