# On-chain monitor

## Local development

- Start the monitor's third party dependencies using [docker-compose](https://docs.docker.com/compose/).
  Use the docker-compose.yml file in `./ops`:

```sh
docker-compose up
```

- Start an http server that mimics weiwatchers locally. Note: this isn't required to run integration tests as it these are created automatically in the test. It needs to export a json configuration file for feeds:

```json
[
  {
    "name": "LINK / USD",
    "path": "link-usd",
    "symbol": "$",
    "heartbeat": 0,
    "contract_type": "numerical_median_feed",
    "status": "testing",
    "contract_address": "<CONTRACT_ADDRESS>",
    "multiply": "100000000",
    "proxy_address": "<PROXY_ADDRESS>"
  }
]
```

It also needs to export a json configuration for for node operators:

```json
[
  {
    "id": "noop",
    "nodeAddress": [<NODE_OPERATOR_ADDRESS>]
  }
]
```

One option is to create a folder `/tmp/configs` and add two files `feeds.json` and `nodes.json` with the configs from above, then:

```bash
python3 -m http.server 4000
```

- Start the monitor locally. You will need the tendermint url and the address of the LINK token.

```bash
COSMOS_TENDERMINT_URL="<tendermint url>" \
COSMOS_NETWORK_NAME="cosmos-devnet" \
COSMOS_NETWORK_ID="cosmos-devnet" \
COSMOS_CHAIN_ID="1" \
COSMOS_READ_TIMEOUT="15s" \
COSMOS_POLL_INTERVAL="5s" \
COSMOS_LINK_TOKEN_ADDRESS="wasm12fykm2xhg5ces2vmf4q2aem8c958exv3v0wmvrspa8zucrdwjedsjax9ms" \
COSMOS_BECH32_PREFIX="wasm" \
COSMOS_GAS_TOKEN="ucosm" \
KAFKA_BROKERS="localhost:29092" \
KAFKA_CLIENT_ID="cosmos" \
KAFKA_SECURITY_PROTOCOL="SASL_PLAINTEXT" \
KAFKA_SASL_MECHANISM="PLAIN" \
KAFKA_SASL_USERNAME="user" \
KAFKA_SASL_PASSWORD="pass" \
KAFKA_TRANSMISSION_TOPIC="transmission_topic" \
KAFKA_CONFIG_SET_SIMPLIFIED_TOPIC="config_set_simplified" \
SCHEMA_REGISTRY_URL="http://localhost:8989" \
SCHEMA_REGISTRY_USERNAME="" \
SCHEMA_REGISTRY_PASSWORD="" \
HTTP_ADDRESS="localhost:3000" \
FEEDS_URL="http://localhost:4000/feeds.json" \
go run ./cmd/monitoring/main.go
```

- Check the output for the Prometheus scraper

```bash
curl http://localhost:3000/metrics
```

- To check the output for Kafka, you need to install [kcat](https://github.com/edenhill/kcat). After you install, run:

```bash
kcat -b localhost:29092 -t config_set_simplified
kcat -b localhost:29092 -t transmission_topic
```

## Example of feed configurations returned by weiwatchers.com

```json
[
  {
    "name": "UST/USD",
    "path": "ust-usd",
    "symbol": "$",
    "heartbeat": 50,
    "contract_type": "",
    "status": "live",
    "contract_address_bech32": "terra1dre5vgujqex83kc4kw3jr6fc6z8erdsqxlsvhk" ,
    "multiply": "100000000"
  }
]
```
