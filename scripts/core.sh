#!/usr/bin/env bash

set -euo pipefail

bash "$(dirname -- "$0")/core.down.sh"

container_name="chainlink-cosmos.core"
container_version="2.3.0"

# https://github.com/smartcontractkit/chainlink/blob/600365a7a27508d699dbd4b994fafba7dc288659/integration-tests/client/chainlink_k8s.go#L82-L83
api_email="notreal@fakeemail.ch"
api_password="fj293fbBnlQ!f9vNs"

if [ $# -lt 1 ]; then
	echo "No config path" >&2
	exit 1
fi

config_path="$1"
if [ ! -f "${config_path}" ]; then
	echo "Config path not found." >&2
	exit 1
fi

docker_ip=$(docker network inspect bridge -f '{{range .IPAM.Config}}{{.Gateway}}{{end}}')
if [ -z "${docker_ip}" ]; then
	echo "Could not fetch docker ip."
	exit 1
fi

declare -i postgres_base_port=5432
declare -i core_base_port=6688
declare -i core_p2p_base_port=6695

for i in {1..4}; do
	echo "Starting core container $i"
	docker run \
		-it --rm \
		--add-host=host.docker.internal:host-gateway \
		--platform linux/amd64 \
		-d \
		-p 127.0.0.1:$(($core_base_port + $i - 1)):$core_base_port \
		-p 127.0.0.1:$(($core_p2p_base_port + $i - 1)):$core_p2p_base_port \
		-p "${docker_ip}:$(($core_base_port + $i - 1)):$core_base_port" \
		-p "${docker_ip}:$(($core_p2p_base_port + $i - 1)):$core_p2p_base_port" \
		-e "CL_CONFIG=$(cat "${config_path}")" \
		-e "CL_DATABASE_URL=postgresql://postgres:postgres@host.docker.internal:$(($postgres_base_port + $i - 1))/cosmos_test?sslmode=disable" \
		-e 'CL_DATABASE_ALLOW_SIMPLE_PASSWORDS=true' \
		-e 'CL_PASSWORD_KEYSTORE=asdfasdfasdfasdf' \
		--name "${container_name}.$i" \
		--entrypoint bash \
		"smartcontract/chainlink:${container_version}" \
		-c \
		"echo -e \"${api_email}\\n${api_password}\" > /tmp/api_credentials && chainlink node start --api /tmp/api_credentials"

	echo "Waiting for core container to become ready.."
	start_time=$(date +%s)
	prev_output=""
	while true; do
		output=$(docker logs "${container_name}.$i" 2>&1)
		if [[ "${output}" != "${prev_output}" ]]; then
			echo -n "${output#$prev_output}"
			prev_output="${output}"
		fi

		if [[ $output == *"Listening and serving HTTP"* ]]; then
			echo ""
			echo "node is ready."
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
