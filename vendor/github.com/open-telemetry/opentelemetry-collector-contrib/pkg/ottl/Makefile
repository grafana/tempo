include ../../Makefile.Common

# benchmark-ottl runs all OTTL benchmarks across the module (the root package,
# ottlfuncs, and e2e). The shared `benchmark` target from Makefile.Common only
# covers the module's root package.
.PHONY: benchmark-ottl
benchmark-ottl:
	$(GOCMD) test -run='^$$' -bench=. -benchmem ./...

# update-ottl-signatures regenerates the golden file that locks the OTTL
# function signature surface (name, argument order, type, optionality). Run it
# after an intentional change to a function's Arguments struct, then commit the
# diff alongside a changelog entry.
.PHONY: update-ottl-signatures
update-ottl-signatures:
	OTTL_UPDATE_SIGNATURES=1 $(GOCMD) test ./ottlfuncs -run Test_FunctionSignatures
