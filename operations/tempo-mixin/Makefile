.PHONY: all check install dashboards alerts rules

all: dashboards alerts rules

check: all
	git diff --exit-code -- yamls/

install:
	jb install

dashboards: install
	jsonnet -J vendor -S dashboards.jsonnet -m yamls

alerts: install
	jsonnet -J vendor -S alerts.jsonnet > yamls/alerts.yaml

rules: install
	jsonnet -J vendor -S rules.jsonnet > yamls/rules.yaml
