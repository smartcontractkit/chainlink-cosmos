package terra

import (
	"fmt"
	"github.com/smartcontractkit/chainlink-terra/pkg/terra/client"
	"github.com/smartcontractkit/chainlink-terra/pkg/terra/db"
	"github.com/smartcontractkit/terra.go/msg"
	"strings"

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
	ReportContext ByteArray      `json:"report_context"`
	Report        ByteArray      `json:"report"`
	Signatures    ByteArrayArray `json:"signatures"`
}

// ByteArrayArray and ByteArray implement custom unmarshalling because go unmarshals []byte and []uint8 to strings
type ByteArrayArray [][]byte
type ByteArray []byte

func (b ByteArray) MarshalJSON() ([]byte, error) {
	return unmarshalByteArrays(b)
}

func (b ByteArrayArray) MarshalJSON() ([]byte, error) {
	return unmarshalByteArrays(b)
}

func unmarshalByteArrays(b interface{}) ([]byte, error) {
	var result string
	if b == nil {
		result = "null"
	} else {
		result = strings.Join(strings.Fields(fmt.Sprintf("%d", b)), ",") // prints a number array in string form
	}
	return []byte(result), nil
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
