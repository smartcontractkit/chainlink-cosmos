package terra

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"math/big"
	"testing"
	"time"

	"github.com/smartcontractkit/libocr/commontypes"
	"github.com/smartcontractkit/libocr/offchainreporting2/reportingplugin/median"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	totalLen := prefixSizeBytes + observationSizeBytes*n + juelsPerFeeCoinSizeBytes
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
		index := prefixSizeBytes + observationSizeBytes*i
		assert.Equal(t, oo[0].Value.FillBytes(make([]byte, observationSizeBytes)), []byte(report[index:index+observationSizeBytes]), fmt.Sprintf("validate median observation #%d", i))
	}

	// validate juelsToEth
	assert.Equal(t, v.FillBytes(make([]byte, juelsPerFeeCoinSizeBytes)), []byte(report[totalLen-juelsPerFeeCoinSizeBytes:totalLen]), "validate juelsToEth")
}

func TestMedianFromOnChainReport(t *testing.T) {
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

func TestMedianFromReport(t *testing.T) {
	var tt = []struct{
		name string
		obs []*big.Int
		expectedMedian *big.Int
	}{
		{
			name: "2 positive one zero",
			obs: []*big.Int{big.NewInt(0), big.NewInt(10), big.NewInt(20)},
			expectedMedian: big.NewInt(10),
		},
		{
			name: "one zero",
			obs: []*big.Int{big.NewInt(0)},
			expectedMedian: big.NewInt(0),
		},
		{
			name: "two equal",
			obs: []*big.Int{big.NewInt(1), big.NewInt(1)},
			expectedMedian: big.NewInt(1),
		},
		{
			name: "one negative one positive",
			obs: []*big.Int{big.NewInt(-1), big.NewInt(1)},
			// sorts to -1, 1
			expectedMedian: big.NewInt(1),
		},
		{
			name: "two negative",
			obs: []*big.Int{big.NewInt(-2), big.NewInt(-1)},
			// will sort to -2, -1
			expectedMedian: big.NewInt(-1),
		},
		{
			name: "three negative",
			obs: []*big.Int{big.NewInt(-5), big.NewInt(-3), big.NewInt(-1)},
			// will sort to -5, -3, -1
			expectedMedian: big.NewInt(-3),
		},
	}
	for _, tc := range tt {
		tc := tc
		cdc := ReportCodec{}
		t.Run(tc.name, func(t *testing.T) {
			var pos []median.ParsedAttributedObservation
			for i, obs := range tc.obs {
				pos = append(pos, median.ParsedAttributedObservation{
					Value:           obs,
					JuelsPerFeeCoin: obs,
					Observer:        commontypes.OracleID(uint8(i))},
				)
			}
			report, err := cdc.BuildReport(pos)
			require.NoError(t, err)
			med, err := cdc.MedianFromReport(report)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedMedian.String(), med.String())
		})
	}
}

func TestFuzzedReportInputs(t *testing.T) {
	// To build corpus
	// go get -u github.com/dvyukov/go-fuzz/go-fuzz@latest github.com/dvyukov/go-fuzz/go-fuzz-build@latest
	// go-fuzz -bin=./terra-fuzz.zip -procs=10
	files, err := ioutil.ReadDir("corpus")
	require.NoError(t, err)
	for _, s := range files {
		b, err := ioutil.ReadFile(fmt.Sprintf("corpus/%s", s.Name()))
		require.NoError(t, err)
		c := ReportCodec{}
		res, err := c.MedianFromReport(b)
		t.Log(s.Name(), res, err)
	}
}

// TODO: TestHashReport - part of Solana report test suite
