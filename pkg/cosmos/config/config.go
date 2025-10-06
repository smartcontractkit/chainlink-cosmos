package config

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"slices"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pelletier/go-toml/v2"
	"github.com/shopspring/decimal"

	"github.com/smartcontractkit/chainlink-common/pkg/config"
	"github.com/smartcontractkit/chainlink-common/pkg/config/configtest"

	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/db"
)

var defaults TOMLConfig

func init() {
	if err := configtest.DocDefaultsOnly(strings.NewReader(docsTOML), &defaults, config.DecodeTOML); err != nil {
		log.Fatalf("Failed to initialize defaults from docs: %v", err)
	}
}

func Defaults() (c TOMLConfig) {
	c.SetFrom(&defaults)
	return
}

type Config interface {
	Bech32Prefix() string
	BlockRate() time.Duration
	BlocksUntilTxTimeout() int64
	ConfirmPollPeriod() time.Duration
	FallbackGasPrice() sdk.Dec
	GasToken() string
	GasLimitMultiplier() float64
	MaxMsgsPerBatch() int64
	OCR2CachePollPeriod() time.Duration
	OCR2CacheTTL() time.Duration
	TxMsgTimeout() time.Duration
}

// opt: remove
type configSet struct {
	Bech32Prefix         string
	BlockRate            time.Duration
	BlocksUntilTxTimeout int64
	ConfirmPollPeriod    time.Duration
	FallbackGasPrice     sdk.Dec
	GasToken             string
	GasLimitMultiplier   float64
	MaxMsgsPerBatch      int64
	OCR2CachePollPeriod  time.Duration
	OCR2CacheTTL         time.Duration
	TxMsgTimeout         time.Duration
}

type Chain struct {
	Bech32Prefix         *string
	BlockRate            *config.Duration
	BlocksUntilTxTimeout *int64
	ConfirmPollPeriod    *config.Duration
	FallbackGasPrice     *decimal.Decimal
	GasToken             *string
	GasLimitMultiplier   *decimal.Decimal
	MaxMsgsPerBatch      *int64
	OCR2CachePollPeriod  *config.Duration
	OCR2CacheTTL         *config.Duration
	TxMsgTimeout         *config.Duration
}

type Node struct {
	Name          *string
	TendermintURL *config.URL
}

func (n *Node) ValidateConfig() (err error) {
	if n.Name == nil {
		err = errors.Join(err, config.ErrMissing{Name: "Name", Msg: "required for all nodes"})
	} else if *n.Name == "" {
		err = errors.Join(err, config.ErrEmpty{Name: "Name", Msg: "required for all nodes"})
	}
	if n.TendermintURL == nil {
		err = errors.Join(err, config.ErrMissing{Name: "TendermintURL", Msg: "required for all nodes"})
	}
	return
}

type TOMLConfigs []*TOMLConfig

func (cs TOMLConfigs) validateKeys() (err error) {
	// Unique chain IDs
	chainIDs := config.UniqueStrings{}
	for i, c := range cs {
		if chainIDs.IsDupe(c.ChainID) {
			err = errors.Join(err, config.NewErrDuplicate(fmt.Sprintf("%d.ChainID", i), *c.ChainID))
		}
	}

	// Unique node names
	names := config.UniqueStrings{}
	for i, c := range cs {
		for j, n := range c.Nodes {
			if names.IsDupe(n.Name) {
				err = errors.Join(err, config.NewErrDuplicate(fmt.Sprintf("%d.Nodes.%d.Name", i, j), *n.Name))
			}
		}
	}

	// Unique TendermintURLs
	urls := config.UniqueStrings{}
	for i, c := range cs {
		for j, n := range c.Nodes {
			u := (*url.URL)(n.TendermintURL)
			if urls.IsDupeFmt(u) {
				err = errors.Join(err, config.NewErrDuplicate(fmt.Sprintf("%d.Nodes.%d.TendermintURL", i, j), u.String()))
			}
		}
	}
	return
}

func (cs TOMLConfigs) ValidateConfig() (err error) {
	return cs.validateKeys()
}

func (cs *TOMLConfigs) SetFrom(fs *TOMLConfigs) (err error) {
	if err1 := fs.validateKeys(); err1 != nil {
		return err1
	}
	for _, f := range *fs {
		if f.ChainID == nil {
			*cs = append(*cs, f)
		} else if i := slices.IndexFunc(*cs, func(c *TOMLConfig) bool {
			return c.ChainID != nil && *c.ChainID == *f.ChainID
		}); i == -1 {
			*cs = append(*cs, f)
		} else {
			(*cs)[i].SetFrom(f)
		}
	}
	return
}

type Nodes []*Node

func (ns *Nodes) SetFrom(fs *Nodes) {
	for _, f := range *fs {
		if f.Name == nil {
			*ns = append(*ns, f)
		} else if i := slices.IndexFunc(*ns, func(n *Node) bool {
			return n.Name != nil && *n.Name == *f.Name
		}); i == -1 {
			*ns = append(*ns, f)
		} else {
			setFromNode((*ns)[i], f)
		}
	}
}

func setFromNode(n, f *Node) {
	if f.Name != nil {
		n.Name = f.Name
	}
	if f.TendermintURL != nil {
		n.TendermintURL = f.TendermintURL
	}
}

func legacyNode(n *Node, id string) db.Node {
	return db.Node{
		Name:          *n.Name,
		CosmosChainID: id,
		TendermintURL: (*url.URL)(n.TendermintURL).String(),
	}
}

type TOMLConfig struct {
	ChainID *string
	// Do not access directly. Use [IsEnabled]
	Enabled *bool
	Chain
	Nodes Nodes
}

func (c *TOMLConfig) IsEnabled() bool {
	return c.Enabled == nil || *c.Enabled
}

func (c *TOMLConfig) SetDefaults() {
	def := Defaults()
	def.SetFrom(c)
	*c = def

}

func (c *TOMLConfig) SetFrom(f *TOMLConfig) {
	if f.ChainID != nil {
		c.ChainID = f.ChainID
	}
	if f.Enabled != nil {
		c.Enabled = f.Enabled
	}
	setFromChain(&c.Chain, &f.Chain)
	c.Nodes.SetFrom(&f.Nodes)
}

func setFromChain(c, f *Chain) {
	if f.Bech32Prefix != nil {
		c.Bech32Prefix = f.Bech32Prefix
	}
	if f.BlockRate != nil {
		c.BlockRate = f.BlockRate
	}
	if f.BlocksUntilTxTimeout != nil {
		c.BlocksUntilTxTimeout = f.BlocksUntilTxTimeout
	}
	if f.ConfirmPollPeriod != nil {
		c.ConfirmPollPeriod = f.ConfirmPollPeriod
	}
	if f.FallbackGasPrice != nil {
		c.FallbackGasPrice = f.FallbackGasPrice
	}
	if f.GasToken != nil {
		c.GasToken = f.GasToken
	}
	if f.GasLimitMultiplier != nil {
		c.GasLimitMultiplier = f.GasLimitMultiplier
	}
	if f.MaxMsgsPerBatch != nil {
		c.MaxMsgsPerBatch = f.MaxMsgsPerBatch
	}
	if f.OCR2CachePollPeriod != nil {
		c.OCR2CachePollPeriod = f.OCR2CachePollPeriod
	}
	if f.OCR2CacheTTL != nil {
		c.OCR2CacheTTL = f.OCR2CacheTTL
	}
	if f.TxMsgTimeout != nil {
		c.TxMsgTimeout = f.TxMsgTimeout
	}
}

func (c *TOMLConfig) ValidateConfig() (err error) {
	if c.ChainID == nil {
		err = errors.Join(err, config.ErrMissing{Name: "ChainID", Msg: "required for all chains"})
	} else if *c.ChainID == "" {
		err = errors.Join(err, config.ErrEmpty{Name: "ChainID", Msg: "required for all chains"})
	}

	if len(c.Nodes) == 0 {
		err = errors.Join(err, config.ErrMissing{Name: "Nodes", Msg: "must have at least one node"})
	}

	return
}

func (c *TOMLConfig) TOMLString() (string, error) {
	b, err := toml.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

var _ Config = &TOMLConfig{}

func (c *TOMLConfig) Bech32Prefix() string {
	return *c.Chain.Bech32Prefix
}

func (c *TOMLConfig) BlockRate() time.Duration {
	return c.Chain.BlockRate.Duration()
}

func (c *TOMLConfig) BlocksUntilTxTimeout() int64 {
	return *c.Chain.BlocksUntilTxTimeout
}

func (c *TOMLConfig) ConfirmPollPeriod() time.Duration {
	return c.Chain.ConfirmPollPeriod.Duration()
}

func (c *TOMLConfig) FallbackGasPrice() sdk.Dec {
	return sdkDecFromDecimal(c.Chain.FallbackGasPrice)
}

func (c *TOMLConfig) GasToken() string {
	return *c.Chain.GasToken
}

func (c *TOMLConfig) GasLimitMultiplier() float64 {
	return c.Chain.GasLimitMultiplier.InexactFloat64()
}

func (c *TOMLConfig) MaxMsgsPerBatch() int64 {
	return *c.Chain.MaxMsgsPerBatch
}

func (c *TOMLConfig) OCR2CachePollPeriod() time.Duration {
	return c.Chain.OCR2CachePollPeriod.Duration()
}

func (c *TOMLConfig) OCR2CacheTTL() time.Duration {
	return c.Chain.OCR2CacheTTL.Duration()
}

func (c *TOMLConfig) TxMsgTimeout() time.Duration {
	return c.Chain.TxMsgTimeout.Duration()
}

func sdkDecFromDecimal(d *decimal.Decimal) sdk.Dec {
	i := d.Shift(sdk.Precision)
	return sdk.NewDecFromBigIntWithPrec(i.BigInt(), sdk.Precision)
}

func (c *TOMLConfig) GetNode(name string) (db.Node, error) {
	for _, n := range c.Nodes {
		if *n.Name == name {
			return legacyNode(n, *c.ChainID), nil
		}
	}
	return db.Node{}, fmt.Errorf("node not found")
}

func (c *TOMLConfig) ListNodes() ([]db.Node, error) {
	var allNodes []db.Node
	for _, n := range c.Nodes {
		allNodes = append(allNodes, legacyNode(n, *c.ChainID))
	}
	return allNodes, nil
}
