package terra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	cosmosSDK "github.com/cosmos/cosmos-sdk/types"
	terraSDK "github.com/terra-money/core/x/wasm/types"

	"github.com/smartcontractkit/chainlink/core/utils"
	"github.com/smartcontractkit/libocr/offchainreporting2/chains/evmutil"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"

	"github.com/smartcontractkit/chainlink-terra/pkg/terra/client"
)

var _ types.ContractTransmitter = (*ContractTransmitter)(nil)

type ContractTransmitter struct {
	utils.StartStopOnce
	msgEnqueuer MsgEnqueuer
	chainReader client.Reader
	lggr        Logger
	jobID       string
	contract    cosmosSDK.AccAddress
	sender      cosmosSDK.AccAddress
	cfg         Config
	stop, done  chan struct{}

	// cached state
	mu           sync.RWMutex
	ts           *time.Time
	configDigest types.ConfigDigest
	epoch        uint32
}

func NewContractTransmitter(jobID string,
	contract cosmosSDK.AccAddress,
	sender cosmosSDK.AccAddress,
	msgEnqueuer MsgEnqueuer,
	chainReader client.Reader,
	lggr Logger,
	cfg Config,
) *ContractTransmitter {
	return &ContractTransmitter{
		jobID:       jobID,
		contract:    contract,
		msgEnqueuer: msgEnqueuer,
		sender:      sender,
		chainReader: chainReader,
		lggr:        lggr,
		cfg:         cfg,
		stop:        make(chan struct{}),
		done:        make(chan struct{}),
	}
}

func (ct *ContractTransmitter) Start() error {
	return ct.StartOnce("TerraContractTransmitter", func() error {
		ct.lggr.Debugf("Starting")
		go ct.pollState()
		return nil
	})
}

func (ct *ContractTransmitter) Close() error {
	return ct.StopOnce("TerraContractTransmitter", func() error {
		ct.lggr.Debugf("Stopping")
		close(ct.stop)
		<-ct.done
		return nil
	})
}

func (ct *ContractTransmitter) pollState() {
	defer close(ct.done)
	tick := time.After(0)
	for {
		select {
		case <-ct.stop:
			return
		case <-tick:
			ctx, cancel := utils.ContextFromChan(ct.stop)
			done := make(chan struct{})
			go func() {
				defer close(done)
				configDigest, epoch, err := ct.latestConfigDigestAndEpoch(ctx)
				if err != nil {
					ct.lggr.Errorf("Failed to get latest config digest and epoch", "err", err)
					return
				}
				now := time.Now()
				ct.mu.Lock()
				ct.ts = &now
				ct.configDigest = configDigest
				ct.epoch = epoch
				ct.mu.Unlock()
			}()
			select {
			case <-ct.stop:
				cancel()
				// Note: the client does not respect context, so just return instead of waiting.
				// <-done
				return
			case <-done:
				tick = time.After(utils.WithJitter(ct.cfg.ConfirmPollPeriod()))
			}
		}
	}
}

// Transmit signs and sends the report
func (ct *ContractTransmitter) Transmit(
	ctx context.Context,
	reportCtx types.ReportContext,
	report types.Report,
	sigs []types.AttributedOnchainSignature,
) error {
	ct.lggr.Infof("[%s] Sending TX to %s", ct.jobID, ct.contract.String())
	msgStruct := TransmitMsg{}
	reportContext := evmutil.RawReportContext(reportCtx)
	for _, r := range reportContext {
		msgStruct.Transmit.ReportContext = append(msgStruct.Transmit.ReportContext, r[:]...)
	}
	msgStruct.Transmit.Report = []byte(report)
	for _, sig := range sigs {
		msgStruct.Transmit.Signatures = append(msgStruct.Transmit.Signatures, sig.Signature)
	}
	msgBytes, err := json.Marshal(msgStruct)
	if err != nil {
		return err
	}
	m := terraSDK.NewMsgExecuteContract(ct.sender, ct.contract, msgBytes, cosmosSDK.Coins{})
	d, err := m.Marshal()
	if err != nil {
		return err
	}
	_, err = ct.msgEnqueuer.Enqueue(ct.contract.String(), d)
	return err
}

// LatestConfigDigestAndEpoch fetches the latest details from cache
func (ct *ContractTransmitter) LatestConfigDigestAndEpoch(ctx context.Context) (
	configDigest types.ConfigDigest,
	epoch uint32,
	err error,
) {
	ct.mu.RLock()
	ts := ct.ts
	configDigest = ct.configDigest
	epoch = ct.epoch
	ct.mu.RUnlock()
	if ts == nil {
		err = errors.New("config digest and epoch not yet initialized")
	} else if since := time.Since(*ts); since > ct.cfg.OCRCacheTTL() {
		err = fmt.Errorf("failed to get config digest and epoch: stale value cached %s ago", since)
	}
	return
}

// latestConfigDigestAndEpoch fetches the latest details from address state
func (ct *ContractTransmitter) latestConfigDigestAndEpoch(ctx context.Context) (
	configDigest types.ConfigDigest,
	epoch uint32,
	err error,
) {
	resp, err := ct.chainReader.ContractStore(
		ct.contract, []byte(`"latest_config_digest_and_epoch"`),
	)
	if err != nil {
		return types.ConfigDigest{}, 0, err
	}

	var digest LatestConfigDigestAndEpoch
	if err := json.Unmarshal(resp, &digest); err != nil {
		return types.ConfigDigest{}, 0, err
	}

	return digest.ConfigDigest, digest.Epoch, nil
}

func (ct *ContractTransmitter) FromAccount() types.Account {
	return types.Account(ct.sender.String())
}
