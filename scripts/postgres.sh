#!/usr/bin/env bash

container_name="chainlink-cosmos.postgres"
container_version="15.2-alpine"

set -euo pipefail

bash "$(dirname -- "$0")/postgres.down.sh"

docker_ip=$(docker network inspect bridge -f '{{range .IPAM.Config}}{{.Gateway}}{{end}}')
if [ -z "${docker_ip}" ]; then
	echo "Could not fetch docker ip."
	exit 1
fi

echo "Starting postgres container"
docker run \
	-p "127.0.0.1:35432:5432" \
	-p "${docker_ip}:35432:5432" \
	-d \
	--name "${container_name}" \
	-e POSTGRES_USER=postgres \
	-e POSTGRES_PASSWORD=postgres \
	-e POSTGRES_DB=cosmos_test \
	"postgres:${container_version}" \
	-c 'listen_addresses=*'
