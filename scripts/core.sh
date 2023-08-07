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

echo "Starting core container"
exec docker run \
	-it --rm \
	--add-host=host.docker.internal:host-gateway \
	-p 127.0.0.1:6688:6688 \
	-p 127.0.0.1:6690:6690 \
	-p "${docker_ip}:6688:6688" \
	-p "${docker_ip}:6690:6690" \
	-e "CL_CONFIG=$(cat "${config_path}")" \
	-e 'CL_DATABASE_URL=postgresql://postgres:postgres@host.docker.internal:35432/cosmos_test?sslmode=disable' \
	-e 'CL_DATABASE_ALLOW_SIMPLE_PASSWORDS=true' \
	-e 'CL_PASSWORD_KEYSTORE=asdfasdfasdfasdf' \
	--name "${container_name}" \
	--entrypoint bash \
	"smartcontract/chainlink:${container_version}" \
	-c \
	"echo -e \"${api_email}\\n${api_password}\" > /tmp/api_credentials && chainlink node start --api /tmp/api_credentials"
