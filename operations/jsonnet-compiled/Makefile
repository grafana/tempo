
all: jb gen check

gen: jb gen-examples
gen-examples: gen-example/microservices gen-example/microservices-with-extras

jb:
	jb install ../jsonnet/microservices
	jb update ../jsonnet/microservices

jb-check:
	cp ../jsonnet/microservices/jsonnetfile.json .
	cp ../jsonnet/microservices/jsonnetfile.lock.json .
	git diff --exit-code -- jsonnetfile*.json

gen-example/%:
	jb update
	tk export out-$*/ $* --format "{{.kind}}-{{or .metadata.name .metadata.generateName}}"
	rm -rf $*/gen
	mkdir -p $*/gen
	cp out-$*/*.yaml $*/gen/
	rm -rf out-$*/

check: check-examples
check-examples: check-example/microservices

check-example/%: gen-example/%
	git diff --exit-code -- ../$*/gen/*.yaml

.PHONY: gen gen-examples gen-example/% check check-examples check-example/% jb jb-check
