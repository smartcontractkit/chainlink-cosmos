//go:generate protoc -I. --gocosmos_out=. ./report.proto

package median_report

import (
	"errors"
	fmt "fmt"
	"math/big"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	proto "github.com/gogo/protobuf/proto"

	"github.com/smartcontractkit/libocr/offchainreporting2/reportingplugin/median"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
)

var _ median.ReportCodec = ReportCodec{}

type ReportCodec struct{}

func (ReportCodec) BuildReport(observations []median.ParsedAttributedObservation) (types.Report, error) {
	if len(observations) == 0 {
		err := errors.New("cannot build report from empty attributed observations")
		return nil, err
	}

	// copy so we can safely re-order subsequently
	observations = append([]median.ParsedAttributedObservation{}, observations...)

	// get median timestamp
	sort.Slice(observations, func(i, j int) bool {
		return observations[i].Timestamp < observations[j].Timestamp
	})

	timestamp := observations[len(observations)/2].Timestamp

	// sort by values
	sort.Slice(observations, func(i, j int) bool {
		return observations[i].Value.Cmp(observations[j].Value) < 0
	})

	reportToPack := &Report{
		ObservationsTimestamp: int64(timestamp),
		Observers:             make([]byte, 0, len(observations)),
		Observations:          make([]sdk.Dec, 0, len(observations)),
	}

	for _, observation := range observations {
		reportToPack.Observers = append(reportToPack.Observers, byte(observation.Observer))
		reportToPack.Observations = append(reportToPack.Observations, sdk.NewDecFromBigInt(observation.Value))
	}

	reportBytes, err := proto.Marshal(reportToPack)
	if err != nil {
		err = fmt.Errorf("failed to marshal MedianObservation message: %w", err)
		return nil, err
	}

	return types.Report(reportBytes), err
}

func (ReportCodec) MaxReportLength(n int) int {
	// TODO:
	return 0
	// return prefixSizeBytes + (n * observationSizeBytes) + juelsPerFeeCoinSizeBytes
}

func (ReportCodec) MedianFromReport(report types.Report) (*big.Int, error) {
	var reportRaw Report

	if err := proto.Unmarshal([]byte(report), &reportRaw); err != nil {
		err = fmt.Errorf("failed to unmarshal data as median_report.Report: %w", err)
		return nil, err
	}

	if len(reportRaw.Observations) == 0 {
		err := errors.New("empty observations set in report")
		return nil, err
	}

	median := reportRaw.Observations[len(reportRaw.Observations)/2].BigInt()

	return median, nil
}

func (ReportCodec) ParseReport(data []byte) (*Report, error) {
	var reportRaw Report

	if err := proto.Unmarshal(data, &reportRaw); err != nil {
		err = fmt.Errorf("failed to unmarshal data as median_report.Report: %w", err)
		return nil, err
	}

	if len(reportRaw.Observations) == 0 {
		err := errors.New("empty observations set in report")
		return nil, err
	}

	return &reportRaw, nil
}
