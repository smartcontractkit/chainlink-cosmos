# Uncomment relevant section to make terrad CLI more usable.
#  Add $NODE on command line for any commands, $TXFLAGS for tx commands

#export CHAIN_ID=localterra
#export RPC=tcp://localhost:26657

#export CHAIN_ID=mainnet
#export RPC=http://public-node.terra.dev:26657

export CHAIN_ID="bombay-12"
export RPC=https://terra-testnet-2.simply-vc.com.mt:443/345DKJ45F6G5/rpc/

export NODE="--node $RPC"
export TXFLAG="${NODE} --chain-id $CHAIN_ID --fees 100000uluna --gas auto --gas-adjustment 1.2 --broadcast-mode=block"
#export TXFLAG="${NODE} --chain-id $CHAIN_ID --gas-prices 0.50luna --gas auto --gas-adjustment 1.2"
