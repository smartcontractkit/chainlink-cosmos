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

declare -i base_port=5432
for i in {1..4}; do
	echo "Starting postgres container $i on port $(($base_port + $i - 1))"
	docker run \
		-p 127.0.0.1:$(($base_port + $i - 1)):$base_port \
		-p "${docker_ip}:$(($base_port + $i - 1)):$base_port" \
		-d \
		--name "${container_name}.$i" \
		-e POSTGRES_USER=postgres \
		-e POSTGRES_PASSWORD=postgres \
		-e POSTGRES_DB=cosmos_test \
		"postgres:${container_version}" \
		-c 'listen_addresses=*'

	echo "Waiting for postgres container to become ready.."
	start_time=$(date +%s)
	prev_output=""
	while true; do
		output=$(docker logs "${container_name}.$i" 2>&1)
		if [[ "${output}" != "${prev_output}" ]]; then
			echo -n "${output#$prev_output}"
			prev_output="${output}"
		fi

		if [[ $output == *"listening on IPv4 address"* ]]; then
			echo ""
			echo "postgres is ready."
			break
		fi

		current_time=$(date +%s)
		elapsed_time=$((current_time - start_time))

		if ((elapsed_time > 600)); then
			echo "Error: Command did not become ready within 600 seconds"
			exit 1
		fi

		sleep 3
	done
done
