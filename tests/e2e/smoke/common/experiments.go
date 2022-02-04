package common

import (
	"fmt"
	"time"

	. "github.com/onsi/gomega"
	"github.com/smartcontractkit/helmenv/chaos/experiments"
)

// LabelChaosGroups sets labels for chaos groups
func (m *OCRv2State) LabelChaosGroups() {
	m.LabelChaosGroup(0, 0, ChaosGroupBootstrap)
	m.LabelChaosGroup(1, 4, ChaosGroupOracles)
	m.LabelChaosGroup(1, 3, ChaosGroupOraclesMinusOne)
	m.LabelChaosGroup(0, 1, ChaosGroupFaulty)
	m.LabelChaosGroup(3, 4, ChaosGroupOnline)
	m.LabelChaosGroup(0, 2, ChaosGroupYellow)
	m.LabelChaosGroup(0, 2, ChaosGroupLeftHalf)
	m.LabelChaosGroup(3, 4, ChaosGroupRightHalf)
}

// LabelChaosGroup sets label for a chaos group
func (m *OCRv2State) LabelChaosGroup(startInstance int, endInstance int, group string) {
	for i := startInstance; i <= endInstance; i++ {
		m.Err = m.Env.AddLabel(fmt.Sprintf("instance=%d,app=chainlink-node", i), fmt.Sprintf("%s=1", group))
		Expect(m.Err).ShouldNot(HaveOccurred())
	}
}

func (m *OCRv2State) CanRecoverAllNodesValidatorConnectionLoss() {
	// nolint
	defer m.Env.ClearAllChaosExperiments()
	_, err := m.Env.ApplyChaosExperiment(
		&experiments.NetworkPartition{
			FromMode:       "all",
			FromLabelKey:   ChaosGroupOnline,
			FromLabelValue: "1",
			ToMode:         "all",
			ToLabelKey:     "app",
			ToLabelValue:   "fcd-api",
		},
	)
	Expect(err).ShouldNot(HaveOccurred())
	time.Sleep(ChaosAwaitingApply)
	err = m.Env.ClearAllChaosExperiments()
	Expect(err).ShouldNot(HaveOccurred())
	m.ValidateRoundsAfter(time.Now(), 10)
}

func (m *OCRv2State) CanWorkYellowGroupNoValidatorConnection() {
	// nolint
	defer m.Env.ClearAllChaosExperiments()
	_, err := m.Env.ApplyChaosExperiment(
		&experiments.NetworkPartition{
			FromMode:       "all",
			FromLabelKey:   ChaosGroupYellow,
			FromLabelValue: "1",
			ToMode:         "all",
			ToLabelKey:     "app",
			ToLabelValue:   "fcd-api",
		},
	)
	Expect(err).ShouldNot(HaveOccurred())
	time.Sleep(ChaosAwaitingApply)
	m.ValidateRoundsAfter(time.Now(), 10)
}

func (m *OCRv2State) CantWorkWithFaultyNodesFailed() {
	// nolint
	defer m.Env.ClearAllChaosExperiments()
	_, err := m.Env.ApplyChaosExperiment(
		&experiments.PodFailure{
			Mode:       "all",
			LabelKey:   ChaosGroupYellow,
			LabelValue: "1",
			Duration:   UntilStop,
		},
	)
	Expect(err).ShouldNot(HaveOccurred())
	time.Sleep(ChaosAwaitingApply)
	m.ValidateNoRoundsAfter(time.Now())
}

func (m *OCRv2State) CanWorkWithFaultyNodesOffline() {
	// nolint
	defer m.Env.ClearAllChaosExperiments()
	_, err := m.Env.ApplyChaosExperiment(
		&experiments.NetworkPartition{
			FromMode:       "all",
			FromLabelKey:   ChaosGroupFaulty,
			FromLabelValue: "1",
			ToMode:         "all",
			ToLabelKey:     ChaosGroupOnline,
			ToLabelValue:   "1",
		},
	)
	Expect(err).ShouldNot(HaveOccurred())
	time.Sleep(ChaosAwaitingApply)
	m.ValidateRoundsAfter(time.Now(), 10)
}

func (m *OCRv2State) CantWorkWithMoreThanFaultyNodesOffline() {
	// nolint
	defer m.Env.ClearAllChaosExperiments()
	_, err := m.Env.ApplyChaosExperiment(
		&experiments.NetworkLoss{
			Mode:        "all",
			LabelKey:    ChaosGroupYellow,
			Loss:        100,
			Correlation: 100,
			LabelValue:  "1",
			Duration:    UntilStop,
		},
	)
	Expect(err).ShouldNot(HaveOccurred())
	time.Sleep(ChaosAwaitingApply)
	m.ValidateRoundsAfter(time.Now(), 10)
}

func (m *OCRv2State) NetworkCorrupt(group string, corrupt int, rounds int) {
	// nolint
	defer m.Env.ClearAllChaosExperiments()
	_, err := m.Env.ApplyChaosExperiment(
		&experiments.NetworkCorrupt{
			Mode:        "all",
			LabelKey:    group,
			LabelValue:  "1",
			Corrupt:     corrupt,
			Correlation: 100,
			Duration:    UntilStop,
		},
	)
	Expect(err).ShouldNot(HaveOccurred())
	time.Sleep(ChaosAwaitingApply)
	m.ValidateRoundsAfter(time.Now(), rounds)
}

func (m *OCRv2State) CanWorkAfterAllOraclesIPChange() {
	// nolint
	defer m.Env.ClearAllChaosExperiments()
	_, err := m.Env.ApplyChaosExperiment(
		&experiments.PodKill{
			Mode:       "all",
			LabelKey:   ChaosGroupOracles,
			LabelValue: "1",
		},
	)
	Expect(err).ShouldNot(HaveOccurred())
	time.Sleep(ChaosAwaitingApply)
	m.ValidateRoundsAfter(time.Now(), 10)
}

func (m *OCRv2State) CanMigrateBootstrap() {
	// nolint
	defer m.Env.ClearAllChaosExperiments()
	_, err := m.Env.ApplyChaosExperiment(
		&experiments.PodFailure{
			Mode:       "all",
			LabelKey:   ChaosGroupBootstrap,
			LabelValue: "1",
			Duration:   UntilStop,
		},
	)
	Expect(err).ShouldNot(HaveOccurred())
	time.Sleep(ChaosAwaitingApply)
	m.ValidateRoundsAfter(time.Now(), 10)
	// now we working without bootstrap, killing all oracles except one, remaining one must bootstrap
	_, err = m.Env.ApplyChaosExperiment(
		&experiments.PodKill{
			Mode:       "all",
			LabelKey:   ChaosGroupOraclesMinusOne,
			LabelValue: "1",
		},
	)
	Expect(err).ShouldNot(HaveOccurred())
	time.Sleep(ChaosAwaitingApply)
	m.ValidateRoundsAfter(time.Now(), 10)
}

func (m *OCRv2State) RestoredAfterNetworkSplit() {
	// nolint
	defer m.Env.ClearAllChaosExperiments()
	_, err := m.Env.ApplyChaosExperiment(
		&experiments.NetworkPartition{
			FromMode:       "all",
			FromLabelKey:   ChaosGroupLeftHalf,
			FromLabelValue: "1",
			ToMode:         "all",
			ToLabelKey:     ChaosGroupRightHalf,
			ToLabelValue:   "1",
		},
	)
	Expect(err).ShouldNot(HaveOccurred())
	time.Sleep(ChaosAwaitingApply)
	m.ValidateNoRoundsAfter(time.Now())
	err = m.Env.ClearAllChaosExperiments()
	Expect(err).ShouldNot(HaveOccurred())
	m.ValidateRoundsAfter(time.Now(), 10)
}

func (m *OCRv2State) CanWorkWithTimeSkewYellowGroup() {
	// target pod linux must have method https://man7.org/linux/man-pages/man2/clock_gettime.2.html in order to work
	// nolint
	defer m.Env.ClearAllChaosExperiments()
	_, err := m.Env.ApplyChaosExperiment(
		&experiments.TimeShift{
			Mode:       "all",
			LabelKey:   ChaosGroupYellow,
			LabelValue: "1",
			TimeOffset: -20 * time.Hour,
			Duration:   UntilStop,
		},
	)
	Expect(err).ShouldNot(HaveOccurred())
	time.Sleep(ChaosAwaitingApply)
	m.ValidateRoundsAfter(time.Now(), 10)
}
