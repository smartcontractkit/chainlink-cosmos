package smoke_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"

	"github.com/smartcontractkit/chainlink/integration-tests/actions"

	"github.com/smartcontractkit/chainlink-cosmos/integration-tests/common"
	"github.com/smartcontractkit/chainlink-cosmos/ops/gauntlet"
	"github.com/smartcontractkit/chainlink-cosmos/ops/utils"
)

var (
	err error
)

func TestOCRBasic(t *testing.T) {
	testState := &common.Test{
		T: t,
	}
	testState.Common = common.New()
	testState.Common.Default(t)
	// Setting this to the root of the repo for cmd exec func for Gauntlet
	testState.Cg, err = gauntlet.NewCosmosGauntlet(fmt.Sprintf("%s/", utils.ProjectRoot))
	require.NoError(t, err, "Could not get a new gauntlet struct")
	testState.DeployCluster()
	require.NoError(t, err, "Deploying cluster should not fail")
	if testState.Common.Env.WillUseRemoteRunner() {
		return // short circuit here if using a remote runner
	}
	err = actions.TeardownSuite(testState.T, testState.Common.Env, "./", nil, nil, zapcore.DPanicLevel, nil)
	require.NoError(testState.T, err)
}
