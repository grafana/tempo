.PHONY: tests

vendor jsonnetfile.lock.json: jsonnetfile.json
	jb install

tests: jsonnetfile.lock.json vendor
	jsonnet -J vendor/ test_*.libsonnet
