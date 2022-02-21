package terrad

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	opsChainlink "github.com/smartcontractkit/chainlink-relay/ops/chainlink"
	"github.com/smartcontractkit/chainlink-relay/ops/utils"
	relayUtils "github.com/smartcontractkit/chainlink-relay/ops/utils"
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
	RequesterAccessController
	BillingAccessController
)

type key struct {
	Name    string
	Address string
}

type Deployer struct {
	gauntlet relayUtils.Gauntlet
	network  string
	Account  map[int]string
}

func New(ctx *pulumi.Context) (Deployer, error) {
	// check if yarn is installed
	yarn, err := exec.LookPath("yarn")
	if err != nil {
		return Deployer{}, errors.New("'yarn' is not installed")
	}
	fmt.Printf("yarn is available at %s\n", yarn)

	// Change path to root directory
	cwd, _ := os.Getwd()
	os.Chdir(filepath.Join(cwd, "../"))

	fmt.Println("Installing dependencies")
	if _, err = exec.Command(yarn).Output(); err != nil {
		return Deployer{}, errors.New("error install dependencies")
	}

	// Generate Gauntlet Binary
	fmt.Println("Generating Gauntlet binary...")
	_, err = exec.Command(yarn, "bundle").Output()
	if err != nil {
		return Deployer{}, errors.New("error generating gauntlet binary")
	}

	// TODO: Should come from pulumi context
	os.Setenv("SKIP_PROMPTS", "true")

	// version := "linux"
	// if config.Get(ctx, "VERSION") == "MACOS" {
	// 	version = "macos"
	// }

	// Check gauntlet works
	os.Chdir(cwd) // move back into ops folder
	gauntletBin := filepath.Join(cwd, "../")
	gauntlet, err := relayUtils.NewGauntlet(gauntletBin)

	if err != nil {
		return Deployer{}, err
	}

	return Deployer{
		gauntlet: gauntlet,
		network:  "local",
		Account:  make(map[int]string),
	}, nil
}

type TxResponse struct {
	Logs cosmostypes.ABCIMessageLogs
}

func (t *Deployer) Load() error {
	msg := utils.LogStatus("Uploading contract artifacts")
	err := t.gauntlet.ExecCommand(
		"upload",
		t.gauntlet.Flag("network", "bombay-testnet"),
		"link",
		"ocr2",
	)

	if err != nil {
		return errors.New("Billing AC initialization failed")
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

func (t *Deployer) DeployLINK() error {
	fmt.Println("Deploying LINK Token...")
	err := t.gauntlet.ExecCommand(
		"token:deploy",
		t.gauntlet.Flag("network", "bombay-testnet"),
	)
	if err != nil {
		return errors.New("LINK contract deployment failed")
	}

	report, err := t.gauntlet.ReadCommandReport()
	fmt.Println(report)
	if err != nil {
		fmt.Println(err)
		return errors.New("report not available")
	}

	linkAddress := report.Responses[0].Contract
	t.Account[LINK] = linkAddress

	msg := utils.LogStatus("Deployed LINK token")

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

func (t *Deployer) DeployOCR() error {
	msg := utils.LogStatus("Deployed OCR contract")

	fmt.Println("Deploying OCR Feed:")
	fmt.Println("Step 1: Init Requester Access Controller")
	err := t.gauntlet.ExecCommand(
		"access_controller:deploy",
		t.gauntlet.Flag("network", "bombay-testnet"),
	)
	if err != nil {
		return errors.New("Request AC initialization failed")
	}
	report, err := t.gauntlet.ReadCommandReport()
	if err != nil {
		return err
	}
	t.Account[RequesterAccessController] = report.Responses[0].Contract

	fmt.Println("Step 2: Init Billing Access Controller")
	err = t.gauntlet.ExecCommand(
		"access_controller:deploy",
		t.gauntlet.Flag("network", "bombay-testnet"),
	)
	if err != nil {
		return errors.New("Billing AC initialization failed")
	}
	report, err = t.gauntlet.ReadCommandReport()
	if err != nil {
		return err
	}
	t.Account[BillingAccessController] = report.Responses[0].Contract

	fmt.Println("Step 6: Init OCR 2 Feed")
	input := map[string]interface{}{
		"minAnswer":                 "0",
		"maxAnswer":                 "10000000000",
		"decimals":                  2,
		"description":               "Hello",
		"requesterAccessController": t.Account[RequesterAccessController],
		"billingAccessController":   t.Account[BillingAccessController],
		"linkToken":                 t.Account[LINK],
	}

	jsonInput, err := json.Marshal(input)
	if err != nil {
		return err
	}

	// TODO: command doesn't throw an error in go if it fails
	err = t.gauntlet.ExecCommand(
		"ocr2:deploy",
		t.gauntlet.Flag("network", "bombay-testnet"),
		t.gauntlet.Flag("input", string(jsonInput)),
	)
	if err != nil {
		return errors.New("feed initialization failed")
	}

	report, err = t.gauntlet.ReadCommandReport()
	if err != nil {
		return err
	}

	t.Account[OCR2] = report.Responses[0].Contract
	fmt.Printf(" - %s", report.Data["state"])
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

func (t Deployer) TransferLINK() error {
	msg := utils.LogStatus("Sending LINK to OCR contract")

	err := t.gauntlet.ExecCommand(
		"token:transfer",
		t.gauntlet.Flag("network", "bombay-testnet"),
		t.gauntlet.Flag("to", t.Account[OCR2]),
		t.gauntlet.Flag("amount", "1000000000"),
		t.gauntlet.Flag("link", t.Account[LINK]),
		t.Account[LINK],
	)
	if err != nil {
		return errors.New("LINK transfer failed")
	}

	return msg.Check(err)
}

const BeginProposal = "begin_proposal"

type ProposeConfig struct {
	ProposeConfig ProposeConfigDetails `json:"propose_config"`
}

type ProposeConfigDetails struct {
	ID            string   `json:"id"`
	Payees        []string `json:"payees"`
	Signers       []string `json:"signers"`
	Transmitters  []string `json:"transmitters"`
	F             uint8    `json:"f"`
	OnchainConfig []byte   `json:"onchain_config"`
}

type ProposeOffchainConfig struct {
	ProposeOffchainConfig ProposeOffchainConfigDetails `json:"propose_offchain_config"`
}

type ProposeOffchainConfigDetails struct {
	ID                    string `json:"id"`
	OffchainConfigVersion uint64 `json:"offchain_config_version"`
	OffchainConfig        []byte `json:"offchain_config"`
}

type ClearProposal struct {
	ClearProposal ClearProposalDetails `json:"clear_proposal"`
}

type ClearProposalDetails struct {
	ID string `json:"id"`
}

type FinalizeProposal struct {
	FinalizeProposal FinalizeProposalDetails `json:"finalize_proposal"`
}

type FinalizeProposalDetails struct {
	ID string `json:"id"`
}

type AcceptProposal struct {
	AcceptProposal AcceptProposalDetails `json:"accept_proposal"`
}

type AcceptProposalDetails struct {
	ID     string `json:"id"`
	Digest []byte `json:"digest"`
}

func (t Deployer) InitOCR(keys []opsChainlink.NodeKeys) (rerr error) {
	S := []int{}
	signersArray := []string{}
	transmitterArray := []string{}
	helperOracles := []confighelper.OracleIdentityExtra{}
	for _, k := range keys {
		S = append(S, 1)
		signersArray = append(signersArray, k.OCR2OnchainPublicKey)
		transmitterArray = append(transmitterArray, k.OCR2Transmitter)
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

	status := utils.LogStatus("InitOCR: set config test args")
	alphaPPB := uint64(1000000)
	_, _, f, onchainConfig, offchainConfigVersion, offchainConfig, err := confighelper.ContractSetConfigArgsForTests(
		2*time.Second,        // deltaProgress time.Duration,
		5*time.Second,        // deltaResend time.Duration,
		1*time.Second,        // deltaRound time.Duration,
		500*time.Millisecond, // deltaGrace time.Duration,
		10*time.Second,       // deltaStage time.Duration,
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
	if status.Check(err) != nil {
		return err
	}

	status = utils.LogStatus("InitOCR: begin proposal")
	err = t.gauntlet.ExecCommand(
		"ocr2:begin_proposal",
		t.gauntlet.Flag("network", "bombay-testnet"),
		t.Account[OCR2],
	)

	report, err := t.gauntlet.ReadCommandReport()
	if err != nil {
		return err
	}
	if err == nil && len(report.Data) == 0 {
		err = errors.New("begin proposal produced no logs")
	}

	if status.Check(err) != nil {
		return err
	}

	fmt.Printf(" - %s", report.Data["proposalId"])

	var id string = report.Data["proposalId"]

	// Be prepared to clear the proposal if incomplete.
	defer func() {
		if rerr == nil {
			return // Success
		}
		// Failure: Try to clean up incomplete proposal.
		status = utils.LogStatus("InitOCR: clear proposal: " + id)
		err = t.gauntlet.ExecCommand(
			"ocr2:clear_proposal",
			t.gauntlet.Flag("network", "bombay-testnet"),
			t.gauntlet.Flag("proposalId", id),
			t.Account[OCR2],
		)
		if status.Check(err) != nil {
			fmt.Println(err)
			return
		}
	}()

	jsonInput, err := json.Marshal(ProposeConfigDetails{
		ID:            id,
		Payees:        transmitterArray,
		Transmitters:  transmitterArray,
		F:             f,
		OnchainConfig: onchainConfig,
		Signers:       signersArray,
	})
	if err != nil {
		return err
	}

	status = utils.LogStatus("InitOCR: propose config " + id)
	err = t.gauntlet.ExecCommand(
		"ocr2:propose_config",
		t.gauntlet.Flag("network", "bombay-testnet"),
		t.gauntlet.Flag("input", string(jsonInput)),
		t.Account[OCR2],
	)
	if status.Check(err) != nil {
		return err
	}

	jsonInput, err = json.Marshal(ProposeOffchainConfig{
		ProposeOffchainConfig: ProposeOffchainConfigDetails{
			ID:                    id,
			OffchainConfigVersion: offchainConfigVersion,
			OffchainConfig:        offchainConfig,
		},
	})
	if err != nil {
		return err
	}
	status = utils.LogStatus("InitOCR: propose offchain config")
	err = t.gauntlet.ExecCommand(
		"ocr2:propose_offchain_config",
		t.gauntlet.Flag("network", "bombay-testnet"),
		t.gauntlet.Flag("proposalId", id),
		t.gauntlet.Flag("input", string(jsonInput)),
		t.Account[OCR2],
	)
	if status.Check(err) != nil {
		return err
	}

	status = utils.LogStatus("InitOCR: finalize proposal")
	err = t.gauntlet.ExecCommand(
		"ocr2:finalize_proposal",
		t.gauntlet.Flag("network", "bombay-testnet"),
		t.gauntlet.Flag("proposalId", id),
		t.Account[OCR2],
	)
	if status.Check(err) != nil {
		fmt.Println(err)
		return
	}

	report, err = t.gauntlet.ReadCommandReport()
	if err != nil {
		return err
	}
	if err == nil && len(report.Data) == 0 {
		err = errors.New("begin proposal produced no logs")
	}

	if status.Check(err) != nil {
		return err
	}

	fmt.Println(report.Data)
	var digest string = report.Data["digest"]

	switch len(digest) {
	case 0:
		return errors.New("failed to find event with attribute: wasm.digest")
	case 32:
		// expected
	default:
		return fmt.Errorf("wrong length for: wasm.digest: %d", len(digest))
	}

	input := map[string]interface{}{
		"digest": digest,
	}
	jsonInput, err = json.Marshal(input)
	if err != nil {
		return err
	}
	status = utils.LogStatus("InitOCR: accept proposal")
	err = t.gauntlet.ExecCommand(
		"ocr2:accept_proposal",
		t.gauntlet.Flag("network", t.network),
		t.gauntlet.Flag("proposalId", id),
		t.gauntlet.Flag("input", string(jsonInput)),
	)
	if status.Check(err) != nil {
		fmt.Println(err)
		return
	}

	return nil
}

func (t Deployer) OCR2Address() string {
	return t.Account[OCR2]
}

func (t Deployer) Addresses() map[int]string {
	return t.Account
}

func (t Deployer) Fund(addresses []string) error {
	// for _, a := range addresses {
	// 	msg := utils.LogStatus(fmt.Sprintf("Funded %s", a))
	// 	args := append([]string{"tx", "bank", "send", t.keyID, a, "1000000000uluna"}, t.args...)
	// 	if _, err := exec.Command("terrad", args...).Output(); msg.Check(err) != nil {
	// 		return err
	// 	}
	// }
	return nil
}
