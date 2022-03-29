package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/rs/zerolog/log"
	"github.com/smartcontractkit/helmenv/environment"
	ifclient "github.com/smartcontractkit/integrations-framework/client"
	"github.com/smartcontractkit/terra.go/client"
	"github.com/smartcontractkit/terra.go/key"
	"github.com/smartcontractkit/terra.go/msg"
	"gopkg.in/yaml.v2"
)

const (
	// DefaultTerraTXTimeout is default http client timeout
	DefaultTerraTXTimeout = 20 * time.Second
	// DefaultBroadcastMode is set to MODE_BLOCK it means when call returns, tx is mined and accepted in the next block
	DefaultBroadcastMode = tx.BroadcastMode_BROADCAST_MODE_BLOCK
	// EventAttrKeyCodeID code id
	EventAttrKeyCodeID = "code_id"
	// EventAttrKeyContractAddress contract Address as bech32
	EventAttrKeyContractAddress = "contract_address"
)

type NetworkConfig struct {
	ContractDeployed bool          `mapstructure:"contracts_deployed" yaml:"contracts_deployed"`
	External         bool          `mapstructure:"external" yaml:"external"`
	RetryAttempts    uint          `mapstructure:"retry_attempts" yaml:"retry_attempts"`
	RetryDelay       time.Duration `mapstructure:"retry_delay" yaml:"retry_delay"`
	Currency         string        `mapstructure:"currency" yaml:"currency"`
	Name             string        `mapstructure:"name" yaml:"name"`
	ID               string        `mapstructure:"id" yaml:"id"`
	ChainID          int64         `mapstructure:"chain_id" yaml:"chain_id"`
	URL              string        `mapstructure:"url" yaml:"url"`
	URLs             []string      `mapstructure:"urls" yaml:"urls"`
	Type             string        `mapstructure:"type" yaml:"type"`
	Mnemonics        []string      `mapstructure:"mnemonics" yaml:"mnemonics"`
	Timeout          time.Duration `mapstructure:"transaction_timeout" yaml:"transaction_timeout"`
}

// TerraWallet is the implementation to allow testing with Terra based wallets
// only first derived key for each Mnemonic is used now (PrivateKey)
type TerraWallet struct {
	Mnemonic   string
	PrivateKey key.PrivKey
	AccAddress msg.AccAddress
}

// LoadWallet returns the instantiated Terra wallet based on a given Mnemonic with 0,0 derivation path
func LoadWallet(mnemonic string) (*TerraWallet, error) {
	privKeyBz, err := key.DerivePrivKeyBz(mnemonic, key.CreateHDPath(0, 0))
	if err != nil {
		return nil, err
	}
	privKey, err := key.PrivKeyGen(privKeyBz)
	if err != nil {
		return nil, err
	}
	accAddr, err := msg.AccAddressFromHex(privKey.PubKey().Address().String())
	if err != nil {
		return nil, err
	}
	return &TerraWallet{
		Mnemonic:   mnemonic,
		PrivateKey: privKey,
		AccAddress: accAddr,
	}, nil
}

// TerraLCDClient is terra lite chain client allowing to upload and interact with the contracts
type TerraLCDClient struct {
	*client.LCDClient
	Clients       []ifclient.BlockchainClient
	Wallets       []*TerraWallet
	DefaultWallet *TerraWallet
	BroadcastMode tx.BroadcastMode
	ID            int
	Config        *NetworkConfig
}

func NewEphemeralWallet() (*TerraWallet, error) {
	m, err := key.CreateMnemonic()
	if err != nil {
		return nil, err
	}
	privKey, err := key.PrivKeyGen([]byte(m))
	if err != nil {
		return nil, err
	}
	accAddr, err := msg.AccAddressFromHex(privKey.PubKey().Address().String())
	if err != nil {
		return nil, err
	}
	return &TerraWallet{
		Mnemonic:   m,
		PrivateKey: privKey,
		AccAddress: accAddr,
	}, nil
}

func (t *TerraLCDClient) GetNetworkType() string {
	return t.Config.Type
}

func (t *TerraLCDClient) ContractsDeployed() bool {
	return t.Config.ContractDeployed
}

func ClientURLSFunc() func(e *environment.Environment) ([]*url.URL, error) {
	return func(e *environment.Environment) ([]*url.URL, error) {
		urls := make([]*url.URL, 0)
		wsURL, err := e.Charts.Connections("localterra").LocalURLByPort("lcd", environment.HTTP)
		if err != nil {
			return nil, err
		}
		log.Debug().Interface("HTTP_URL", wsURL).Msg("URLS loaded")
		urls = append(urls, wsURL)
		return urls, nil
	}
}

func ClientInitFunc(contracts int) func(networkName string, networkConfig map[string]interface{}, urls []*url.URL) (ifclient.BlockchainClient, error) {
	return func(networkName string, networkConfig map[string]interface{}, urls []*url.URL) (ifclient.BlockchainClient, error) {
		d, err := yaml.Marshal(networkConfig)
		if err != nil {
			return nil, err
		}
		var cfg *NetworkConfig
		if err = yaml.Unmarshal(d, &cfg); err != nil {
			return nil, err
		}
		cfg.ID = networkName
		urlStrings := make([]string, 0)
		for _, u := range urls {
			urlStrings = append(urlStrings, u.String())
		}
		cfg.URLs = urlStrings
		rootClient, err := NewClient(cfg)
		if err != nil {
			return nil, err
		}
		if err := rootClient.LoadWallets(cfg); err != nil {
			return nil, err
		}
		rootClient.LCDClient.PrivKey = rootClient.Wallets[0].PrivateKey
		rootClient.DefaultWallet = rootClient.Wallets[0]
		for i := 0; i < contracts; i++ {
			c, err := NewClient(cfg)
			if err != nil {
				return nil, err
			}
			w, err := NewEphemeralWallet()
			if err != nil {
				return nil, err
			}
			if err := rootClient.Fund(w.AccAddress.String(), big.NewFloat(1e10)); err != nil {
				return nil, err
			}
			c.LCDClient.PrivKey = w.PrivateKey
			c.DefaultWallet = w
			rootClient.Clients = append(rootClient.Clients, c)
		}
		return rootClient, nil
	}
}

// NewClient derives deployer key and creates new LCD client for Terra
func NewClient(cfg *NetworkConfig) (*TerraLCDClient, error) {
	return &TerraLCDClient{
		LCDClient: client.NewLCDClient(
			cfg.URLs[0],
			cfg.Name,
			msg.NewDecCoinFromDec(cfg.Currency, msg.NewDecFromIntWithPrec(msg.NewInt(15), 2)),
			msg.NewDecFromIntWithPrec(msg.NewInt(15), 1),
			nil,
			DefaultTerraTXTimeout,
		),
		Config:        cfg,
		BroadcastMode: DefaultBroadcastMode,
	}, nil
}

func (t *TerraLCDClient) LoadWallets(nc interface{}) error {
	cfg := nc.(*NetworkConfig)
	for _, mnemonic := range cfg.Mnemonics {
		w, err := LoadWallet(mnemonic)
		if err != nil {
			return err
		}
		t.Wallets = append(t.Wallets, w)
	}
	return nil
}

func (t *TerraLCDClient) SetWallet(num int) error {
	if num > len(t.Wallets) {
		return fmt.Errorf("wallet %d not found", num)
	}
	t.LCDClient.PrivKey = t.Wallets[num].PrivateKey
	t.DefaultWallet = t.Wallets[num]
	return nil
}

func (t *TerraLCDClient) EstimateCostForChainlinkOperations(amountOfOperations int) (*big.Float, error) {
	panic("implement me")
}

func (t *TerraLCDClient) GetChainID() int64 {
	panic("implement me")
}

func (t *TerraLCDClient) SwitchNode(node int) error {
	panic("implement me")
}

func (t *TerraLCDClient) GetClients() []ifclient.BlockchainClient {
	return t.Clients
}

func (t *TerraLCDClient) HeaderHashByNumber(ctx context.Context, bn *big.Int) (string, error) {
	panic("implement me")
}

func (t *TerraLCDClient) BlockNumber(ctx context.Context) (uint64, error) {
	panic("implement me")
}

func (t *TerraLCDClient) HeaderTimestampByNumber(ctx context.Context, bn *big.Int) (uint64, error) {
	panic("implement me")
}

func (t *TerraLCDClient) GasStats() *ifclient.GasStats {
	panic("implement me")
}

func (t *TerraLCDClient) ParallelTransactions(enabled bool) {
	panic("implement me")
}

func (t *TerraLCDClient) Close() error {
	panic("implement me")
}

func (t *TerraLCDClient) AddHeaderEventSubscription(key string, subscriber ifclient.HeaderEventSubscription) {
	panic("implement me")
}

func (t *TerraLCDClient) DeleteHeaderEventSubscription(key string) {
	panic("implement me")
}

func (t *TerraLCDClient) WaitForEvents() error {
	panic("implement me")
}

// Get gets default client as an interface{}
func (t *TerraLCDClient) Get() interface{} {
	return t
}

// GetNetworkName gets the ID of the chain that the clients are connected to
func (t *TerraLCDClient) GetNetworkName() string {
	return t.Config.Name
}

// GetID gets client ID, node number it's connected to
func (t *TerraLCDClient) GetID() int {
	return t.ID
}

// SetID sets client ID (node)
func (t *TerraLCDClient) SetID(id int) {
	t.ID = id
}

// SetDefaultClient sets default client to perform calls to the network
func (t *TerraLCDClient) SetDefaultClient(clientID int) error {
	// We are using SetDefaultClient and GetClients only for multinode networks to check reorgs,
	// but Terra uses Tendermint PBFT with an absolute finality
	return nil
}

// SuggestGasPrice gets suggested gas price
func (t *TerraLCDClient) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	// client already have simulation for gas estimation by default turned on
	panic("implement me")
}

// CalculateTxGas calculates tx gas cost accordingly gas used plus buffer, converts it to big.Float for funding
func (t *TerraLCDClient) CalculateTxGas(gasUsedValue *big.Int) (*big.Float, error) {
	panic("implement me")
}

// GetDefaultWallet gets the default wallet
func (t *TerraLCDClient) GetDefaultWallet() *ifclient.EthereumWallet {
	panic("implement me")
}

// Instantiate deploys WASM code and instantiating a contract
func (t *TerraLCDClient) Instantiate(path string, instMsg interface{}) (string, error) {
	sender := t.DefaultWallet.AccAddress
	dat, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	instMsgBytes, err := json.Marshal(instMsg)
	if err != nil {
		return "", err
	}
	txBlockResp, err := t.SendTX(client.CreateTxOptions{
		Msgs: []msg.Msg{
			msg.NewMsgStoreCode(sender, dat),
		},
	}, false)
	if err != nil {
		return "", err
	}
	codeID, err := t.GetEventAttrValue(txBlockResp, EventAttrKeyCodeID)
	if err != nil {
		return "", err
	}
	cID, err := strconv.Atoi(codeID)
	if err != nil {
		return "", err
	}
	log.Info().
		Str("Path", path).
		Int("CodeID", cID).
		Msg("Instantiating contract")
	txBlockResp, err = t.SendTX(client.CreateTxOptions{
		Msgs: []msg.Msg{
			msg.NewMsgInstantiateContract(
				sender,
				sender,
				uint64(cID),
				instMsgBytes,
				msg.NewCoins(msg.NewInt64Coin(t.Config.Currency, 1e8)),
			),
		},
	}, false)
	if err != nil {
		return "", err
	}
	contractAddr, err := t.GetEventAttrValue(txBlockResp, EventAttrKeyContractAddress)
	if err != nil {
		return "", err
	}
	return contractAddr, nil
}

// SendTX signs and broadcast tx using default broadcast mode
func (t *TerraLCDClient) SendTX(txOpts client.CreateTxOptions, logMsgs bool) (*types.TxResponse, error) {
	var txBlockResp *types.TxResponse
	if logMsgs {
		log.Info().Interface("Msgs", txOpts.Msgs).Msg("Sending TX")
	}
	for i := 0; i < int(t.Config.RetryAttempts); i++ {
		txn, err := t.CreateAndSignTx(context.Background(), txOpts)
		if err != nil {
			log.Error().Err(err).Msg("Simulate error, retrying")
			continue
		}
		txBlockResp, err = t.Broadcast(context.Background(), txn, t.BroadcastMode)
		if err != nil {
			log.Error().Err(err).Msg("Broadcast error, retrying")
			continue
		}
		log.Info().Interface("Response", txBlockResp).Msg("TX Response")
		switch txBlockResp.Code {
		case 32:
			log.Warn().Msg("Account sequence mismatch, retrying")
			continue
		case 0:
			return txBlockResp, nil
		default:
			return txBlockResp, errors.Wrapf(err, "tx failed with code: %d: %s", txBlockResp.Code, txBlockResp.RawLog)
		}
	}
	return txBlockResp, nil
}

// GetEventAttrValue gets attr value by key from sdkTypes.TxResponse
func (t *TerraLCDClient) GetEventAttrValue(tx *types.TxResponse, attrKey string) (string, error) {
	if tx == nil {
		return "", errors.New("tx is nil")
	}
	for _, eventLog := range tx.Logs {
		for _, event := range eventLog.Events {
			for _, eventAttr := range event.Attributes {
				if eventAttr.Key == attrKey {
					return eventAttr.Value, nil
				}
			}
		}
	}
	return "", fmt.Errorf("no attr %s found in TX response", attrKey)
}

// Fund funds a contracts with both native currency and LINK token
func (t *TerraLCDClient) Fund(toAddress string, nativeAmount *big.Float) error {
	sender := t.DefaultWallet.AccAddress
	toAddrBech32, err := msg.AccAddressFromBech32(toAddress)
	if err != nil {
		return err
	}
	amount, _ := nativeAmount.Int64()
	_, err = t.SendTX(client.CreateTxOptions{
		Msgs: []msg.Msg{
			msg.NewMsgSend(
				sender,
				toAddrBech32,
				msg.NewCoins(msg.NewInt64Coin(t.Config.Currency, amount))),
		},
	}, true)
	if err != nil {
		return err
	}
	return nil
}
