.PHONY: all check install dashboards alerts rules fmt

all: install dashboards alerts rules

check: all
	git diff --exit-code -- ../tempo-mixin-compiled/

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
