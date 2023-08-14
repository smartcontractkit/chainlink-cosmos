#!/usr/bin/env bash

echo "Tearing down integration test environment.."

bash ./core.down.sh

bash ./mock-adapter.down.sh

bash ./postgres.down.sh

bash ./wasmd.down.sh

echo "Teardown finished."