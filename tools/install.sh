#! /bin/bash

# Receive an import, which may be a module path.  If we can list the module
# path, return it.  If we cannot, strip off the last path segment and try
# again.  Repeat until we find a module path.
function module_for_import() {
	local import=$1

	local module

	module=$(go list -m -f '{{.Path}}' "$import" 2>/dev/null)
	if [ "$module" ]; then
		echo "$module"
		return
	fi

	while [ ! "$module" ]; do
		import=$(echo "$import" | sed -e 's:/[^/]*$::')
		module=$(go list -m -f '{{.Path}}' "$import" 2>/dev/null)
		if [ "$module" ]; then
			echo "$module"
			return
		fi
	done

	return 1
}

declare -a imports

imports=($(go list -e -f '{{join .Imports " "}}' tools.go))
for i in "${imports[@]}"; do
	module=$(module_for_import "$i")

	# if we have a module, perform go list -m to get the module version.  Then
	# install the import at that version.
	if [ "$module" ]; then
		version=$(go list -m -f '{{.Version}}' "$module")
		if [ "$version" ]; then
			go install "$i@$version"
		else
			exit 1
		fi
	fi

done
