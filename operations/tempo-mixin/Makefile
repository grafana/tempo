.PHONY: all check install dashboards alerts rules

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
