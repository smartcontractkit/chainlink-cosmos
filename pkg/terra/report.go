package terra

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"sort"

	"github.com/smartcontractkit/libocr/offchainreporting2/reportingplugin/median"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
)

var _ median.ReportCodec = (*ReportCodec)(nil)

type ReportCodec struct{}

func (c ReportCodec) BuildReport(oo []median.ParsedAttributedObservation) (types.Report, error) {
	n := len(oo)
	if n == 0 {
		return nil, fmt.Errorf("cannot build report from empty attributed observations")
	}

	// copy so we can safely re-order subsequently
	oo = append([]median.ParsedAttributedObservation{}, oo...)

	// get median timestamp
	sort.Slice(oo, func(i, j int) bool {
		return oo[i].Timestamp < oo[j].Timestamp
	})
	timestamp := oo[n/2].Timestamp

	// get median juelsPerFeeCoin
	sort.Slice(oo, func(i, j int) bool {
		return oo[i].JuelsPerFeeCoin.Cmp(oo[j].JuelsPerFeeCoin) < 0
	})
	juelsPerFeeCoin := oo[n/2].JuelsPerFeeCoin

	// sort by values
	sort.Slice(oo, func(i, j int) bool {
		return oo[i].Value.Cmp(oo[j].Value) < 0
	})

	observers := [32]byte{}
	observations := []*big.Int{}

	for i, o := range oo {
		observers[i] = byte(o.Observer)
		observations = append(observations, o.Value)
	}

	// encoding
	report := []byte{}

	time := make([]byte, 4)
	binary.BigEndian.PutUint32(time, timestamp)
	report = append(report, time[:]...)

	report = append(report, observers[:]...)
	report = append(report, byte(len(observations)))

	for _, o := range observations {
		obs, err := NewObservationFromInt(o)
		if err != nil {
			return nil, err
		}
		report = append(report, obs[:]...)
	}

	jBytes := make([]byte, JuelsPerFeeCoinSizeBytes)
	report = append(report, juelsPerFeeCoin.FillBytes(jBytes)[:]...)

	return report, nil
}

func (c ReportCodec) MedianFromReport(report types.Report) (*big.Int, error) {
	// report should at least be able to contain timestamp, observers, observations length
	rLen := len(report)
	if rLen < PrefixSizeBytes {
		return nil, fmt.Errorf("report length missmatch: %d (received), %d (expected)", rLen, PrefixSizeBytes)
	}

	n := int(report[4+32])
	if n == 0 {
		return nil, fmt.Errorf("unpacked report has no 'observations'")
	}

	if rLen < PrefixSizeBytes+(ObservationSizeBytes*n)+JuelsPerFeeCoinSizeBytes {
		return nil, fmt.Errorf("report does not contain enough observations or is missing juels/feeCoin observation")
	}

	// unpack observations
	var observations []*big.Int
	for i := 0; i < n; i++ {
		start := PrefixSizeBytes + ObservationSizeBytes*i
		end := start + ObservationSizeBytes
		o, err := Observation(report[start:end]).ToInt()
		if err != nil {
			return nil, err
		}
		observations = append(observations, o)
	}

	// Returns the "median" (the n//2-th ranked element to be more precise where n
	// is the length of the list) observation from the report.
	return observations[n/2], nil
}
