package config

import (
	"fmt"
	"net/url"

	"github.com/shopspring/decimal"
	"go.uber.org/multierr"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/smartcontractkit/chainlink-relay/pkg/config"
	"github.com/smartcontractkit/chainlink-relay/pkg/utils"

	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos"
	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/db"
)

type Chain struct {
	BlockRate             *utils.Duration
	BlocksUntilTxTimeout  *int64
	ConfirmPollPeriod     *utils.Duration
	FallbackGasPriceUAtom *decimal.Decimal
	FCDURL                *utils.URL
	GasLimitMultiplier    *decimal.Decimal
	MaxMsgsPerBatch       *int64
	OCR2CachePollPeriod   *utils.Duration
	OCR2CacheTTL          *utils.Duration
	TxMsgTimeout          *utils.Duration
}

func (c *Chain) SetFromDB(cfg *db.ChainCfg) error {
	if cfg == nil {
		return nil
	}
	if cfg.BlockRate != nil {
		c.BlockRate = utils.MustNewDuration(cfg.BlockRate.Duration())
	}
	if cfg.BlocksUntilTxTimeout.Valid {
		c.BlocksUntilTxTimeout = &cfg.BlocksUntilTxTimeout.Int64
	}
	if cfg.ConfirmPollPeriod != nil {
		c.ConfirmPollPeriod = utils.MustNewDuration(cfg.ConfirmPollPeriod.Duration())
	}
	if cfg.FallbackGasPriceUAtom.Valid {
		s := cfg.FallbackGasPriceUAtom.String
		d, err := decimal.NewFromString(s)
		if err != nil {
			return fmt.Errorf("invalid decimal FallbackGasPriceUAtom: %s", s)
		}
		c.FallbackGasPriceUAtom = &d
	}
	if cfg.FCDURL.Valid {
		s := cfg.FCDURL.String
		d, err := url.Parse(s)
		if err != nil {
			return fmt.Errorf("invalid FCDURL: %s", s)
		}
		c.FCDURL = (*utils.URL)(d)
	}
	if cfg.GasLimitMultiplier.Valid {
		d := decimal.NewFromFloat(cfg.GasLimitMultiplier.Float64)
		c.GasLimitMultiplier = &d
	}
	if cfg.MaxMsgsPerBatch.Valid {
		c.MaxMsgsPerBatch = &cfg.MaxMsgsPerBatch.Int64
	}
	if cfg.OCR2CachePollPeriod != nil {
		c.OCR2CachePollPeriod = utils.MustNewDuration(cfg.OCR2CachePollPeriod.Duration())
	}
	if cfg.OCR2CacheTTL != nil {
		c.OCR2CacheTTL = utils.MustNewDuration(cfg.OCR2CacheTTL.Duration())
	}
	if cfg.TxMsgTimeout != nil {
		c.TxMsgTimeout = utils.MustNewDuration(cfg.TxMsgTimeout.Duration())
	}
	return nil
}

func (c *Chain) SetDefaults() {
	if c.BlockRate == nil {
		c.BlockRate = utils.MustNewDuration(cosmos.DefaultConfigSet.BlockRate)
	}
	if c.BlocksUntilTxTimeout == nil {
		c.BlocksUntilTxTimeout = &cosmos.DefaultConfigSet.BlocksUntilTxTimeout
	}
	if c.ConfirmPollPeriod == nil {
		c.ConfirmPollPeriod = utils.MustNewDuration(cosmos.DefaultConfigSet.ConfirmPollPeriod)
	}
	if c.FallbackGasPriceUAtom == nil {
		d := decimal.NewFromBigInt(cosmos.DefaultConfigSet.FallbackGasPriceUAtom.BigInt(), -sdk.Precision)
		c.FallbackGasPriceUAtom = &d
	}
	if c.FCDURL == nil {
		c.FCDURL = (*utils.URL)(&cosmos.DefaultConfigSet.FCDURL)
	}
	if c.GasLimitMultiplier == nil {
		d := decimal.NewFromFloat(cosmos.DefaultConfigSet.GasLimitMultiplier)
		c.GasLimitMultiplier = &d
	}
	if c.MaxMsgsPerBatch == nil {
		c.MaxMsgsPerBatch = &cosmos.DefaultConfigSet.MaxMsgsPerBatch
	}
	if c.OCR2CachePollPeriod == nil {
		c.OCR2CachePollPeriod = utils.MustNewDuration(cosmos.DefaultConfigSet.OCR2CachePollPeriod)
	}
	if c.OCR2CacheTTL == nil {
		c.OCR2CacheTTL = utils.MustNewDuration(cosmos.DefaultConfigSet.OCR2CacheTTL)
	}
	if c.TxMsgTimeout == nil {
		c.TxMsgTimeout = utils.MustNewDuration(cosmos.DefaultConfigSet.TxMsgTimeout)
	}
}

type Node struct {
	Name          *string
	TendermintURL *utils.URL
}

func (n *Node) SetFromDB(db db.Node) error {
	if db.Name != "" {
		n.Name = &db.Name
	}
	if db.TendermintURL != "" {
		u, err := url.Parse(db.TendermintURL)
		if err != nil {
			return err
		}
		n.TendermintURL = (*utils.URL)(u)
	}
	return nil
}

func (n *Node) ValidateConfig() (err error) {
	if n.Name == nil {
		err = multierr.Append(err, config.ErrMissing{Name: "Name", Msg: "required for all nodes"})
	} else if *n.Name == "" {
		err = multierr.Append(err, config.ErrEmpty{Name: "Name", Msg: "required for all nodes"})
	}
	if n.TendermintURL == nil {
		err = multierr.Append(err, config.ErrMissing{Name: "TendermintURL", Msg: "required for all nodes"})
	}
	return
}
