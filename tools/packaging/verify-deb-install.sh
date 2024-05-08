#!/usr/bin/env sh

set -euxo pipefail

docker ps
image="$(docker ps --filter ancestor=jrei/systemd-debian:12 --latest --format "{{.ID}}")"
echo "Running on container: ${image}"

dir="."
if [ -n "${CI}" ]; then
    dir="/drone/src"
fi
echo "Running on directory: ${dir}"

cat <<EOF | docker exec --interactive "${image}" sh
    set -x

    # Install tempo and check it's running
    dpkg -i ${dir}/dist/tempo*_amd64.deb
    [ "\$(systemctl is-active tempo)" = "active" ] || (echo "tempo is inactive" && exit 1)

    systemctl is-active tempo
    tail /var/log/dpkg.log
    journalctl -b

    # Wait for tempo to be ready. The script is cat-ed because it is passed to docker exec
    apt update && apt install -y curl
    $(cat ${dir}/tools/packaging/wait-for-ready.sh)
EOF