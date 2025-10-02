#!/usr/bin/env sh

status="$(curl -s http://localhost:3200/ready)"
i=0
while [ "${status}" != "ready" ]; do
  if [ "${i}" -gt "15" ]; then
    echo "tempo never became ready"
    exit 1
  fi
  sleep 2
  status="$(curl -s http://localhost:3200/ready)"
  i=$((i+1))
done