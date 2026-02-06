
all: jb gen check

jb:
	jb install ../jsonnet/microservices
	jb update ../jsonnet/microservices

jb-check:
	cp ../jsonnet/microservices/jsonnetfile.json .
	cp ../jsonnet/microservices/jsonnetfile.lock.json .
	git diff --exit-code -- jsonnetfile*.json

gen: jb gen-examples
gen-examples: gen-example/microservices gen-example/microservices-with-extras

gen-example/%:
	jb update
	tk export out-$*/ $* --format "{{.kind}}-{{or .metadata.name .metadata.generateName}}"
	rm -rf $*/gen
	mkdir -p $*/gen
	cp out-$*/*.yaml $*/gen/
	rm -rf out-$*/

check: check-examples
check-examples: check-example/microservices check-example/microservices-with-extras

check-example/%: gen-example/%
	@# Check for modified tracked files
	git diff --exit-code -- $*/gen
	@# Check for untracked files (new files not yet in git)
	@if [ -n "$$(git ls-files --others --exclude-standard $*/gen)" ]; then \
		echo "Error: Untracked files found in $*/gen:"; \
		git ls-files --others --exclude-standard $*/gen; \
		exit 1; \
	fi

.PHONY: gen gen-examples gen-example/% check check-examples check-example/% jb jb-check
