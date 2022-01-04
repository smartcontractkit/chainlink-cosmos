package terra

import (
	"context"
	"encoding/json"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/smartcontractkit/chainlink-terra/pkg/terra/mocks"
	"github.com/stretchr/testify/mock"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/terra-money/core/app"
	terraSDK "github.com/terra-money/core/x/wasm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/pelletier/go-toml"
	//tmjson "github.com/tendermint/tendermint/libs/json"
	//tmtypes "github.com/tendermint/tendermint/rpc/core/types"
	//rpcclient "github.com/tendermint/tendermint/rpc/client"
	"github.com/terra-money/terra.go/client"
	"github.com/terra-money/terra.go/key"
	"github.com/terra-money/terra.go/msg"
	"golang.org/x/net/context/ctxhttp"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"time"

	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func createKeyFromMnemonic(t *testing.T, mnemonic string) (key.PrivKey, sdk.AccAddress) {
	// Derive Raw Private Key
	privKeyBz, err := key.DerivePrivKeyBz(mnemonic, key.CreateHDPath(0, 0))
	assert.NoError(t, err)
	// Generate StdPrivKey
	privKey, err := key.PrivKeyGen(privKeyBz)
	assert.NoError(t, err)
	addr := msg.AccAddress(privKey.PubKey().Address())
	return privKey, addr
}

type Account struct {
	Name       string
	PrivateKey key.PrivKey
	Address    sdk.AccAddress
}

func setup(t *testing.T) ([]Account, string) {
	testdir, err := ioutil.TempDir("", "integration-test")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(testdir))
	})
	t.Log(testdir)
	chainID := "42"
	_, err = exec.Command("terrad", "init", "integration-test", "-o", "--chain-id", chainID, "--home", testdir).Output()
	require.NoError(t, err)

	// Enable the api server
	p := path.Join(testdir, "config", "app.toml")
	f, err := os.ReadFile(p)
	config, err := toml.Load(string(f))
	require.NoError(t, err)
	config.Set("api.enable", "true")
	require.NoError(t, os.WriteFile(p, []byte(config.String()), 644))
	// TODO: could also speed up the block mining config

	// Create 2 test accounts
	var accounts []Account
	for i := 0; i < 2; i++ {
		account := fmt.Sprintf("test%d", i)
		key, err := exec.Command("terrad", "keys", "add", account, "--output", "json", "--keyring-backend", "test", "--keyring-dir", testdir).Output()
		require.NoError(t, err)
		t.Log("key", string(key), account)
		var k struct {
			Address  string `json:"address"`
			Mnemonic string `json:"mnemonic"`
		}
		require.NoError(t, json.Unmarshal(key, &k))
		privateKey, address := createKeyFromMnemonic(t, k.Mnemonic)
		// Give it 100 luna
		_, err = exec.Command("terrad", "add-genesis-account", k.Address, "100000000uluna", "--home", testdir).Output()
		require.NoError(t, err)
		accounts = append(accounts, Account{
			Name:       account,
			Address:    address,
			PrivateKey: privateKey,
		})
	}
	// Stake 10 luna in first acct
	out, err := exec.Command("terrad", "gentx", accounts[0].Name, "10000000uluna", fmt.Sprintf("--chain-id=%s", chainID), "--keyring-backend", "test", "--keyring-dir", testdir, "--home", testdir).CombinedOutput()
	require.NoError(t, err, string(out))
	out, err = exec.Command("terrad", "collect-gentxs", "--home", testdir).CombinedOutput()
	require.NoError(t, err, string(out))
	cmd := exec.Command("terrad", "start", "--home", testdir)
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		require.NoError(t, cmd.Process.Kill())
	})
	time.Sleep(10 * time.Second) // Wait for api server to boot
	out, err = exec.Command("terrad", "tx", "wasm", "store", "testdata/my_first_contract.wasm",
		"--from", accounts[0].Name, "--gas", "auto", "--fees", "100000uluna", "--chain-id", "42", "--broadcast-mode", "block", "--home", testdir, "--keyring-backend", "test", "--keyring-dir", testdir, "--yes").CombinedOutput()
	out, err = exec.Command("terrad", "tx", "wasm", "instantiate", "1", `{"count":0}`,
		"--from", accounts[0].Name, "--gas", "auto", "--fees", "100000uluna", "--output", "json", "--chain-id", "42", "--broadcast-mode", "block", "--home", testdir, "--keyring-backend", "test", "--keyring-dir", testdir, "--yes").Output()
	require.NoError(t, err, string(out))
	var deployment struct {
		TxHash string `json:"txhash"`
	}
	require.NoError(t, json.Unmarshal(out, &deployment))
	t.Log("deployment", deployment.TxHash)
	return accounts, deployment.TxHash
}

func getContractAddr(t *testing.T, tc *Client, deploymentHash string) sdk.AccAddress {
	deploymentTx, err := tc.clientCtx.Client.Tx(context.Background(), hexutil.MustDecode("0x"+deploymentHash), false)
	require.NoError(t, err)
	var contractAddr string
	for _, etype := range deploymentTx.TxResult.Events {
		if etype.Type == "wasm" {
			for _, attr := range etype.Attributes {
				if string(attr.Key) == "contract_address" {
					contractAddr = string(attr.Value)
				}
			}
		}
	}
	require.NotEqual(t, "", contractAddr)
	contract, err := sdk.AccAddressFromBech32(contractAddr)
	require.NoError(t, err)
	return contract
}

func TestTerraClient(t *testing.T) {
	// Local only for now, could maybe run on CI if we install terrad there?
	//if os.Getenv("TEST_CLIENT") == "" {
	//	t.Skip()
	//}
	accounts, deploymentHash := setup(t)
	cosmosURL := "http://127.0.0.1:1317"
	tendermintURL := "http://127.0.0.1:26657"
	fcdURL := "https://fcd.terra.dev/" // TODO we can mock this

	// https://lcd.terra.dev/swagger/#/
	// https://fcd.terra.dev/swagger
	cl := http.Client{Timeout: 5 * time.Second}
	get := func(url, path string) []byte {
		r, err := ctxhttp.Get(context.Background(), &cl, url+path)
		t.Log(url + path)
		require.NoError(t, err)
		b, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()
		return b
	}

	lggr := new(mocks.Logger)
	lggr.Test(t)
	lggr.On("Infof", mock.Anything, mock.Anything, mock.Anything).Maybe()
	lggr.On("Errorf", mock.Anything, mock.Anything, mock.Anything).Maybe()
	tc, err := NewClient(OCR2Spec{
		FCDNodeEndpointURL: fcdURL,
		TendermintURL:      tendermintURL,
		CosmosURL:          cosmosURL,
		ChainID:            "42",
		FallbackGasPrice:   "0.01",
		GasLimitMultiplier: "1.3",
	}, lggr)
	require.NoError(t, err)
	require.NoError(t, tc.Start())
	defer tc.Close()

	time.Sleep(5 * time.Second)

	// Query initial contract state
	contract := getContractAddr(t, tc, deploymentHash)
	q := NewAbciQueryParams(contract.String(), []byte(`{"get_count":{}}`))
	data, err := tc.codec.MarshalJSON(q)
	require.NoError(t, err)
	count, err := tc.clientCtx.QueryABCI(abci.RequestQuery{
		Data:   data,
		Path:   "custom/wasm/contractStore",
		Height: 0,
		Prove:  false,
	})
	require.NoError(t, err)
	assert.Equal(t, `{"count":0}`, string(count.Value))

	// Change the contract state
	rawMsg := terraSDK.NewMsgExecuteContract(accounts[0].Address, contract, []byte(`{"reset":{"count":5}}`), sdk.Coins{})
	lcd := tc.LCD(tc.GasPrice(), tc.gasLimitMultiplier, accounts[0].PrivateKey, 5*time.Second)
	tx, err := lcd.CreateAndSignTx(context.Background(), client.CreateTxOptions{Msgs: []msg.Msg{
		rawMsg,
	},
		GasLimit: 1000000,
	})
	require.NoError(t, err)
	b, err := tx.GetTxBytes()
	require.NoError(t, err)
	_, err = tc.clientCtx.WithBroadcastMode("block").BroadcastTx(b)
	require.NoError(t, err)

	// Observe changed contract state
	count, err = tc.clientCtx.QueryABCI(abci.RequestQuery{
		Data:   data,
		Path:   "custom/wasm/contractStore",
		Height: 0,
		Prove:  false,
	})
	require.NoError(t, err)
	assert.Equal(t, `{"count":5}`, string(count.Value))

	gp := tc.GasPrice()
	// Should not use fallback
	assert.NotEqual(t, gp.String(), "0.01uluna")
	t.Log(gp)

	lcd = tc.LCD(tc.GasPrice(), tc.gasLimitMultiplier, accounts[0].PrivateKey, 5*time.Second)
	tx, err = lcd.CreateAndSignTx(context.Background(), client.CreateTxOptions{
		Msgs: []msg.Msg{
			msg.NewMsgSend(accounts[0].Address, accounts[1].Address, msg.NewCoins(msg.NewInt64Coin("uluna", 1))), // 1uusd
		},
		GasLimit: 200000,
	})
	require.NoError(t, err)
	b, err = tx.GetTxBytes()
	require.NoError(t, err)
	resp, err := tc.clientCtx.WithBroadcastMode("block").BroadcastTx(b)
	require.NoError(t, err)

	// Note even the blocking command doesn't let you query for the tx right away
	time.Sleep(1 * time.Second)

	// Ensure cosmos endpoints work
	b = get(cosmosURL, "/cosmos/tx/v1beta1/txs/"+resp.TxHash)
	var tx2 txtypes.GetTxResponse
	require.NoError(t, app.MakeEncodingConfig().Marshaler.UnmarshalJSON(b, &tx2))
	t.Log(tx.GetTx().GetFee().String())

	b = get(cosmosURL, "/cosmos/bank/v1beta1/balances/"+accounts[0].Address.String())
	var balances banktypes.QueryAllBalancesResponse
	require.NoError(t, app.MakeEncodingConfig().Marshaler.UnmarshalJSON(b, &balances))
	t.Log(balances.GetBalances().AmountOf("uluna").String())

	// Ensure we can read back the tx with Query
	tr, err := tc.clientCtx.Client.TxSearch(context.Background(), fmt.Sprintf("tx.height=%v", tx2.TxResponse.Height), false, nil, nil, "desc")
	require.NoError(t, err)
	assert.Equal(t, 1, tr.TotalCount)
	assert.Equal(t, tx2.TxResponse.TxHash, tr.Txs[0].Hash.String())
	// Ensure head tracker is working
	assert.True(t, tc.Height > 1)
}
