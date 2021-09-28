.PHONY: gen check

gen:
	jb update
	tk export out/ example --format "{{.kind}}-{{or .metadata.name .metadata.generateName}}"

	cp out/*.yaml ../
	rm -r out/

check: gen
	git diff --exit-code -- ../
