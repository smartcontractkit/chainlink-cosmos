# smartcontractkit Go modules
## Main module
```mermaid
flowchart LR

	chain-selectors
	click chain-selectors href "https://github.com/smartcontractkit/chain-selectors"
	chainlink-common --> chainlink-common/pkg/chipingress
	chainlink-common --> chainlink-protos/billing/go
	chainlink-common --> chainlink-protos/cre/go
	chainlink-common --> chainlink-protos/linking-service/go
	chainlink-common --> chainlink-protos/node-platform
	chainlink-common --> chainlink-protos/storage-service
	chainlink-common --> chainlink-protos/workflows/go
	chainlink-common --> freeport
	chainlink-common --> grpc-proxy
	chainlink-common --> libocr
	click chainlink-common href "https://github.com/smartcontractkit/chainlink-common"
	chainlink-common/pkg/chipingress
	click chainlink-common/pkg/chipingress href "https://github.com/smartcontractkit/chainlink-common"
	chainlink-common/pkg/monitoring --> chainlink-common
	chainlink-common/pkg/monitoring --> chainlink-common/pkg/values
	click chainlink-common/pkg/monitoring href "https://github.com/smartcontractkit/chainlink-common"
	chainlink-common/pkg/values
	click chainlink-common/pkg/values href "https://github.com/smartcontractkit/chainlink-common"
	chainlink-cosmos --> chainlink-common/pkg/monitoring
	chainlink-cosmos --> chainlink-framework/chains
	click chainlink-cosmos href "https://github.com/smartcontractkit/chainlink-cosmos"
	chainlink-framework/chains --> chainlink-common
	chainlink-framework/chains --> chainlink-framework/multinode
	click chainlink-framework/chains href "https://github.com/smartcontractkit/chainlink-framework"
	chainlink-framework/multinode
	click chainlink-framework/multinode href "https://github.com/smartcontractkit/chainlink-framework"
	chainlink-protos/billing/go
	click chainlink-protos/billing/go href "https://github.com/smartcontractkit/chainlink-protos"
	chainlink-protos/cre/go --> chain-selectors
	click chainlink-protos/cre/go href "https://github.com/smartcontractkit/chainlink-protos"
	chainlink-protos/linking-service/go
	click chainlink-protos/linking-service/go href "https://github.com/smartcontractkit/chainlink-protos"
	chainlink-protos/node-platform
	click chainlink-protos/node-platform href "https://github.com/smartcontractkit/chainlink-protos"
	chainlink-protos/storage-service
	click chainlink-protos/storage-service href "https://github.com/smartcontractkit/chainlink-protos"
	chainlink-protos/workflows/go
	click chainlink-protos/workflows/go href "https://github.com/smartcontractkit/chainlink-protos"
	freeport
	click freeport href "https://github.com/smartcontractkit/freeport"
	go-sumtype2
	click go-sumtype2 href "https://github.com/smartcontractkit/go-sumtype2"
	grpc-proxy
	click grpc-proxy href "https://github.com/smartcontractkit/grpc-proxy"
	libocr --> go-sumtype2
	click libocr href "https://github.com/smartcontractkit/libocr"

	subgraph chainlink-common-repo[chainlink-common]
		 chainlink-common
		 chainlink-common/pkg/chipingress
		 chainlink-common/pkg/monitoring
		 chainlink-common/pkg/values
	end
	click chainlink-common-repo href "https://github.com/smartcontractkit/chainlink-common"

	subgraph chainlink-framework-repo[chainlink-framework]
		 chainlink-framework/chains
		 chainlink-framework/multinode
	end
	click chainlink-framework-repo href "https://github.com/smartcontractkit/chainlink-framework"

	subgraph chainlink-protos-repo[chainlink-protos]
		 chainlink-protos/billing/go
		 chainlink-protos/cre/go
		 chainlink-protos/linking-service/go
		 chainlink-protos/node-platform
		 chainlink-protos/storage-service
		 chainlink-protos/workflows/go
	end
	click chainlink-protos-repo href "https://github.com/smartcontractkit/chainlink-protos"

	classDef outline stroke-dasharray:6,fill:none;
	class chainlink-common-repo,chainlink-framework-repo,chainlink-protos-repo outline
```
## All modules
```mermaid
flowchart LR

	ccip-contract-examples/chains/evm
	click ccip-contract-examples/chains/evm href "https://github.com/smartcontractkit/ccip-contract-examples"
	ccip-owner-contracts
	click ccip-owner-contracts href "https://github.com/smartcontractkit/ccip-owner-contracts"
	chain-selectors
	click chain-selectors href "https://github.com/smartcontractkit/chain-selectors"
	chainlink-aptos
	click chainlink-aptos href "https://github.com/smartcontractkit/chainlink-aptos"
	chainlink-automation
	click chainlink-automation href "https://github.com/smartcontractkit/chainlink-automation"
	chainlink-ccip
	click chainlink-ccip href "https://github.com/smartcontractkit/chainlink-ccip"
	chainlink-ccip/ccv/chains/evm
	click chainlink-ccip/ccv/chains/evm href "https://github.com/smartcontractkit/chainlink-ccip"
	chainlink-ccip/chains/solana
	click chainlink-ccip/chains/solana href "https://github.com/smartcontractkit/chainlink-ccip"
	chainlink-ccip/chains/solana/gobindings
	click chainlink-ccip/chains/solana/gobindings href "https://github.com/smartcontractkit/chainlink-ccip"
	chainlink-ccip/deployment
	click chainlink-ccip/deployment href "https://github.com/smartcontractkit/chainlink-ccip"
	chainlink-ccv
	click chainlink-ccv href "https://github.com/smartcontractkit/chainlink-ccv"
	chainlink-common --> chainlink-common/pkg/chipingress
	chainlink-common --> chainlink-protos/billing/go
	chainlink-common --> chainlink-protos/cre/go
	chainlink-common --> chainlink-protos/linking-service/go
	chainlink-common --> chainlink-protos/node-platform
	chainlink-common --> chainlink-protos/storage-service
	chainlink-common --> chainlink-protos/workflows/go
	chainlink-common --> freeport
	chainlink-common --> grpc-proxy
	chainlink-common --> libocr
	click chainlink-common href "https://github.com/smartcontractkit/chainlink-common"
	chainlink-common/keystore
	click chainlink-common/keystore href "https://github.com/smartcontractkit/chainlink-common"
	chainlink-common/pkg/chipingress
	click chainlink-common/pkg/chipingress href "https://github.com/smartcontractkit/chainlink-common"
	chainlink-common/pkg/monitoring --> chainlink-common
	chainlink-common/pkg/monitoring --> chainlink-common/pkg/values
	click chainlink-common/pkg/monitoring href "https://github.com/smartcontractkit/chainlink-common"
	chainlink-common/pkg/values
	click chainlink-common/pkg/values href "https://github.com/smartcontractkit/chainlink-common"
	chainlink-cosmos --> chainlink-common/pkg/monitoring
	chainlink-cosmos --> chainlink-framework/chains
	click chainlink-cosmos href "https://github.com/smartcontractkit/chainlink-cosmos"
	chainlink-cosmos/integration-tests --> chainlink-cosmos
	chainlink-cosmos/integration-tests --> chainlink-cosmos/ops
	chainlink-cosmos/integration-tests --> chainlink/deployment
	click chainlink-cosmos/integration-tests href "https://github.com/smartcontractkit/chainlink-cosmos"
	chainlink-cosmos/ops --> chainlink-testing-framework/lib
	click chainlink-cosmos/ops href "https://github.com/smartcontractkit/chainlink-cosmos"
	chainlink-data-streams
	click chainlink-data-streams href "https://github.com/smartcontractkit/chainlink-data-streams"
	chainlink-deployments-framework
	click chainlink-deployments-framework href "https://github.com/smartcontractkit/chainlink-deployments-framework"
	chainlink-evm --> chainlink-common/keystore
	chainlink-evm --> chainlink-evm/gethwrappers
	chainlink-evm --> chainlink-framework/capabilities
	chainlink-evm --> chainlink-framework/chains
	chainlink-evm --> chainlink-protos/svr
	chainlink-evm --> chainlink-tron/relayer
	click chainlink-evm href "https://github.com/smartcontractkit/chainlink-evm"
	chainlink-evm/contracts/cre/gobindings
	click chainlink-evm/contracts/cre/gobindings href "https://github.com/smartcontractkit/chainlink-evm"
	chainlink-evm/gethwrappers
	click chainlink-evm/gethwrappers href "https://github.com/smartcontractkit/chainlink-evm"
	chainlink-feeds
	click chainlink-feeds href "https://github.com/smartcontractkit/chainlink-feeds"
	chainlink-framework/capabilities
	click chainlink-framework/capabilities href "https://github.com/smartcontractkit/chainlink-framework"
	chainlink-framework/chains --> chainlink-framework/multinode
	click chainlink-framework/chains href "https://github.com/smartcontractkit/chainlink-framework"
	chainlink-framework/metrics --> chainlink-common
	click chainlink-framework/metrics href "https://github.com/smartcontractkit/chainlink-framework"
	chainlink-framework/multinode --> chainlink-framework/metrics
	click chainlink-framework/multinode href "https://github.com/smartcontractkit/chainlink-framework"
	chainlink-protos/billing/go
	click chainlink-protos/billing/go href "https://github.com/smartcontractkit/chainlink-protos"
	chainlink-protos/chainlink-ccv/committee-verifier
	click chainlink-protos/chainlink-ccv/committee-verifier href "https://github.com/smartcontractkit/chainlink-protos"
	chainlink-protos/chainlink-ccv/heartbeat
	click chainlink-protos/chainlink-ccv/heartbeat href "https://github.com/smartcontractkit/chainlink-protos"
	chainlink-protos/chainlink-ccv/message-discovery
	click chainlink-protos/chainlink-ccv/message-discovery href "https://github.com/smartcontractkit/chainlink-protos"
	chainlink-protos/chainlink-ccv/verifier
	click chainlink-protos/chainlink-ccv/verifier href "https://github.com/smartcontractkit/chainlink-protos"
	chainlink-protos/cre/go --> chain-selectors
	click chainlink-protos/cre/go href "https://github.com/smartcontractkit/chainlink-protos"
	chainlink-protos/job-distributor
	click chainlink-protos/job-distributor href "https://github.com/smartcontractkit/chainlink-protos"
	chainlink-protos/linking-service/go
	click chainlink-protos/linking-service/go href "https://github.com/smartcontractkit/chainlink-protos"
	chainlink-protos/node-platform
	click chainlink-protos/node-platform href "https://github.com/smartcontractkit/chainlink-protos"
	chainlink-protos/orchestrator
	click chainlink-protos/orchestrator href "https://github.com/smartcontractkit/chainlink-protos"
	chainlink-protos/ring/go
	click chainlink-protos/ring/go href "https://github.com/smartcontractkit/chainlink-protos"
	chainlink-protos/rmn/v1.6/go
	click chainlink-protos/rmn/v1.6/go href "https://github.com/smartcontractkit/chainlink-protos"
	chainlink-protos/storage-service
	click chainlink-protos/storage-service href "https://github.com/smartcontractkit/chainlink-protos"
	chainlink-protos/svr
	click chainlink-protos/svr href "https://github.com/smartcontractkit/chainlink-protos"
	chainlink-protos/workflows/go
	click chainlink-protos/workflows/go href "https://github.com/smartcontractkit/chainlink-protos"
	chainlink-solana
	click chainlink-solana href "https://github.com/smartcontractkit/chainlink-solana"
	chainlink-solana/contracts
	click chainlink-solana/contracts href "https://github.com/smartcontractkit/chainlink-solana"
	chainlink-sui
	click chainlink-sui href "https://github.com/smartcontractkit/chainlink-sui"
	chainlink-sui/deployment
	click chainlink-sui/deployment href "https://github.com/smartcontractkit/chainlink-sui"
	chainlink-testing-framework/framework
	click chainlink-testing-framework/framework href "https://github.com/smartcontractkit/chainlink-testing-framework"
	chainlink-testing-framework/lib --> chainlink-testing-framework/parrot
	chainlink-testing-framework/lib --> chainlink-testing-framework/seth
	click chainlink-testing-framework/lib href "https://github.com/smartcontractkit/chainlink-testing-framework"
	chainlink-testing-framework/parrot
	click chainlink-testing-framework/parrot href "https://github.com/smartcontractkit/chainlink-testing-framework"
	chainlink-testing-framework/seth
	click chainlink-testing-framework/seth href "https://github.com/smartcontractkit/chainlink-testing-framework"
	chainlink-ton
	click chainlink-ton href "https://github.com/smartcontractkit/chainlink-ton"
	chainlink-ton/deployment
	click chainlink-ton/deployment href "https://github.com/smartcontractkit/chainlink-ton"
	chainlink-tron/relayer --> chainlink-common
	click chainlink-tron/relayer href "https://github.com/smartcontractkit/chainlink-tron"
	chainlink/deployment --> ccip-contract-examples/chains/evm
	chainlink/deployment --> ccip-owner-contracts
	chainlink/deployment --> chainlink-ccip/deployment
	chainlink/deployment --> chainlink-deployments-framework
	chainlink/deployment --> chainlink-protos/chainlink-ccv/heartbeat
	chainlink/deployment --> chainlink-protos/job-distributor
	chainlink/deployment --> chainlink-solana/contracts
	chainlink/deployment --> chainlink-sui/deployment
	chainlink/deployment --> chainlink-testing-framework/framework
	chainlink/deployment --> chainlink-testing-framework/lib
	chainlink/deployment --> chainlink-ton/deployment
	chainlink/deployment --> chainlink/v2
	chainlink/deployment --> mcms
	click chainlink/deployment href "https://github.com/smartcontractkit/chainlink"
	chainlink/v2 --> chainlink-aptos
	chainlink/v2 --> chainlink-automation
	chainlink/v2 --> chainlink-ccip
	chainlink/v2 --> chainlink-ccip/ccv/chains/evm
	chainlink/v2 --> chainlink-ccip/chains/solana
	chainlink/v2 --> chainlink-ccip/chains/solana/gobindings
	chainlink/v2 --> chainlink-ccv
	chainlink/v2 --> chainlink-data-streams
	chainlink/v2 --> chainlink-evm
	chainlink/v2 --> chainlink-evm/contracts/cre/gobindings
	chainlink/v2 --> chainlink-feeds
	chainlink/v2 --> chainlink-protos/chainlink-ccv/committee-verifier
	chainlink/v2 --> chainlink-protos/chainlink-ccv/message-discovery
	chainlink/v2 --> chainlink-protos/chainlink-ccv/verifier
	chainlink/v2 --> chainlink-protos/orchestrator
	chainlink/v2 --> chainlink-protos/ring/go
	chainlink/v2 --> chainlink-protos/rmn/v1.6/go
	chainlink/v2 --> chainlink-solana
	chainlink/v2 --> chainlink-sui
	chainlink/v2 --> chainlink-ton
	chainlink/v2 --> cre-sdk-go
	chainlink/v2 --> cre-sdk-go/capabilities/networking/http
	chainlink/v2 --> cre-sdk-go/capabilities/scheduler/cron
	chainlink/v2 --> quarantine
	chainlink/v2 --> smdkg
	chainlink/v2 --> tdh2/go/ocr2/decryptionplugin
	chainlink/v2 --> wsrpc
	click chainlink/v2 href "https://github.com/smartcontractkit/chainlink"
	cre-sdk-go
	click cre-sdk-go href "https://github.com/smartcontractkit/cre-sdk-go"
	cre-sdk-go/capabilities/networking/http
	click cre-sdk-go/capabilities/networking/http href "https://github.com/smartcontractkit/cre-sdk-go"
	cre-sdk-go/capabilities/scheduler/cron
	click cre-sdk-go/capabilities/scheduler/cron href "https://github.com/smartcontractkit/cre-sdk-go"
	freeport
	click freeport href "https://github.com/smartcontractkit/freeport"
	go-sumtype2
	click go-sumtype2 href "https://github.com/smartcontractkit/go-sumtype2"
	grpc-proxy
	click grpc-proxy href "https://github.com/smartcontractkit/grpc-proxy"
	libocr --> go-sumtype2
	click libocr href "https://github.com/smartcontractkit/libocr"
	mcms
	click mcms href "https://github.com/smartcontractkit/mcms"
	quarantine
	click quarantine href "https://github.com/smartcontractkit/quarantine"
	smdkg --> libocr
	smdkg --> tdh2/go/tdh2
	click smdkg href "https://github.com/smartcontractkit/smdkg"
	tdh2/go/ocr2/decryptionplugin
	click tdh2/go/ocr2/decryptionplugin href "https://github.com/smartcontractkit/tdh2"
	tdh2/go/tdh2
	click tdh2/go/tdh2 href "https://github.com/smartcontractkit/tdh2"
	wsrpc
	click wsrpc href "https://github.com/smartcontractkit/wsrpc"

	subgraph chainlink-repo[chainlink]
		 chainlink/deployment
		 chainlink/v2
	end
	click chainlink-repo href "https://github.com/smartcontractkit/chainlink"

	subgraph chainlink-ccip-repo[chainlink-ccip]
		 chainlink-ccip
		 chainlink-ccip/ccv/chains/evm
		 chainlink-ccip/chains/solana
		 chainlink-ccip/chains/solana/gobindings
		 chainlink-ccip/deployment
	end
	click chainlink-ccip-repo href "https://github.com/smartcontractkit/chainlink-ccip"

	subgraph chainlink-common-repo[chainlink-common]
		 chainlink-common
		 chainlink-common/keystore
		 chainlink-common/pkg/chipingress
		 chainlink-common/pkg/monitoring
		 chainlink-common/pkg/values
	end
	click chainlink-common-repo href "https://github.com/smartcontractkit/chainlink-common"

	subgraph chainlink-cosmos-repo[chainlink-cosmos]
		 chainlink-cosmos
		 chainlink-cosmos/integration-tests
		 chainlink-cosmos/ops
	end
	click chainlink-cosmos-repo href "https://github.com/smartcontractkit/chainlink-cosmos"

	subgraph chainlink-evm-repo[chainlink-evm]
		 chainlink-evm
		 chainlink-evm/contracts/cre/gobindings
		 chainlink-evm/gethwrappers
	end
	click chainlink-evm-repo href "https://github.com/smartcontractkit/chainlink-evm"

	subgraph chainlink-framework-repo[chainlink-framework]
		 chainlink-framework/capabilities
		 chainlink-framework/chains
		 chainlink-framework/metrics
		 chainlink-framework/multinode
	end
	click chainlink-framework-repo href "https://github.com/smartcontractkit/chainlink-framework"

	subgraph chainlink-protos-repo[chainlink-protos]
		 chainlink-protos/billing/go
		 chainlink-protos/chainlink-ccv/committee-verifier
		 chainlink-protos/chainlink-ccv/heartbeat
		 chainlink-protos/chainlink-ccv/message-discovery
		 chainlink-protos/chainlink-ccv/verifier
		 chainlink-protos/cre/go
		 chainlink-protos/job-distributor
		 chainlink-protos/linking-service/go
		 chainlink-protos/node-platform
		 chainlink-protos/orchestrator
		 chainlink-protos/ring/go
		 chainlink-protos/rmn/v1.6/go
		 chainlink-protos/storage-service
		 chainlink-protos/svr
		 chainlink-protos/workflows/go
	end
	click chainlink-protos-repo href "https://github.com/smartcontractkit/chainlink-protos"

	subgraph chainlink-solana-repo[chainlink-solana]
		 chainlink-solana
		 chainlink-solana/contracts
	end
	click chainlink-solana-repo href "https://github.com/smartcontractkit/chainlink-solana"

	subgraph chainlink-sui-repo[chainlink-sui]
		 chainlink-sui
		 chainlink-sui/deployment
	end
	click chainlink-sui-repo href "https://github.com/smartcontractkit/chainlink-sui"

	subgraph chainlink-testing-framework-repo[chainlink-testing-framework]
		 chainlink-testing-framework/framework
		 chainlink-testing-framework/lib
		 chainlink-testing-framework/parrot
		 chainlink-testing-framework/seth
	end
	click chainlink-testing-framework-repo href "https://github.com/smartcontractkit/chainlink-testing-framework"

	subgraph chainlink-ton-repo[chainlink-ton]
		 chainlink-ton
		 chainlink-ton/deployment
	end
	click chainlink-ton-repo href "https://github.com/smartcontractkit/chainlink-ton"

	subgraph cre-sdk-go-repo[cre-sdk-go]
		 cre-sdk-go
		 cre-sdk-go/capabilities/networking/http
		 cre-sdk-go/capabilities/scheduler/cron
	end
	click cre-sdk-go-repo href "https://github.com/smartcontractkit/cre-sdk-go"

	subgraph tdh2-repo[tdh2]
		 tdh2/go/ocr2/decryptionplugin
		 tdh2/go/tdh2
	end
	click tdh2-repo href "https://github.com/smartcontractkit/tdh2"

	classDef outline stroke-dasharray:6,fill:none;
	class chainlink-repo,chainlink-ccip-repo,chainlink-common-repo,chainlink-cosmos-repo,chainlink-evm-repo,chainlink-framework-repo,chainlink-protos-repo,chainlink-solana-repo,chainlink-sui-repo,chainlink-testing-framework-repo,chainlink-ton-repo,cre-sdk-go-repo,tdh2-repo outline
```
