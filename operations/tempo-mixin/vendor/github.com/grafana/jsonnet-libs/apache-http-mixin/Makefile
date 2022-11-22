JSONNET_FMT := jsonnetfmt -n 2 --max-blank-lines 1 --string-style s --comment-style s

.PHONY: all
all: build dashboards_out prometheus_alerts.yaml

vendor: jsonnetfile.json
	jb install

.PHONY: build
build: vendor

.PHONY: fmt
fmt:
	find . -name 'vendor' -prune -o -name '*.libsonnet' -print -o -name '*.jsonnet' -print | \
		xargs -n 1 -- $(JSONNET_FMT) -i

.PHONY: lint
lint: build
	find . -name 'vendor' -prune -o -name '*.libsonnet' -print -o -name '*.jsonnet' -print | \
		while read f; do \
			$(JSONNET_FMT) "$$f" | diff -u "$$f" -; \
		done
	mixtool lint mixin.libsonnet

dashboards_out: mixin.libsonnet config.libsonnet $(wildcard dashboards/*)
	@mkdir -p dashboards_out
	mixtool generate dashboards mixin.libsonnet -d dashboards_out

prometheus_alerts.yaml: mixin.libsonnet alerts/*.libsonnet
	mixtool generate alerts mixin.libsonnet -a prometheus_alerts.yaml


.PHONY: deploy deploy_rules deploy_dashboards
deploy: deploy_rules deploy_dashboards

deploy_dashboards:
ifdef GRAFANA_URL
	grr apply mixin.libsonnet --target "Dashboard.*" --target "DashboardFolder.*"
else
	$(warning GRAFANA_URL is not set, skipping grafana dashboards deployment)
endif

deploy_rules:
ifdef CORTEX_ADDRESS
	grr apply mixin.libsonnet --target "PrometheusRuleGroup.*"
else
	$(warning CORTEX_ADDRESS is not set, skipping prometheus alerts deployment)
endif

.PHONY: clean
clean:
	rm -rf dashboards_out prometheus_alerts.yaml
