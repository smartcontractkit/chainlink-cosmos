package terra

import (
	"github.com/smartcontractkit/chainlink-terra/pkg/terra/client"
	"github.com/smartcontractkit/chainlink-terra/pkg/terra/db"
	"github.com/smartcontractkit/terra.go/msg"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
)

const (
	// Report data
	PrefixLen = 4 + 32 + 1
	MedianLen = 16
	JuelsLen  = 16
)

type TransmitMsg struct {
	Transmit TransmitPayload `json:"transmit"`
}

type TransmitPayload struct {
	ReportContext []byte   `json:"report_context"`
	Report        []byte   `json:"report"`
	Signatures    [][]byte `json:"signatures"`
}

type ConfigDetails struct {
	BlockNumber  uint64             `json:"block_number"`
	ConfigDigest types.ConfigDigest `json:"config_digest"`
}

type LatestTransmissionDetails struct {
	LatestConfigDigest types.ConfigDigest `json:"latest_config_digest"`
	Epoch              uint32             `json:"epoch"`
	Round              uint8              `json:"round"`
	LatestAnswer       string             `json:"latest_answer"`
	LatestTimestamp    int64              `json:"latest_timestamp"`
}

type LatestConfigDigestAndEpoch struct {
	ConfigDigest types.ConfigDigest `json:"config_digest"`
	Epoch        uint32             `json:"epoch"`
}

type Msg struct {
	db.Msg

	// In memory only
	ExecuteContract *msg.ExecuteContract
}

type Msgs []Msg

func (tms Msgs) GetSimMsgs() client.SimMsgs {
	var msgs []client.SimMsg
	for i := range tms {
		msgs = append(msgs, client.SimMsg{
			ID:  tms[i].ID,
			Msg: tms[i].ExecuteContract,
		})
	}
	return msgs
}

func (tms Msgs) GetIDs() []int64 {
	ids := make([]int64, len(tms))
	for i := range tms {
		ids[i] = tms[i].ID
	}
	return ids
}
