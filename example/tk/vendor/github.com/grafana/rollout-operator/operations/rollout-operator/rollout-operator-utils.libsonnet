{
  newRolloutOperatorNodeAffinityMatchers(nodeAffinityMatchers)::
    local deployment = $.apps.v1.deployment;

    if std.length(nodeAffinityMatchers) == 0 then {} else (
      deployment.spec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.withNodeSelectorTerms(
        local sorted = std.sort(nodeAffinityMatchers, function(x) x.key);
        $.core.v1.nodeSelectorTerm.withMatchExpressions(sorted)
      )
    ),
}
