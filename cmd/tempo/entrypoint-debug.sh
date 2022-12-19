#!/usr/bin/env bash

# entrypoint script used only in the debug image

DLV_OPTIONS="--listen=:2345 --headless=true --accept-multiclient --api-version=2"
if [ -z "${DEBUG_BLOCK}" ] || [ "${DEBUG_BLOCK}" = 0 ]; then
  echo "start delve in non blocking mode"
  DLV_OPTIONS="${DLV_OPTIONS} --continue"
fi
DLV_LOG_OPTIONS="--log=true --log-output=debugger,gdbwire,lldbout"

exec /dlv ${DLV_OPTIONS} ${DLV_LOG_OPTIONS} exec /tempo -- "$@"
