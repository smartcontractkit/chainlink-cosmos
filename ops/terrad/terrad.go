package terrad

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
	opsChainlink "github.com/smartcontractkit/chainlink-relay/ops/chainlink"
	"github.com/smartcontractkit/chainlink-relay/ops/utils"
	"github.com/smartcontractkit/libocr/offchainreporting2/confighelper"
	"github.com/smartcontractkit/libocr/offchainreporting2/reportingplugin/median"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
)

const (
	// uploaded code IDs
	CW20_ID = iota
	OCR2_ID

	// deployed contract addresses
	LINK
	OCR2
)

type Terrad struct {
	url      string
	chainID  string
	keyID    string
	addr     string
	args     []string
	Uploaded map[int]string
	Deployed map[int]string
}

type key struct {
	Name    string
	Address string
}

func New(ctx *pulumi.Context) (Terrad, error) {
	// check if terrad is installed
	_, err := exec.LookPath("terrad")
	if err != nil {
		return Terrad{}, errors.New("'terrad' is not installed")
	}

	// check the deployer key exists in terrad
	msg := utils.LogStatus("'terrad' is configured correctly")
	keyID := config.Require(ctx, "TERRA-DEPLOYER")
	out, err := exec.Command("terrad", "keys", "show", keyID, "--output", "json").Output()
	if msg.Check(err) != nil {
		return Terrad{}, fmt.Errorf("'%s' key not found in terrad - please set", keyID)
	}
	var deployerAddr key
	if err := json.Unmarshal(out, &deployerAddr); err != nil {
		return Terrad{}, err
	}
	fmt.Printf("Deployer address: %s\n", deployerAddr.Address)

	chainID := config.Require(ctx, "CL-RELAY_CHAINID")
	return Terrad{
		url:      config.Require(ctx, "CL-TENDERMINT_URL"),
		chainID:  chainID,
		keyID:    keyID,
		addr:     deployerAddr.Address,
		args:     []string{"--from", keyID, "--chain-id", chainID, "--gas=auto", "--gas-adjustment=1.25", "--fees=100000uluna", "--broadcast-mode=block", "-y", "-o=json"},
		Uploaded: map[int]string{},
		Deployed: map[int]string{},
	}, err
}

type TxResponse struct {
	Logs cosmostypes.ABCIMessageLogs
}

func (t *Terrad) Load() error {
	msg := utils.LogStatus("Uploading contract artifacts")
	for _, contract := range []string{"ocr2", "cw20_base"} {
		args := append([]string{"tx", "wasm", "store", fmt.Sprintf("./terrad/artifacts/%s.wasm", contract)}, t.args...)
		out, err := exec.Command("terrad", args...).Output()
		if err != nil {
			return msg.Check(err)
		}

		var res TxResponse
		if err := json.Unmarshal(out, &res); err != nil {
			return msg.Check(err)
		}
		for _, event := range res.Logs[0].Events {
			if event.Type == "store_code" {
				for _, attr := range event.Attributes {
					if attr.Key == "code_id" {
						var key int
						switch contract {
						case "ocr2":
							key = OCR2_ID
						case "cw20_base":
							key = CW20_ID
						default:
							return errors.New("unknown contract type does not have assigned key")
						}
						t.Uploaded[key] = attr.Value
					}
				}
			}
		}
	}
	return msg.Check(nil)
}

type balance struct {
	Address string `json:"address"`
	Amount  string `json:"amount"`
}

type LINKinit struct {
	Name            string      `json:"name"`
	Symbol          string      `json:"symbol"`
	Decimals        int         `json:"decimals"`
	InitialBalances []balance   `json:"initial_balances"`
	Mint            interface{} `json:"mint"`
	Marketing       interface{} `json:"marketing"`
}

func (t *Terrad) DeployLINK() error {
	msg := utils.LogStatus("Deployed LINK token")
	initBal := balance{Address: t.addr, Amount: "1000000000000000000000000000"}
	initMsg := LINKinit{
		Name:            "ChainLink Token",
		Symbol:          "LINK",
		Decimals:        18,
		InitialBalances: []balance{initBal},
		Mint:            nil,
		Marketing:       nil,
	}

	msgBytes, err := json.Marshal(initMsg)
	if err != nil {
		return msg.Check(err)
	}

	args := append([]string{"tx", "wasm", "instantiate", t.Uploaded[CW20_ID], string(msgBytes)}, t.args...)
	out, err := exec.Command("terrad", args...).Output()
	if err != nil {
		return msg.Check(err)
	}

	var res TxResponse
	if err := json.Unmarshal(out, &res); err != nil {
		return msg.Check(err)
	}
	for _, event := range res.Logs[0].Events {
		if event.Type == "instantiate_contract" {
			for _, attr := range event.Attributes {
				if attr.Key == "contract_address" {
					t.Deployed[LINK] = attr.Value
					fmt.Printf(" - %s", attr.Value)
				}
			}
		}
	}

	return msg.Check(nil)
}

type OCRinit struct {
	LinkToken                 string `json:"link_token"`
	MinAnswer                 string `json:"min_answer"`
	MaxAnswer                 string `json:"max_answer"`
	BillingAccessController   string `json:"billing_access_controller"`
	RequesterAccessController string `json:"requester_access_controller"`
	Decimals                  int    `json:"decimals"`
	Description               string `json:"description"`
}

func (t *Terrad) DeployOCR() error {
	msg := utils.LogStatus("Deployed OCR contract")
	initMsg := OCRinit{
		LinkToken:                 t.Deployed[LINK],
		MinAnswer:                 "0",
		MaxAnswer:                 "999999999999999999",
		Decimals:                  8,
		BillingAccessController:   "terra1dcegyrekltswvyy0xy69ydgxn9x8x32zdtapd8", // placeholder
		RequesterAccessController: "terra1dcegyrekltswvyy0xy69ydgxn9x8x32zdtapd8", // placeholder
		Description:               "LINK/USD - OCR2",
	}

	msgBytes, err := json.Marshal(initMsg)
	if err != nil {
		return msg.Check(err)
	}

	args := append([]string{"tx", "wasm", "instantiate", t.Uploaded[OCR2_ID], string(msgBytes)}, t.args...)
	out, err := exec.Command("terrad", args...).Output()
	if err != nil {
		return msg.Check(err)
	}

	var res TxResponse
	if err := json.Unmarshal(out, &res); err != nil {
		return msg.Check(err)
	}
	for _, event := range res.Logs[0].Events {
		if event.Type == "instantiate_contract" {
			for _, attr := range event.Attributes {
				if attr.Key == "contract_address" {
					t.Deployed[OCR2] = attr.Value
					fmt.Printf(" - %s", attr.Value)
				}
			}
		}
	}

	return msg.Check(nil)
}

type Send struct {
	Send SendDetails `json:"send"`
}

type SendDetails struct {
	Contract string `json:"contract"`
	Amount   string `json:"amount"`
	Msg      string `json:"msg"`
}

func (t Terrad) TransferLINK() error {
	msg := utils.LogStatus("Sending LINK to OCR contract")
	tx := Send{
		Send: SendDetails{
			Contract: t.Deployed[OCR2],
			Amount:   "100000000000000000000",
			Msg:      "",
		},
	}

	txBytes, err := json.Marshal(tx)
	if err != nil {
		return msg.Check(err)
	}

	args := append([]string{"tx", "wasm", "execute", t.Deployed[LINK], string(txBytes)}, t.args...)
	_, err = exec.Command("terrad", args...).Output()
	return msg.Check(err)
}

type SetConfig struct {
	SetConfig SetConfigDetails `json:"set_config"`
}

type SetConfigDetails struct {
	Signers               ByteArrayArray `json:"signers"`
	Transmitters          []string       `json:"transmitters"`
	F                     uint8          `json:"f"`
	OnchainConfig         ByteArray      `json:"onchain_config"`
	OffchainConfigVersion uint64         `json:"offchain_config_version"`
	OffchainConfig        ByteArray      `json:"offchain_config"`
}

type ByteArray []byte

func (b ByteArray) MarshalJSON() ([]byte, error) {
	var result string
	if b == nil {
		result = "null"
	} else {
		result = strings.Join(strings.Fields(fmt.Sprintf("%d", b)), ",")
	}
	return []byte(result), nil
}

type ByteArrayArray [][]byte

func (b ByteArrayArray) MarshalJSON() ([]byte, error) {
	var result string
	if b == nil {
		result = "null"
	} else {
		result = strings.Join(strings.Fields(fmt.Sprintf("%d", b)), ",")
	}
	return []byte(result), nil
}

func (t Terrad) InitOCR(keys []opsChainlink.NodeKeys) error {
	msg := utils.LogStatus("Set config on OCR contract")
	S := []int{}
	helperOracles := []confighelper.OracleIdentityExtra{}
	for _, k := range keys {
		S = append(S, 1)
		offchainPKByte, err := hex.DecodeString(k.OCR2OffchainPublicKey)
		if err != nil {
			return err
		}
		onchainPKByte, err := hex.DecodeString(k.OCR2OnchainPublicKey)
		if err != nil {
			return err
		}
		configPKByteTemp, err := hex.DecodeString(k.OCR2ConfigPublicKey)
		if err != nil {
			return err
		}
		configPKByte := [32]byte{}
		copy(configPKByte[:], configPKByteTemp)
		helperOracles = append(helperOracles, confighelper.OracleIdentityExtra{
			OracleIdentity: confighelper.OracleIdentity{
				OffchainPublicKey: types.OffchainPublicKey(offchainPKByte),
				OnchainPublicKey:  types.OnchainPublicKey(onchainPKByte),
				PeerID:            k.P2PID,
				TransmitAccount:   types.Account(k.OCR2Transmitter),
			},
			ConfigEncryptionPublicKey: types.ConfigEncryptionPublicKey(configPKByte),
		})
	}

	alphaPPB := uint64(1000000)
	signers, transmitters, f, onchainConfig, offchainConfigVersion, offchainConfig, err := confighelper.ContractSetConfigArgsForTests(
		2*time.Second,        // deltaProgress time.Duration,
		5*time.Second,        // deltaResend time.Duration,
		1*time.Second,        // deltaRound time.Duration,
		500*time.Millisecond, // deltaGrace time.Duration,
		5*time.Second,        // deltaStage time.Duration,
		3,                    // rMax uint8,
		S,                    // s []int,
		helperOracles,        // oracles []OracleIdentityExtra,
		median.OffchainConfig{
			false,
			alphaPPB,
			false,
			alphaPPB,
			15 * time.Second,
		}.Encode(), //reportingPluginConfig []byte,
		500*time.Millisecond, // maxDurationQuery time.Duration,
		500*time.Millisecond, // maxDurationObservation time.Duration,
		500*time.Millisecond, // maxDurationReport time.Duration,
		2*time.Second,        // maxDurationShouldAcceptFinalizedReport time.Duration,
		2*time.Second,        // maxDurationShouldTransmitAcceptedReport time.Duration,
		1,                    // f int,
		[]byte{},             // onchainConfig []byte (calculated by the contract)
	)

	// convert type for marshalling
	signerArray := [][]byte{}
	transmitterArray := []string{}
	for i := 0; i < len(signers); i++ {
		signerArray = append(signerArray, signers[i])
		transmitterArray = append(transmitterArray, string(transmitters[i]))
	}

	tx := SetConfig{
		SetConfig: SetConfigDetails{
			Signers:               signerArray,
			Transmitters:          transmitterArray,
			F:                     f,
			OnchainConfig:         onchainConfig,
			OffchainConfigVersion: offchainConfigVersion,
			OffchainConfig:        offchainConfig,
		},
	}
	if err != nil {
		return msg.Check(err)
	}

	txBytes, err := json.Marshal(tx)
	if err != nil {
		return msg.Check(err)
	}

	args := append([]string{"tx", "wasm", "execute", t.Deployed[OCR2], string(txBytes)}, t.args...)
	_, err = exec.Command("terrad", args...).Output()
	return msg.Check(err)
}

func (t Terrad) OCR2Address() string {
	return t.Deployed[OCR2]
}

func (t Terrad) Addresses() map[int]string {
	return t.Deployed
}

func (t Terrad) Fund(addresses []string) error {
	for _, a := range addresses {
		msg := utils.LogStatus(fmt.Sprintf("Funded %s", a))
		args := append([]string{"tx", "bank", "send", t.keyID, a, "1000000000uluna"}, t.args...)
		if _, err := exec.Command("terrad", args...).Output(); msg.Check(err) != nil {
			return err
		}
	}
	return nil
}
