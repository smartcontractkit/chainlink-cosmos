package chaos_test

import (
	"testing"

	"github.com/smartcontractkit/chainlink-terra/tests/e2e/utils"

	. "github.com/onsi/ginkgo/v2"
)

func Test_Suite(t *testing.T) {
	utils.GinkgoSuite()
	RunSpecs(t, "Chaos")
}
