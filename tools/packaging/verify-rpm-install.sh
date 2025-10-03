#!/usr/bin/env bash

set -euxo pipefail

# Import the Grafana GPG key
rpm --import https://packages.grafana.com/gpg.key

# Install tempo and check it's running
rpm -i ./tempo_*_linux_amd64.rpm
[ "$(systemctl is-active tempo)" = "active" ] || (echo "tempo is inactive" && exit 1)

# Wait for tempo to be ready.
./wait-for-ready.sh
