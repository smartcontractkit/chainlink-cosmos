package terra

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/smartcontractkit/libocr/commontypes"
	"github.com/smartcontractkit/libocr/offchainreporting2/reportingplugin/median"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"github.com/stretchr/testify/assert"
)

func TestBuildReport(t *testing.T) {
	c := ReportCodec{}
	oo := []median.ParsedAttributedObservation{}

	// expected outputs
	n := 4
	observers := make([]byte, 32)
	v := big.NewInt(0)
	v.SetString("1000000000000000000", 10)

	for i := 0; i < n; i++ {
		oo = append(oo, median.ParsedAttributedObservation{
			Timestamp:       uint32(time.Now().Unix()),
			Value:           big.NewInt(1234567890),
			JuelsPerFeeCoin: v,
			Observer:        commontypes.OracleID(i),
		})

		// create expected outputs
		observers[i] = uint8(i)
	}

	report, err := c.BuildReport(oo)
	assert.NoError(t, err)

	// validate length
	totalLen := PrefixSizeBytes + ObservationSizeBytes*n + JuelsPerFeeCoinSizeBytes
	assert.Equal(t, totalLen, len(report), "validate length")

	// validate timestamp
	assert.Equal(t, oo[0].Timestamp, binary.BigEndian.Uint32(report[0:4]), "validate timestamp")

	// validate observers
	index := 4
	assert.Equal(t, observers, []byte(report[index:index+32]), "validate observers")

	// validate observer count
	assert.Equal(t, uint8(n), report[36], "validate observer count")

	// validate observations
	for i := 0; i < n; i++ {
		index := PrefixSizeBytes + ObservationSizeBytes*i
		assert.Equal(t, oo[0].Value.FillBytes(make([]byte, ObservationSizeBytes)), []byte(report[index:index+ObservationSizeBytes]), fmt.Sprintf("validate median observation #%d", i))
	}

	// validate juelsToEth
	assert.Equal(t, v.FillBytes(make([]byte, JuelsPerFeeCoinSizeBytes)), []byte(report[totalLen-JuelsPerFeeCoinSizeBytes:totalLen]), "validate juelsToEth")
}

func TestMedianFromReport(t *testing.T) {
	c := ReportCodec{}

	report := types.Report{
		97, 91, 43, 83, // observations_timestamp
		0, 1, 2, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // observers
		2,                                                   // len
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 73, 150, 2, 210, // observation 1
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 73, 150, 2, 210, // observation 2
		0, 0, 0, 0, 0, 0, 0, 0, 13, 224, 182, 179, 167, 100, 0, 0, // juels per luna (1 with 18 decimal places)
	}
	res, err := c.MedianFromReport(report)
	assert.NoError(t, err)
	assert.Equal(t, "1234567890", res.String())
}

// TODO: TestHashReport - part of Solana report test suite
