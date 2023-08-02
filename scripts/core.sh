#!/usr/bin/env bash

set -euo pipefail

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

echo "Starting core container"
exec docker run \
	-it --rm \
	--add-host=host.docker.internal:host-gateway \
	-p 127.0.0.1:6688:6688 \
	-p 127.0.0.1:6690:6690 \
	-e "CL_CONFIG=$(cat "${config_path}")" \
	-e 'CL_DATABASE_URL=postgresql://postgres:postgres@host.docker.internal:35432/cosmos_test?sslmode=disable' \
	-e 'CL_DATABASE_ALLOW_SIMPLE_PASSWORDS=true' \
	-e 'CL_PASSWORD_KEYSTORE=asdfasdfasdfasdf' \
	--name "${container_name}" \
	--entrypoint bash \
	"smartcontract/chainlink:${container_version}" \
	-c \
	"echo -e \"${api_email}\\n${api_password}\" > /tmp/api_credentials && chainlink node start --api /tmp/api_credentials"

#echo "Waiting for wasmd container to become ready.."
#start_time=$(date +%s)
#prev_output=""
#while true; do
#output=$(docker logs "${container_name}" 2>&1)
#if [[ "${output}" != "${prev_output}" ]]; then
#echo -n "${output#$prev_output}"
#prev_output="${output}"
#fi

#if [[ $output == *"Replay: Done"* ]]; then
#echo ""
#echo "wasmd is ready."
#exit 0
#fi

#current_time=$(date +%s)
#elapsed_time=$((current_time - start_time))

#if ((elapsed_time > 600)); then
#echo "Error: Command did not become ready within 600 seconds"
#exit 1
#fi

#sleep 3
#done
