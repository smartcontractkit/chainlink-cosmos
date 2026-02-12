package cosmwasm

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
)

// JSONConfigDigest supports both CosmWasm's [u8;32] JSON arrays and hex strings.
type JSONConfigDigest types.ConfigDigest

func (d *JSONConfigDigest) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return fmt.Errorf("cannot unmarshal empty config digest")
	}

	// CosmWasm query responses return digests as a fixed-size byte array.
	if trimmed[0] == '[' {
		var raw []uint8
		if err := json.Unmarshal(trimmed, &raw); err != nil {
			return err
		}
		digest, err := types.BytesToConfigDigest(raw)
		if err != nil {
			return err
		}
		*d = JSONConfigDigest(digest)
		return nil
	}

	// Keep compatibility with any string-encoded digest formats.
	var digest types.ConfigDigest
	if err := json.Unmarshal(trimmed, &digest); err != nil {
		return err
	}
	*d = JSONConfigDigest(digest)
	return nil
}

func (d JSONConfigDigest) ConfigDigest() types.ConfigDigest {
	return types.ConfigDigest(d)
}

type TransmitMsg struct {
	Transmit TransmitPayload `json:"transmit"`
}

type TransmitPayload struct {
	ReportContext []byte   `json:"report_context"`
	Report        []byte   `json:"report"`
	Signatures    [][]byte `json:"signatures"`
}

type ConfigDetails struct {
	BlockNumber  uint64           `json:"block_number"`
	ConfigDigest JSONConfigDigest `json:"config_digest"`
}

type LatestTransmissionDetails struct {
	LatestConfigDigest JSONConfigDigest `json:"latest_config_digest"`
	Epoch              uint32           `json:"epoch"`
	Round              uint8            `json:"round"`
	LatestAnswer       string           `json:"latest_answer"`
	LatestTimestamp    int64            `json:"latest_timestamp"`
}

type LatestConfigDigestAndEpoch struct {
	ConfigDigest JSONConfigDigest `json:"config_digest"`
	Epoch        uint32           `json:"epoch"`
}
