set -euxo pipefail


docker run -d --name wasmd -e PASSWORD=xxxxxxxx -p 1317:1317 -p 26656:26656 -p 26657:26657 cosmwasm/wasmd:v0.29.2 ./setup_and_run.sh wasm1lsagfzrm4gz28he4wunt63sts5xzmczwda8vl6