.PHONY: all check install dashboards alerts rules fmt

all: install dashboards alerts rules

check: all
	git diff --exit-code -- yamls/

install:
	jb install

dashboards:
	jsonnet -J vendor -S dashboards.jsonnet -m yamls

alerts:
	jsonnet -J vendor -S alerts.jsonnet > yamls/alerts.yaml

rules:
	jsonnet -J vendor -S rules.jsonnet > yamls/rules.yaml

fmt:
	find . -name 'vendor' -prune -o -name '*.libsonnet' -print -o -name '*.jsonnet' -print | \
		xargs -n 1 -- jsonnetfmt -i
