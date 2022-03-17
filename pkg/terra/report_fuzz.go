//go:build gofuzz

package terra

func FuzzMedianFromReport(report []byte) int {
	cdc := ReportCodec{}
	_, err := cdc.MedianFromReport(report)
	if err != nil {
		return 0
	}
	return 1
}

