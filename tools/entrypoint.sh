#!/bin/sh -eu

if [ -n "${GITHUB_EMAIL+x}" ]; then
	git config --global user.email "${GITHUB_EMAIL}"
fi

if [ -n "${GITHUB_NAME+x}" ]; then
	git config --global user.name "${GITHUB_NAME}"
fi

# Only add safe.directory if not already present to avoid duplicates
if ! git config --global --get-all safe.directory | grep -q "^/tools$"; then
	git config --global --add safe.directory /tools
fi

exec "$@"
