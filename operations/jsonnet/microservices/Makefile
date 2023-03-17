.PHONY: test
test:
	@cd test/ && \
		jb install && \
		jsonnet -J vendor -J lib main.libsonnet
