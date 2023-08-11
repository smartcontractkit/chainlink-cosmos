#!/usr/bin/env bash

set -euo pipefail

cache_path="$(git rev-parse --show-toplevel)/.local-mock-server"
binary_name="dummy-external-adapter"
binary_path="${cache_path}/bin/${binary_name}"

bash "$(dirname -- "$0")/mock-adapter.down.sh"

if [ $# -gt 0 ]; then
	listen_address="$1"
elif [ "$(uname)" = "Darwin" ]; then
	echo "Listening on all interfaces on MacOS"
	listen_address="0.0.0.0:6060"
else
	docker_ip=$(docker network inspect bridge -f '{{range .IPAM.Config}}{{.Gateway}}{{end}}')
	if [ -z "${docker_ip}" ]; then
		echo "Could not fetch docker ip."
		exit 1
	fi
  echo "Listening on docker interface"
  listen_address="${docker_ip}:6060"
fi

echo "Listen address: ${listen_address}"

if [ ! -f "${binary_path}" ]; then
	echo "Installing mock-adapter"
	export GOPATH="${cache_path}"
	export GOBIN="${cache_path}/bin"
	go install 'github.com/smartcontractkit/dummy-external-adapter@latest'
fi

nohup "${binary_path}" "${listen_address}" &>/dev/null &
echo "Started mock-adapter (PID $!)"
