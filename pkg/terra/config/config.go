package config

import (
	"net/url"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"go.uber.org/multierr"

	"github.com/smartcontractkit/chainlink-relay/pkg/config"
	"github.com/smartcontractkit/chainlink-relay/pkg/utils"

	"github.com/smartcontractkit/chainlink-terra/pkg/terra/db"
)

type Chain struct {
	BlockRate             *utils.Duration
	BlocksUntilTxTimeout  *int64
	ConfirmPollPeriod     *utils.Duration
	FallbackGasPriceULuna *decimal.Decimal
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
	if cfg.FallbackGasPriceULuna.Valid {
		s := cfg.FallbackGasPriceULuna.String
		d, err := decimal.NewFromString(s)
		if err != nil {
			return errors.Wrapf(err, "invalid decimal FallbackGasPriceULuna: %s", s)
		}
		c.FallbackGasPriceULuna = &d
	}
	if cfg.FCDURL.Valid {
		s := cfg.FCDURL.String
		d, err := url.Parse(s)
		if err != nil {
			return errors.Wrapf(err, "invalid FCDURL: %s", s)
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
