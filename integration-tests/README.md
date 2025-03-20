# Cosmos OCR2 Integration Test

## Running Local tests

1. Build the core node image locally:

```shell
   cd chainlink
   docker buildx build -t local_chainlink -f core/chainlink.Dockerfile .
```

2. Build the Cosmos relayer:

```shell
   cd chainlink-cosmos
   make build-go
   docker buildx build --build-arg BASE_IMAGE=local_chainlink -t chainlink-cosmos -f ./Dockerfile .
```

3. Run the e2e test:

```shell
    cd chainlink-cosmos/integration-tests
    CORE_IMAGE=chainlink-cosmos DEFAULT_GAS_PRICE='0.025ucosm' MNEMONIC='surround miss nominee dream gap cross assault thank captain prosper drop duty group candy wealth weather scale put' NODE_URL='http://127.0.0.1:26657' TTL='1m' NODE_COUNT='4' go test --timeout=1h -v ./
```

4. To know if your test is successful, you should see the following logs:

```shell
INF Transmission Details: {Digest:0001a1a988f75da34469ae9e5bbef908ee33d1855c9ea9c835d751b4296705e4 Epoch:2 Round:1 LatestAnswer:+5 LatestTimestamp:2024-07-07 06:24:36 +0400 +04}
```
