.PHONY: tests efficiency

# `stubs/` supplies a minimal ksonnet-util/kausal so the sharding code path
# resolves without the full k8s-libsonnet tree; it takes precedence over vendor.
JPATH := -J stubs/ -J vendor/

vendor jsonnetfile.lock.json: jsonnetfile.json
	jb install

# Correctness assertions (testonnet).
tests: jsonnetfile.lock.json vendor
	jsonnet $(JPATH) ./test_sharding.libsonnet

# Efficiency comparison table (Markdown, for the PR description).
efficiency: jsonnetfile.lock.json vendor
	@jsonnet -S $(JPATH) ./efficiency.libsonnet
