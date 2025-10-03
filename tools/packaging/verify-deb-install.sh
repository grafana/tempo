#!/usr/bin/env bash

set -euxo pipefail

# Install tempo and check it's running
dpkg -i ./tempo_*_linux_amd64.deb
[ "$(systemctl is-active tempo)" = "active" ] || (echo "tempo is inactive" && exit 1)

# Wait for tempo to be ready.
apt update && apt install -y curl
./wait-for-ready.sh
