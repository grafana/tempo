#!/bin/sh -eu

if [ -n "${GITHUB_EMAIL+x}" ]; then
	git config --global user.email "${GITHUB_EMAIL}"
fi

if [ -n "${GITHUB_NAME+x}" ]; then
	git config --global user.name "${GITHUB_NAME}"
fi

exec "$@"
