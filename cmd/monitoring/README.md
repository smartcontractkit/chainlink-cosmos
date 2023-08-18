# On-chain monitor

## Example of running the monitor locally

```bash
COSMOS_TENDERMINT_URL="<some terra url>" \
COSMOS_NETWORK_NAME="cosmos-devnet" \
COSMOS_NETWORK_ID="cosmos-devnet" \
COSMOS_CHAIN_ID="1" \
COSMOS_READ_TIMEOUT="15s" \
COSMOS_POLL_INTERVAL="5s" \
COSMOS_LINK_TOKEN_ADDRESS="terra1eq0xqc88ceuvw2ztz2a08200he8lrgvnplrcst" \
KAFKA_BROKERS="localhost:29092" \
KAFKA_CLIENT_ID=“terra” \
KAFKA_SECURITY_PROTOCOL="PLAINTEXT" \
KAFKA_SASL_MECHANISM="PLAIN" \
KAFKA_SASL_USERNAME="" \
KAFKA_SASL_PASSWORD="" \
KAFKA_TRANSMISSION_TOPIC="transmission_topic" \
KAFKA_CONFIG_SET_SIMPLIFIED_TOPIC="config_set_simplified" \
SCHEMA_REGISTRY_URL="http://localhost:8989" \
SCHEMA_REGISTRY_USERNAME="" \
SCHEMA_REGISTRY_PASSWORD="" \
HTTP_ADDRESS="localhost:3000" \
FEEDS_URL="http://localhost:4000/feeds.json" \
go run ./cmd/monitoring/main.go
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
