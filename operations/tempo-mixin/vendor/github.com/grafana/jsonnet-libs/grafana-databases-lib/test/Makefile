.PHONY: tests

vendor jsonnetfile.lock.json: jsonnetfile.json
	jb install

tests: jsonnetfile.lock.json vendor
	jsonnet -J vendor/ ./test_autoscaling.libsonnet
	jsonnet -J vendor/ ./test_legacy.libsonnet
	jsonnet -J vendor/ ./test_dashboard.libsonnet
	jsonnet -J vendor/ ./dashboard.jsonnet > /dev/null
