.PHONY: test
test: jsonnet-test tanka-test

tanka-test:
	tk show test/environments/default

jsonnet-test:
	@cd test/ && \
		jb install && \
		jsonnet -J vendor -J lib environments/default/main.jsonnet
