package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/pelletier/go-toml"
	"github.com/smartcontractkit/terra.go/key"
	"github.com/smartcontractkit/terra.go/msg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// 0.001
var minGasPrice = msg.NewDecCoinFromDec("uluna", msg.NewDecWithPrec(1, 3))

func SetupLocalTerraNode(t *testing.T, chainID string) ([]Account, string) {
	testdir, err := ioutil.TempDir("", "integration-test")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(testdir))
	})
	t.Log(testdir)
	_, err = exec.Command("terrad", "init", "integration-test", "-o", "--chain-id", chainID, "--home", testdir).Output()
	require.NoError(t, err)

	p := path.Join(testdir, "config", "app.toml")
	f, err := os.ReadFile(p)
	require.NoError(t, err)
	config, err := toml.Load(string(f))
	require.NoError(t, err)
	// Enable if desired to use lcd endpoints config.Set("api.enable", "true")
	config.Set("minimum-gas-prices", minGasPrice.String())
	require.NoError(t, os.WriteFile(p, []byte(config.String()), 0600))
	// TODO: could also speed up the block mining config

	// Create 2 test accounts
	var accounts []Account
	for i := 0; i < 2; i++ {
		account := fmt.Sprintf("test%d", i)
		key, err2 := exec.Command("terrad", "keys", "add", account, "--output", "json", "--keyring-backend", "test", "--keyring-dir", testdir).CombinedOutput()
		require.NoError(t, err2, string(key))
		var k struct {
			Address  string `json:"address"`
			Mnemonic string `json:"mnemonic"`
		}
		require.NoError(t, json.Unmarshal(key, &k))
		expAcctAddr, err3 := sdk.AccAddressFromBech32(k.Address)
		require.NoError(t, err3)
		privateKey, address := createKeyFromMnemonic(t, k.Mnemonic)
		require.Equal(t, expAcctAddr, address)
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
	// Wait for api server to boot
	var ready bool
	for i := 0; i < 10; i++ {
		time.Sleep(1 * time.Second)
		out, err = exec.Command("curl", "http://127.0.0.1:26657/abci_info").Output()
		require.NoError(t, err)
		var a struct {
			Result struct {
				Response struct {
					LastBlockHeight string `json:"last_block_height"`
				} `json:"response"`
			} `json:"result"`
		}
		require.NoError(t, json.Unmarshal(out, &a))
		if a.Result.Response.LastBlockHeight != "" {
			ready = true
			break
		}
	}
	require.True(t, ready)
	return accounts, testdir
}

func DeployTestContract(t *testing.T, deployAccount, ownerAccount Account, tc *Client, testdir, wasmTestContractPath string) sdk.AccAddress {
	out, err := exec.Command("terrad", "tx", "wasm", "store", wasmTestContractPath,
		"--from", deployAccount.Name, "--gas", "auto", "--fees", "100000uluna", "--chain-id", "42", "--broadcast-mode", "block", "--home", testdir, "--keyring-backend", "test", "--keyring-dir", testdir, "--yes").CombinedOutput()
	require.NoError(t, err, string(out))
	an, sn, err2 := tc.Account(ownerAccount.Address)
	require.NoError(t, err2)
	r, err3 := tc.SignAndBroadcast([]msg.Msg{
		msg.NewMsgInstantiateContract(ownerAccount.Address, nil, 1, []byte(`{"count":0}`), nil)}, an, sn, minGasPrice, ownerAccount.PrivateKey, txtypes.BroadcastMode_BROADCAST_MODE_BLOCK)
	require.NoError(t, err3)
	return GetContractAddr(t, tc, r.TxResponse.TxHash)
}

func GetContractAddr(t *testing.T, tc *Client, deploymentHash string) sdk.AccAddress {
	var deploymentTx *txtypes.GetTxResponse
	var err error
	for try := 0; try < 5; try++ {
		deploymentTx, err = tc.Tx(deploymentHash)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
	}
	require.NoError(t, err)
	var contractAddr string
	for _, etype := range deploymentTx.TxResponse.Events {
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
