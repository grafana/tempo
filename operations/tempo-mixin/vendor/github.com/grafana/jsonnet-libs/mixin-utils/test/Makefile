.PHONY: tests

vendor jsonnetfile.lock.json: jsonnetfile.json
	jb install

tests: jsonnetfile.lock.json vendor
	jsonnet -J vendor/ ./test_native-classic-histogram.libsonnet
	jsonnet -J vendor/ ./test_remove_rules.libsonnet
