.PHONY: test
test: jsonnet-compile jsonnet-test tanka-test

tanka-test:
	tk show test/environments/default

jsonnet-compile:
	@cd test/ && \
		mkdir -p outputs && \
		jb install && \
		jsonnet -J vendor -J lib -o outputs/base.json environments/default/main.jsonnet

jsonnet-test: jsonnet-compile
	@cd test/ && \
		jsonnet -J vendor -J lib test1.jsonnet

