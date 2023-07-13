#!/usr/bin/env bash

container_name="chainlink-cosmos.wasmd"
container_version="v0.31.0"
genesis_account="wasm1lsagfzrm4gz28he4wunt63sts5xzmczwda8vl6"

set -euo pipefail

# Clean up first
bash "$(dirname -- "$0")/devnet-wasmd-down.sh"

echo "Starting wasmd container"

# we need to replace the entrypoint because starknet-devnet's docker builds at 0.5.1 don't include cargo or gcc.
docker run \
	-p 127.0.0.1:26657:26657 \
	-d \
	--name "${container_name}" \
	"cosmwasm/wasmd:${container_version}" \
	"./setup_and_run.sh" \
	"${genesis_account}"

echo "Waiting for wasmd container to become ready.."
start_time=$(date +%s)
prev_output=""
while true; do
	output=$(docker logs "${container_name}" 2>&1)
	if [[ "${output}" != "${prev_output}" ]]; then
		echo -n "${output#$prev_output}"
		prev_output="${output}"
	fi

	if [[ $output == *"Replay: Done"* ]]; then
		echo ""
		echo "wasmd is ready."
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
