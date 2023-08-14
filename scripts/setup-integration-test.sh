#!/usr/bin/env bash

echo "Setting up integration test environment.."

core_image=""
if [ -n "${CORE_IMAGE:-}" ]; then
	core_image="CORE_IMAGE=${CORE_IMAGE}"
fi

bash ./wasmd.sh

bash ./postgres.sh

bash ./mock-adapter.sh

bash ./core.sh "${core_image}"

echo "Setup finished."