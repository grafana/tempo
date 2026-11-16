.PHONY: all check install dashboards alerts rules fmt

all: install dashboards alerts rules

check: all
	git diff --exit-code -- ../tempo-mixin-compiled/
	@out=$$(jsonnet -J vendor -S cluster-label-check.jsonnet) && \
		if echo "$$out" | grep -nw cluster; then \
			echo "Hard-coded cluster label above: use \$$._config.per_cluster_label or group_by_cluster."; exit 1; \
		fi

install:
	jb install

dashboards:
	jsonnet -J vendor -S dashboards.jsonnet -m ../tempo-mixin-compiled/dashboards/

alerts:
	jsonnet -J vendor -S alerts.jsonnet > ../tempo-mixin-compiled/alerts.yaml

rules:
	jsonnet -J vendor -S rules.jsonnet > ../tempo-mixin-compiled/rules.yaml

fmt:
	find . -name 'vendor' -prune -o -name '*.libsonnet' -print -o -name '*.jsonnet' -print | \
		xargs -n 1 -- jsonnetfmt -i
