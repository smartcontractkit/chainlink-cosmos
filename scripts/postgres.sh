#!/usr/bin/env bash

container_name="chainlink-cosmos.postgres"
container_version="15.2-alpine"

set -euo pipefail

bash "$(dirname -- "$0")/postgres.down.sh"

docker_ip=""
if [ "$(uname)" != "Darwin" ]; then
	docker_ip=$(docker network inspect bridge -f '{{range .IPAM.Config}}{{.Gateway}}{{end}}')
	if [ -z "${docker_ip}" ]; then
		echo "Could not fetch docker ip."
		exit 1
	fi
	echo "Docker IP: ${docker_ip}"
else
	echo "Listening on all interfaces on MacOS"
fi

echo "Starting postgres container"
docker run \
	-p "${docker_ip}:5432:5432" \
	-d \
	--name "${container_name}" \
	-e POSTGRES_USER=postgres \
	-e POSTGRES_PASSWORD=postgres \
	-e POSTGRES_DB=cosmos_test \
	"postgres:${container_version}" \
	-c 'listen_addresses=*'

echo "Waiting for postgres container to become ready.."
start_time=$(date +%s)
prev_output=""
while true; do
	output=$(docker logs "${container_name}" 2>&1)
	if [[ "${output}" != "${prev_output}" ]]; then
		echo -n "${output#$prev_output}"
		prev_output="${output}"
	fi

	if [[ $output == *"listening on IPv4 address"* ]]; then
		echo ""
		echo "postgres is ready."
		exit 0
	fi

	current_time=$(date +%s)
	elapsed_time=$((current_time - start_time))

	if ((elapsed_time > 600)); then
		echo "Error: Command did not become ready within 600 seconds"
		exit 1
	fi

	sleep 3
done
